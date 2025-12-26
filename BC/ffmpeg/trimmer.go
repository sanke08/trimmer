package ffmpeg

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sanke08/videoprocessor/models"
	"github.com/sanke08/videoprocessor/utils"
)

// TrimSegmentWithMetadata trims a video segment while preserving all streams and metadata
// Returns the final trimmed file path and the shifted metadata path
func TrimSegmentWithMetadata(file string, outputDir string, start, end float64) (string, string, error) {
	// prepare filenames
	tempDir, err := os.MkdirTemp(outputDir, "tmp_trim_*")
	if err != nil {
		return "", "", fmt.Errorf("failed create temp dir: %v", err)
	}
	// paths
	origMeta := filepath.Join(tempDir, "orig_meta.txt")
	shiftedMeta := filepath.Join(outputDir, fmt.Sprintf("%s_meta_%d.txt", strings.TrimSuffix(filepath.Base(file), filepath.Ext(file)), time.Now().UnixNano()))
	tempTrim := filepath.Join(tempDir, "temp_trim.mkv")
	finalOut := utils.MakeTrimFilename(outputDir, file, start, end)

	// 1. extract metadata from original
	if err := ExtractMetadata(file, origMeta); err != nil {
		// if metadata extraction fails, continue but we won't be able to apply chapters
		log.Printf("⚠️ metadata extract failed for %s: %v", file, err)
		// still proceed but without metadata
	}

	// 2. create shifted metadata if origMeta exists
	if _, err := os.Stat(origMeta); err == nil {
		if err := CreateShiftedMetadata(origMeta, shiftedMeta, start, end); err != nil {
			log.Printf("⚠️ create shifted metadata failed: %v", err)
			// proceed without applying metadata
			shiftedMeta = ""
		}
	} else {
		shiftedMeta = ""
	}

	// 3. trim without copying chapters (we will reapply them)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	args := []string{
		"-y",
		"-ss", fmt.Sprintf("%.3f", start), // Place -ss BEFORE -i for faster, accurate keyframe seeking
		"-i", file,
		"-to", fmt.Sprintf("%.3f", end),
		"-map", "0",
		"-ignore_unknown",
		"-c", "copy",
		"-copyts", // Copy timestamps to maintain accuracy
		"-avoid_negative_ts", "make_zero",
		"-map_chapters", "-1",
		tempTrim,
	}
	out, err := RunCmd(ctx, "ffmpeg", args...)
	if err != nil {
		// cleanup
		_ = os.Remove(tempTrim)
		return "", "", fmt.Errorf("ffmpeg trim failed: %v (%s)", err, string(out))
	}

	// 4. reapply metadata if shiftedMeta exists
	if shiftedMeta != "" {
		ctx2, cancel2 := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel2()
		// ffmpeg -y -i tempTrim -i shiftedMeta -map 0 -map_metadata 1 -c copy finalOut
		out2, err2 := RunCmd(ctx2, "ffmpeg", "-y", "-i", tempTrim, "-i", shiftedMeta, "-map", "0", "-ignore_unknown", "-map_metadata", "1", "-c", "copy", finalOut)
		if err2 != nil {
			// fallback: rename tempTrim to finalOut
			_ = os.Rename(tempTrim, finalOut)
			log.Printf("⚠️ ffmpeg reapply metadata failed: %v (%s). Using trimmed file without metadata.", err2, string(out2))
			// return finalOut and shiftedMeta (maybe partially useful)
			_ = os.Remove(tempTrim)
			// cleanup origMeta
			_ = os.Remove(origMeta)
			return finalOut, shiftedMeta, nil
		}
		// success -> remove tempTrim
		_ = os.Remove(tempTrim)
		_ = os.Remove(origMeta)
		return finalOut, shiftedMeta, nil
	}

	// no metadata to apply -> move tempTrim to finalOut
	if err := os.Rename(tempTrim, finalOut); err != nil {
		// fallback to copy
		input, err2 := os.ReadFile(tempTrim)
		if err2 == nil {
			_ = os.WriteFile(finalOut, input, 0644)
			_ = os.Remove(tempTrim)
		} else {
			return "", "", fmt.Errorf("failed to move trimmed file: %v", err)
		}
	}
	_ = os.Remove(origMeta)
	return finalOut, "", nil
}

// ComputeKeepSegments calculates which segments to keep based on skip ranges
func ComputeKeepSegments(ch models.Chapters, skips []models.SkipRange) []struct{ Start, End float64 } {
	end := ch["End"]
	segments := []struct{ Start, End float64 }{{0, end}}

	for _, skip := range skips {
		s, okS := ch[skip.Start]
		e, okE := ch[skip.End]
		if !okS || !okE || e <= s {
			continue
		}

		newSegments := []struct{ Start, End float64 }{}
		for _, seg := range segments {
			if seg.End <= s || seg.Start >= e {
				newSegments = append(newSegments, seg)
				continue
			}
			if seg.Start < s {
				newSegments = append(newSegments, struct{ Start, End float64 }{seg.Start, s})
			}
			if seg.End > e {
				newSegments = append(newSegments, struct{ Start, End float64 }{e, seg.End})
			}
		}
		segments = newSegments
	}

	if len(segments) == 0 {
		segments = append(segments, struct{ Start, End float64 }{0, end})
	}
	return segments
}
