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

// TrimSegmentWithMetadata trims a video segment while preserving all streams and metadata.
// If norm is non-nil and the file's resolution differs from norm, the video is scaled inline.
// Returns the final trimmed file path and the shifted metadata path.
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
		log.Printf("⚠️ metadata extract failed for %s: %v", file, err)
	}

	// 2. trim WITHOUT timestamp normalisation so we can probe the real keyframe cut point
	timeout := 5 * time.Minute
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	args := []string{
		"-y",
		"-ss", fmt.Sprintf("%.3f", start),
		"-i", file,
		"-to", fmt.Sprintf("%.3f", end),
		"-map", "0:v?",
		"-map", "0:a?",
		"-ignore_unknown",
		"-c", "copy",
		"-copyts",
		// no -avoid_negative_ts here; we need original timestamps to probe actual cut point
		"-map_chapters", "-1",
		tempTrim,
	}

	out, err := RunCmd(ctx, "ffmpeg", args...)
	if err != nil {
		_ = os.Remove(tempTrim)
		return "", "", fmt.Errorf("ffmpeg trim failed: %v (%s)", err, string(out))
	}

	// 3. probe the actual first-frame PTS so chapter shifts account for keyframe alignment
	actualStart := start
	if probed, perr := GetStartTime(tempTrim); perr == nil && probed >= 0 {
		actualStart = probed
	}

	// 4. build shifted metadata using the real cut point
	if _, err := os.Stat(origMeta); err == nil {
		if err := CreateShiftedMetadata(origMeta, shiftedMeta, actualStart, end); err != nil {
			log.Printf("⚠️ create shifted metadata failed: %v", err)
			shiftedMeta = ""
		}
	} else {
		shiftedMeta = ""
	}

	// 5. apply metadata + normalise timestamps into finalOut
	ctx2, cancel2 := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel2()

	var applyArgs []string
	if shiftedMeta != "" {
		applyArgs = []string{
			"-y", "-i", tempTrim, "-i", shiftedMeta,
			"-map", "0:v?", "-map", "0:a?", "-ignore_unknown",
			"-map_metadata", "1", "-c", "copy",
			"-avoid_negative_ts", "make_zero",
			finalOut,
		}
	} else {
		applyArgs = []string{
			"-y", "-i", tempTrim,
			"-map", "0:v?", "-map", "0:a?", "-ignore_unknown",
			"-c", "copy",
			"-avoid_negative_ts", "make_zero",
			finalOut,
		}
	}

	out2, err2 := RunCmd(ctx2, "ffmpeg", applyArgs...)
	if err2 != nil {
		_ = os.Rename(tempTrim, finalOut)
		log.Printf("⚠️ ffmpeg finalise failed: %v (%s). Using raw trim.", err2, string(out2))
	} else {
		_ = os.Remove(tempTrim)
	}

	_ = os.Remove(origMeta)
	if shiftedMeta == "" {
		return finalOut, "", nil
	}
	return finalOut, shiftedMeta, nil
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
