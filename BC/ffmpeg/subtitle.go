package ffmpeg

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// SubtitleTrack represents a subtitle stream
type SubtitleTrack struct {
	Index    int    `json:"index"`
	Language string `json:"language"`
	Title    string `json:"title"`
	Codec    string `json:"codec_name"`
}

// ScanSubtitles scans all subtitle tracks in a video file
func ScanSubtitles(file string) ([]SubtitleTrack, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "ffprobe",
		"-v", "error",
		"-select_streams", "s",
		"-show_entries", "stream=index,codec_name:stream_tags=language,title",
		"-of", "json", file)

	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("ffprobe subtitle scan failed: %v", err)
	}

	var data struct {
		Streams []struct {
			Index     int    `json:"index"`
			CodecName string `json:"codec_name"`
			Tags      struct {
				Language string `json:"language"`
				Title    string `json:"title"`
			} `json:"tags"`
		} `json:"streams"`
	}

	if err := json.Unmarshal(out, &data); err != nil {
		return nil, fmt.Errorf("json unmarshal failed: %v", err)
	}

	var tracks []SubtitleTrack
	for _, s := range data.Streams {
		tracks = append(tracks, SubtitleTrack{
			Index:    s.Index,
			Language: s.Tags.Language,
			Title:    s.Tags.Title,
			Codec:    s.CodecName,
		})
	}

	return tracks, nil
}

// ExtractSubtitle extracts a specific subtitle track to a file
func ExtractSubtitle(inputFile string, trackIndex int, outputFile string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// Determine output format based on extension
	ext := strings.ToLower(filepath.Ext(outputFile))
	var codec string
	switch ext {
	case ".srt":
		codec = "srt"
	case ".ass", ".ssa":
		codec = "ass"
	case ".vtt":
		codec = "webvtt"
	default:
		codec = "srt" // default to SRT
		outputFile = strings.TrimSuffix(outputFile, ext) + ".srt"
	}

	args := []string{
		"-y",
		"-i", inputFile,
		"-map", fmt.Sprintf("0:%d", trackIndex),
		"-c:s", codec,
		outputFile,
	}

	out, err := RunCmd(ctx, "ffmpeg", args...)
	if err != nil {
		return fmt.Errorf("subtitle extraction failed: %v (%s)", err, string(out))
	}

	return nil
}

// AdjustSubtitleTiming adjusts subtitle timing by shifting timestamps
func AdjustSubtitleTiming(inputSub, outputSub string, offsetSeconds float64) error {
	// Read subtitle file
	content, err := os.ReadFile(inputSub)
	if err != nil {
		return fmt.Errorf("failed to read subtitle: %v", err)
	}

	ext := strings.ToLower(filepath.Ext(inputSub))

	if ext == ".srt" {
		return adjustSRTTiming(string(content), outputSub, offsetSeconds)
	} else if ext == ".ass" || ext == ".ssa" {
		return adjustASSTiming(string(content), outputSub, offsetSeconds)
	}

	// For other formats, use ffmpeg to shift
	return shiftSubtitleWithFFmpeg(inputSub, outputSub, offsetSeconds)
}

func adjustSRTTiming(content, outputFile string, offset float64) error {
	lines := strings.Split(content, "\n")
	var result []string

	for _, line := range lines {
		// Check if line contains timestamp (format: 00:00:00,000 --> 00:00:00,000)
		if strings.Contains(line, " --> ") {
			parts := strings.Split(line, " --> ")
			if len(parts) == 2 {
				start := shiftSRTTimestamp(strings.TrimSpace(parts[0]), offset)
				end := shiftSRTTimestamp(strings.TrimSpace(parts[1]), offset)
				line = fmt.Sprintf("%s --> %s", start, end)
			}
		}
		result = append(result, line)
	}

	return os.WriteFile(outputFile, []byte(strings.Join(result, "\n")), 0644)
}

func shiftSRTTimestamp(timestamp string, offset float64) string {
	// Parse SRT timestamp: HH:MM:SS,mmm
	parts := strings.Split(timestamp, ",")
	if len(parts) != 2 {
		return timestamp
	}

	timeParts := strings.Split(parts[0], ":")
	if len(timeParts) != 3 {
		return timestamp
	}

	hours, _ := strconv.Atoi(timeParts[0])
	minutes, _ := strconv.Atoi(timeParts[1])
	seconds, _ := strconv.Atoi(timeParts[2])
	millis, _ := strconv.Atoi(parts[1])

	totalSeconds := float64(hours*3600+minutes*60+seconds) + float64(millis)/1000.0
	totalSeconds += offset

	if totalSeconds < 0 {
		totalSeconds = 0
	}

	newHours := int(totalSeconds) / 3600
	newMinutes := (int(totalSeconds) % 3600) / 60
	newSeconds := int(totalSeconds) % 60
	newMillis := int((totalSeconds - float64(int(totalSeconds))) * 1000)

	return fmt.Sprintf("%02d:%02d:%02d,%03d", newHours, newMinutes, newSeconds, newMillis)
}

func adjustASSTiming(content, outputFile string, offset float64) error {
	lines := strings.Split(content, "\n")
	var result []string

	for _, line := range lines {
		// ASS format: Dialogue: layer,start,end,style,name,marginL,marginR,marginV,effect,text
		if strings.HasPrefix(line, "Dialogue:") {
			parts := strings.SplitN(line, ",", 10)
			if len(parts) >= 3 {
				parts[1] = shiftASSTimestamp(strings.TrimSpace(parts[1]), offset)
				parts[2] = shiftASSTimestamp(strings.TrimSpace(parts[2]), offset)
				line = strings.Join(parts, ",")
			}
		}
		result = append(result, line)
	}

	return os.WriteFile(outputFile, []byte(strings.Join(result, "\n")), 0644)
}

func shiftASSTimestamp(timestamp string, offset float64) string {
	// Parse ASS timestamp: H:MM:SS.cc
	parts := strings.Split(timestamp, ":")
	if len(parts) != 3 {
		return timestamp
	}

	hours, _ := strconv.Atoi(parts[0])
	minutes, _ := strconv.Atoi(parts[1])
	secParts := strings.Split(parts[2], ".")
	seconds, _ := strconv.Atoi(secParts[0])
	centisec := 0
	if len(secParts) > 1 {
		centisec, _ = strconv.Atoi(secParts[1])
	}

	totalSeconds := float64(hours*3600+minutes*60+seconds) + float64(centisec)/100.0
	totalSeconds += offset

	if totalSeconds < 0 {
		totalSeconds = 0
	}

	newHours := int(totalSeconds) / 3600
	newMinutes := (int(totalSeconds) % 3600) / 60
	newSeconds := int(totalSeconds) % 60
	newCentisec := int((totalSeconds - float64(int(totalSeconds))) * 100)

	return fmt.Sprintf("%d:%02d:%02d.%02d", newHours, newMinutes, newSeconds, newCentisec)
}

func shiftSubtitleWithFFmpeg(inputSub, outputSub string, offset float64) error {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	offsetStr := fmt.Sprintf("%.3f", offset)

	args := []string{
		"-y",
		"-itsoffset", offsetStr,
		"-i", inputSub,
		"-c", "copy",
		outputSub,
	}

	out, err := RunCmd(ctx, "ffmpeg", args...)
	if err != nil {
		log.Printf("⚠️ FFmpeg subtitle shift failed, copying original: %v", err)
		// Fallback: just copy the file
		content, err2 := os.ReadFile(inputSub)
		if err2 == nil {
			return os.WriteFile(outputSub, content, 0644)
		}
		return fmt.Errorf("subtitle timing adjustment failed: %v (%s)", err, string(out))
	}

	return nil
}

// ExtractAllSubtitles extracts all subtitle tracks from a video file
func ExtractAllSubtitles(videoFile, outputDir string) (map[int]string, error) {
	tracks, err := ScanSubtitles(videoFile)
	if err != nil {
		return nil, err
	}

	if len(tracks) == 0 {
		log.Printf("No subtitle tracks found in %s", videoFile)
		return make(map[int]string), nil
	}

	// Create subtitles subfolder
	subsDir := filepath.Join(outputDir, "subtitles")
	os.MkdirAll(subsDir, 0755)

	extractedFiles := make(map[int]string)
	baseName := strings.TrimSuffix(filepath.Base(videoFile), filepath.Ext(videoFile))

	for _, track := range tracks {
		lang := track.Language
		if lang == "" {
			lang = "unknown"
		}

		subFile := filepath.Join(subsDir, fmt.Sprintf("%s_%s_%d.srt", baseName, lang, track.Index))

		err := ExtractSubtitle(videoFile, track.Index, subFile)
		if err != nil {
			log.Printf("⚠️ Failed to extract subtitle track %d: %v", track.Index, err)
			continue
		}

		extractedFiles[track.Index] = subFile
		log.Printf("✅ Extracted subtitle: %s (track %d, %s)", filepath.Base(subFile), track.Index, lang)
	}

	return extractedFiles, nil
}
