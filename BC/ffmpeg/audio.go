package ffmpeg

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// AudioTrackInfo represents an audio stream
type AudioTrackInfo struct {
	Index    int    `json:"index"`
	Language string `json:"language"`
	Title    string `json:"title"`
	Codec    string `json:"codec_name"`
	Channels int    `json:"channels"`
}

// ScanAudioTracks scans all audio tracks in a video file
func ScanAudioTracks(file string) ([]AudioTrackInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "ffprobe",
		"-v", "error",
		"-select_streams", "a",
		"-show_entries", "stream=index,codec_name,channels:stream_tags=language,title",
		"-of", "json", file)

	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("ffprobe audio scan failed: %v", err)
	}

	var data struct {
		Streams []struct {
			Index     int    `json:"index"`
			CodecName string `json:"codec_name"`
			Channels  int    `json:"channels"`
			Tags      struct {
				Language string `json:"language"`
				Title    string `json:"title"`
			} `json:"tags"`
		} `json:"streams"`
	}

	if err := json.Unmarshal(out, &data); err != nil {
		return nil, fmt.Errorf("json unmarshal failed: %v", err)
	}

	var tracks []AudioTrackInfo
	for _, s := range data.Streams {
		tracks = append(tracks, AudioTrackInfo{
			Index:    s.Index,
			Language: s.Tags.Language,
			Title:    s.Tags.Title,
			Codec:    s.CodecName,
			Channels: s.Channels,
		})
	}

	return tracks, nil
}

// ExtractAudio extracts a specific audio track to a file
func ExtractAudio(inputFile string, trackIndex int, outputFile string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Determine output format based on extension or default to AAC
	ext := strings.ToLower(filepath.Ext(outputFile))
	var codec string
	switch ext {
	case ".aac":
		codec = "aac"
	case ".mp3":
		codec = "libmp3lame"
	case ".opus":
		codec = "libopus"
	case ".flac":
		codec = "flac"
	case ".mka":
		codec = "copy" // Keep original codec
	default:
		codec = "copy"
		outputFile = strings.TrimSuffix(outputFile, ext) + ".mka"
	}

	args := []string{
		"-y",
		"-i", inputFile,
		"-map", fmt.Sprintf("0:%d", trackIndex),
		"-vn", // No video
	}

	if codec == "copy" {
		args = append(args, "-c", "copy")
	} else {
		args = append(args, "-c:a", codec)
	}

	args = append(args, outputFile)

	out, err := RunCmd(ctx, "ffmpeg", args...)
	if err != nil {
		return fmt.Errorf("audio extraction failed: %v (%s)", err, string(out))
	}

	return nil
}

// ExtractAllAudioTracks extracts all audio tracks from a video file
func ExtractAllAudioTracks(videoFile, outputDir string) (map[int]string, error) {
	tracks, err := ScanAudioTracks(videoFile)
	if err != nil {
		return nil, err
	}

	if len(tracks) == 0 {
		log.Printf("No audio tracks found in %s", videoFile)
		return make(map[int]string), nil
	}

	// Create audios subfolder
	audiosDir := filepath.Join(outputDir, "audios")
	os.MkdirAll(audiosDir, 0755)

	extractedFiles := make(map[int]string)
	baseName := strings.TrimSuffix(filepath.Base(videoFile), filepath.Ext(videoFile))

	for _, track := range tracks {
		lang := track.Language
		if lang == "" {
			lang = "unknown"
		}

		title := track.Title
		if title == "" {
			title = track.Codec
		}
		// Clean title for filename
		title = strings.ReplaceAll(title, " ", "_")
		title = strings.ReplaceAll(title, "/", "_")

		audioFile := filepath.Join(audiosDir, fmt.Sprintf("%s_%s_%s_track%d.mka", baseName, lang, title, track.Index))

		err := ExtractAudio(videoFile, track.Index, audioFile)
		if err != nil {
			log.Printf("⚠️ Failed to extract audio track %d: %v", track.Index, err)
			continue
		}

		extractedFiles[track.Index] = audioFile
		log.Printf("✅ Extracted audio: %s (track %d, %s, %dch)", filepath.Base(audioFile), track.Index, lang, track.Channels)
	}

	return extractedFiles, nil
}
