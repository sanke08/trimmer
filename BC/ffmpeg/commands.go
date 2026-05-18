package ffmpeg

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
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

// GetVideoResolution returns the width and height of the first video stream
func GetVideoResolution(path string) (int, int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	out, err := RunCmd(ctx, "ffprobe", "-v", "error", "-select_streams", "v:0",
		"-show_entries", "stream=width,height", "-of", "csv=s=x:p=0", path)
	if err != nil {
		return 0, 0, fmt.Errorf("ffprobe resolution failed: %v (%s)", err, string(out))
	}
	parts := strings.SplitN(strings.TrimSpace(string(out)), "x", 2)
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("unexpected resolution output: %s", string(out))
	}
	w, err1 := strconv.Atoi(parts[0])
	h, err2 := strconv.Atoi(parts[1])
	if err1 != nil || err2 != nil {
		return 0, 0, fmt.Errorf("parse resolution failed: %s", string(out))
	}
	return w, h, nil
}

// GetAudioStreamCount returns the number of audio streams in a file
func GetAudioStreamCount(path string) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	out, err := RunCmd(ctx, "ffprobe", "-v", "error", "-select_streams", "a",
		"-show_entries", "stream=index", "-of", "csv=p=0", path)
	if err != nil {
		return 0, fmt.Errorf("ffprobe audio streams failed: %v (%s)", err, string(out))
	}
	lines := 0
	for _, l := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if strings.TrimSpace(l) != "" {
			lines++
		}
	}
	return lines, nil
}

// GetVideoCodecAndPixFmt returns the codec name and pix_fmt of the first video stream
func GetVideoCodecAndPixFmt(path string) (codec string, pixFmt string) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	out, _ := RunCmd(ctx, "ffprobe", "-v", "error", "-select_streams", "v:0",
		"-show_entries", "stream=codec_name,pix_fmt", "-of", "csv=p=0", path)
	parts := strings.SplitN(strings.TrimSpace(string(out)), ",", 2)
	if len(parts) == 2 {
		return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
	}
	return "h264", "yuv420p"
}

// NormalizeStreams re-encodes src to match targetW x targetH resolution and targetAudioCount audio streams.
// Missing audio streams are filled with silent tracks. Encoder and pixel format are matched to the target file's codec.
func NormalizeStreams(src, dst string, targetW, targetH, targetAudioCount int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Minute)
	defer cancel()

	srcW, srcH, _ := GetVideoResolution(src)
	srcAudio, _ := GetAudioStreamCount(src)

	args := []string{"-y", "-i", src}

	// add silent audio inputs for missing tracks
	silentInputIdx := 1
	for i := srcAudio; i < targetAudioCount; i++ {
		args = append(args, "-f", "lavfi", "-i", "anullsrc=r=48000:cl=stereo")
		silentInputIdx++
	}

	// video: scale if resolution differs, choosing encoder to match source codec
	if srcW != targetW || srcH != targetH {
		srcCodec, srcPixFmt := GetVideoCodecAndPixFmt(src)
		scale := fmt.Sprintf("scale=%d:%d:flags=lanczos,setsar=1", targetW, targetH)

		var encoder, preset string
		switch srcCodec {
		case "hevc", "h265":
			encoder = "libx265"
			preset = "ultrafast"
		default:
			encoder = "libx264"
			preset = "fast"
		}
		args = append(args, "-vf", scale, "-c:v", encoder, "-preset", preset, "-crf", "18", "-pix_fmt", srcPixFmt)
	} else {
		args = append(args, "-c:v", "copy")
	}

	// map video
	args = append(args, "-map", "0:v?")

	// map existing audio streams
	for i := 0; i < srcAudio && i < targetAudioCount; i++ {
		args = append(args, "-map", fmt.Sprintf("0:a:%d", i))
	}
	// map silent tracks for missing audio
	for i := srcAudio; i < targetAudioCount; i++ {
		silentStream := i - srcAudio + 1
		args = append(args, "-map", fmt.Sprintf("%d:a", silentStream))
	}

	args = append(args, "-c:a", "copy", "-ignore_unknown", "-avoid_negative_ts", "make_zero", dst)

	out, err := RunCmd(ctx, "ffmpeg", args...)
	if err != nil {
		return fmt.Errorf("normalize streams failed: %v (%s)", err, string(out))
	}
	return nil
}

// GetStartTime returns the PTS of the first video packet in seconds.
// Used to detect the real keyframe cut point after a stream-copy trim.
func GetStartTime(path string) (float64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	out, err := RunCmd(ctx, "ffprobe", "-v", "error", "-select_streams", "v:0",
		"-show_entries", "packet=pts_time",
		"-read_intervals", "%+#1",
		"-of", "csv=p=0", path)
	if err != nil {
		return 0, fmt.Errorf("ffprobe start time failed: %v (%s)", err, string(out))
	}
	s := strings.TrimSpace(string(out))
	if s == "" {
		return 0, fmt.Errorf("empty start time")
	}
	return strconv.ParseFloat(s, 64)
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
	folder = strings.TrimSpace(folder)

	mkvFiles, err := filepath.Glob(filepath.Join(folder, "*.mkv"))
	if err != nil || len(mkvFiles) < 1 {
		return nil, fmt.Errorf("no MKV files found in %s", folder)
	}

	chapters := make(models.Chapters)
	cumulativeTime := 0.0
	audioTracks := []models.AudioTrack{}

	sort.Slice(mkvFiles, func(i, j int) bool {
		return utils.NaturalLess(mkvFiles[i], mkvFiles[j])
	})

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
