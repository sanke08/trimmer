package ffmpeg

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/sanke08/videoprocessor/models"
	"github.com/sanke08/videoprocessor/utils"
)

// RunCmd executes a command with context and timeout
func RunCmd(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	// make windows hide window if available
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	return cmd.CombinedOutput()
}

// GetDuration gets the duration of a video file using ffprobe
func GetDuration(path string) (float64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	out, err := RunCmd(ctx, "ffprobe", "-v", "error", "-show_entries", "format=duration",
		"-of", "default=noprint_wrappers=1:nokey=1", path)
	if err != nil {
		return 0, fmt.Errorf("ffprobe duration failed: %v (%s)", err, string(out))
	}
	s := strings.TrimSpace(string(out))
	if s == "" {
		return 0, fmt.Errorf("empty duration")
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, err
	}
	return v, nil
}

// ExtractMetadata extracts ffmetadata from original file to outPath
func ExtractMetadata(input, outPath string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	// ffmpeg -y -i input -f ffmetadata outPath
	out, err := RunCmd(ctx, "ffmpeg", "-y", "-i", input, "-f", "ffmetadata", outPath)
	if err != nil {
		// return with output so caller can inspect
		return fmt.Errorf("ffmpeg extract metadata failed: %v (%s)", err, string(out))
	}
	return nil
}

// ScanChapters scans chapters from a single file
func ScanChapters(file string) (models.Chapters, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "ffprobe", "-v", "error", "-show_chapters", "-of", "json", file)
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	var data struct {
		Chapters []struct {
			StartTime string `json:"start_time"`
			Tags      struct {
				Title string `json:"title"`
			} `json:"tags"`
		} `json:"chapters"`
	}
	if err := json.Unmarshal(out, &data); err != nil {
		return nil, err
	}
	chapters := make(models.Chapters)
	for idx, ch := range data.Chapters {
		t, _ := strconv.ParseFloat(ch.StartTime, 64)
		title := ch.Tags.Title
		if title == "" {
			title = fmt.Sprintf("Chapter_%02d", idx+1)
		}
		chapters[title] = t
	}
	// ensure End exists
	cmdDur := exec.Command("ffprobe", "-v", "error", "-show_entries", "format=duration",
		"-of", "default=noprint_wrappers=1:nokey=1", file)
	durBytes, _ := cmdDur.Output()
	dur, _ := strconv.ParseFloat(strings.TrimSpace(string(durBytes)), 64)
	if dur <= 0 {
		maxT := 0.0
		for _, v := range chapters {
			if v > maxT {
				maxT = v
			}
		}
		dur = maxT + 1.0
	}
	chapters["End"] = dur
	return chapters, nil
}

// ScanFirstTwoEpisodes scans the first two episodes in a folder for analysis
func ScanFirstTwoEpisodes(folder string) (*models.ScanResult, error) {
	// Normalize path for Windows - keep backslashes for Windows commands
	folder = strings.TrimSpace(folder)
	folder = strings.ReplaceAll(folder, "/", "\\")

	// Use dir command to list .mkv files (Windows style with backslashes)
	pattern := folder + "\\*.mkv"
	files, err := exec.Command("cmd", "/c", "dir", "/b", pattern).Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list files in %s: %v", folder, err)
	}

	fileList := strings.Split(strings.TrimSpace(string(files)), "\n")
	var mkvFiles []string
	for _, f := range fileList {
		f = strings.TrimSpace(f)
		f = strings.ReplaceAll(f, "\r", "") // Remove carriage return
		if f != "" {
			mkvFiles = append(mkvFiles, folder+"\\"+f)
		}
	}

	if len(mkvFiles) < 2 {
		return nil, fmt.Errorf("less than 2 MKV files found in %s", folder)
	}

	chapters := make(models.Chapters)
	cumulativeTime := 0.0
	audioTracks := []models.AudioTrack{}

	sort.Strings(mkvFiles)

	for i := 0; i < utils.Min(2, len(mkvFiles)); i++ {
		file := mkvFiles[i]
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		cmd := exec.CommandContext(ctx, "ffprobe", "-v", "error", "-show_chapters", "-of", "json", file)
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		out, err := cmd.Output()
		cancel()
		if err != nil {
			return nil, fmt.Errorf("ffprobe error on %s: %v", file, err)
		}

		var data struct {
			Chapters []struct {
				StartTime string `json:"start_time"`
				Tags      struct {
					Title string `json:"title"`
				} `json:"tags"`
			} `json:"chapters"`
		}
		if err := json.Unmarshal(out, &data); err != nil {
			return nil, fmt.Errorf("json unmarshal error on %s: %v", file, err)
		}

		for _, ch := range data.Chapters {
			t, _ := strconv.ParseFloat(ch.StartTime, 64)
			title := ch.Tags.Title
			if title == "" {
				title = fmt.Sprintf("Chapter_%.0f", t)
			}
			chapters[title] = cumulativeTime + t
		}

		// Duration
		cmdDur := exec.Command("ffprobe", "-v", "error", "-show_entries", "format=duration",
			"-of", "default=noprint_wrappers=1:nokey=1", file)
		durBytes, _ := cmdDur.Output()
		dur, _ := strconv.ParseFloat(strings.TrimSpace(string(durBytes)), 64)
		cumulativeTime += dur

		if i == 0 {
			// Audio tracks of first file
			ctxAudio, cancelAudio := context.WithTimeout(context.Background(), 30*time.Second)
			cmdAudio := exec.CommandContext(ctxAudio, "ffprobe",
				"-v", "error", "-select_streams", "a",
				"-show_entries", "stream=index:stream_tags=language,title",
				"-of", "json", file)
			outAudio, err := cmdAudio.Output()
			cancelAudio()
			if err == nil {
				var audioData struct {
					Streams []struct {
						Index int `json:"index"`
						Tags  struct {
							Language string `json:"language"`
							Title    string `json:"title"`
						} `json:"tags"`
					} `json:"streams"`
				}
				if err := json.Unmarshal(outAudio, &audioData); err == nil {
					for _, s := range audioData.Streams {
						audioTracks = append(audioTracks, models.AudioTrack{
							Index: s.Index - 1,
							Lang:  s.Tags.Language,
							Title: s.Tags.Title,
						})
					}
				}
			}
		}
	}

	chapters["End"] = cumulativeTime
	return &models.ScanResult{
		Chapters:    chapters,
		AudioTracks: audioTracks,
		FirstFile:   mkvFiles[0],
	}, nil
}
