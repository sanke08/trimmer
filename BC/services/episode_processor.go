package services

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sanke08/videoprocessor/ffmpeg"
	"github.com/sanke08/videoprocessor/models"
	"github.com/sanke08/videoprocessor/utils"
)

// ProcessSingleEpisode processes a single episode with trimming and metadata preservation
func ProcessSingleEpisode(file string, output string, ch models.Chapters, opts models.TrimOptions) (string, string, float64, error) {
	log.Printf("📼 Processing: %s", filepath.Base(file))

	// compute segments
	segmentsData := ffmpeg.ComputeKeepSegments(ch, opts.SkipRanges)
	if len(segmentsData) == 0 {
		return "", "", 0, fmt.Errorf("no segments to keep for %s", file)
	}

	// If more than one segment per episode: trim each segment with metadata and merge segments into one episode file
	trimmedParts := []string{}
	trimmedMetaFiles := []string{}
	totalDur := 0.0

	for i, seg := range segmentsData {
		if seg.End <= seg.Start {
			continue
		}
		// TrimSegmentWithMetadata already uses -map 0 which preserves ALL streams (video, audio, subs)
		trimFile, metaFile, err := ffmpeg.TrimSegmentWithMetadata(file, output, seg.Start, seg.End)
		if err != nil {
			log.Printf("⚠️ Trim part %d failed for %s: %v", i, file, err)
			// continue to next segment
			continue
		}
		trimmedParts = append(trimmedParts, trimFile)
		trimmedMetaFiles = append(trimmedMetaFiles, metaFile)
		dur, _ := ffmpeg.GetDuration(trimFile)
		totalDur += dur
	}

	if len(trimmedParts) == 0 {
		return "", "", 0, fmt.Errorf("no valid segments created for %s", file)
	}

	var finalFile string
	if len(trimmedParts) == 1 {
		// single piece: its metadata file is trimmedMetaFiles[0] (may be empty)
		finalFile = trimmedParts[0]
	} else {
		// multiple pieces -> concat them into a single episode (preserve ALL streams)
		listFile := filepath.Join(output, fmt.Sprintf("concat_list_%d.txt", time.Now().UnixNano()))
		f, err := os.Create(listFile)
		if err != nil {
			return "", "", 0, err
		}
		for _, p := range trimmedParts {
			abs, _ := filepath.Abs(p)
			_, _ = f.WriteString(fmt.Sprintf("file '%s'\n", utils.EscapeForFFmpeg(abs)))
		}
		f.Close()

		mergedEpisode := filepath.Join(output, fmt.Sprintf("merged_%s_%d.mkv", strings.TrimSuffix(filepath.Base(file), filepath.Ext(file)), time.Now().UnixNano()))
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()
		// Use -map 0 to copy ALL streams (video, all audio, all subtitles)
		outb, err := ffmpeg.RunCmd(ctx, "ffmpeg", "-y", "-f", "concat", "-safe", "0", "-i", listFile, "-map", "0", "-ignore_unknown", "-c", "copy", mergedEpisode)
		os.Remove(listFile)
		if err != nil {
			return "", "", 0, fmt.Errorf("ffmpeg concat episode parts failed: %v (%s)", err, string(outb))
		}

		// clean up individual trimmedParts
		for _, p := range trimmedParts {
			// conservative remove
			if strings.Contains(strings.ToLower(p), "tmp") || strings.Contains(strings.ToLower(p), "_seg_") {
				_ = os.Remove(p)
			}
		}
		finalFile = mergedEpisode
	}

	metaFile := ""
	if len(trimmedMetaFiles) > 0 {
		metaFile = trimmedMetaFiles[0]
	}

	return finalFile, metaFile, totalDur, nil
}
