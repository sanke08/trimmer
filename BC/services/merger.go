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

// MergeEpisodes merges processed episodes into final parts
func MergeEpisodes(processedFiles []string, metaFiles []string, durations []float64, output string, parts int) error {
	// Filter empty
	valid := make([]string, 0, len(processedFiles))
	validMeta := []string{}
	validDur := []float64{}
	for i, f := range processedFiles {
		if strings.TrimSpace(f) == "" {
			continue
		}
		if _, err := os.Stat(f); err != nil {
			log.Printf("⚠️ Skipping missing file in merge list: %s", f)
			continue
		}
		valid = append(valid, f)
		validMeta = append(validMeta, metaFiles[i])
		validDur = append(validDur, durations[i])
	}

	if len(valid) == 0 {
		return fmt.Errorf("no files to merge")
	}

	// sanitize parts
	if parts <= 0 {
		parts = 1
	}
	if parts > len(valid) {
		parts = len(valid)
	}

	partSize := (len(valid) + parts - 1) / parts

	for i := 0; i < parts; i++ {
		start := i * partSize
		end := utils.Min((i+1)*partSize, len(valid))
		partFiles := valid[start:end]
		partMeta := validMeta[start:end]
		partDur := validDur[start:end]
		if len(partFiles) == 0 {
			continue
		}

		models.ProgressState.Update(func(p *models.Progress) {
			if i < len(p.Parts) {
				p.Parts[i].State = "merging"
			}
		})

		// concat list
		listFile := filepath.Join(output, fmt.Sprintf("merge_part_%d_%d.txt", i+1, time.Now().UnixNano()))
		f, err := os.Create(listFile)
		if err != nil {
			return err
		}
		for _, pf := range partFiles {
			abs, _ := filepath.Abs(pf)
			_, _ = f.WriteString(fmt.Sprintf("file '%s'\n", utils.EscapeForFFmpeg(abs)))
		}
		f.Close()

		tmpMerged := filepath.Join(output, fmt.Sprintf("Part%d_tmp.mkv", i+1))
		// concat preserving streams
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
		outb, err := ffmpeg.RunCmd(ctx, "ffmpeg", "-y", "-f", "concat", "-safe", "0", "-i", listFile, "-map", "0:v?", "-map", "0:a?", "-ignore_unknown", "-c", "copy", "-fflags", "+genpts", "-avoid_negative_ts", "make_zero", tmpMerged)
		cancel()
		_ = os.Remove(listFile)
		if err != nil {
			models.ProgressState.Update(func(p *models.Progress) {
				if i < len(p.Parts) {
					p.Parts[i].State = "failed"
					p.Parts[i].Error = fmt.Sprintf("concat failed: %v", err)
				}
			})
			return fmt.Errorf("concat failed for part %d: %v (%s)", i+1, err, string(outb))
		}

		// Build combined chapters for this part
		partMetaOut := filepath.Join(output, fmt.Sprintf("part_%d_chapters.txt", i+1))
		if err := ffmpeg.BuildCombinedChapters(partMeta, partDur, partMetaOut); err != nil {
			// if build failed, we can continue without chapters for this part
			log.Printf("⚠️ buildCombinedChapters failed for part %d: %v", i+1, err)
			_ = os.Remove(partMetaOut)
			partMetaOut = ""
		}

		// apply chapters metadata to create final PartX.mkv
		partFinal := filepath.Join(output, fmt.Sprintf("Part%d.mkv", i+1))
		if partMetaOut != "" {
			ctx2, cancel2 := context.WithTimeout(context.Background(), 5*time.Minute)
			outb2, err2 := ffmpeg.RunCmd(ctx2, "ffmpeg", "-y", "-i", tmpMerged, "-i", partMetaOut, "-map", "0:v?", "-map", "0:a?", "-ignore_unknown", "-map_metadata", "1", "-c", "copy", partFinal)
			cancel2()
			if err2 != nil {
				// fallback to tmpMerged
				log.Printf("⚠️ failed apply chapters for part %d: %v (%s). Using tmp merged.", i+1, err2, string(outb2))
				_ = os.Rename(tmpMerged, partFinal)
			} else {
				_ = os.Remove(tmpMerged)
			}
			_ = os.Remove(partMetaOut)
		} else {
			_ = os.Rename(tmpMerged, partFinal)
		}

		models.ProgressState.Update(func(p *models.Progress) {
			p.Completed++
			if p.Total > 0 {
				p.Percent = (float64(p.Completed) / float64(p.Total)) * 100
			}
			if i < len(p.Parts) {
				p.Parts[i].State = "done"
			}
		})
	}

	// conservative cleanup: remove files matching trimmed naming pattern
	for _, f := range processedFiles {
		l := strings.ToLower(filepath.Base(f))
		if strings.Contains(l, "_seg_") || strings.HasPrefix(l, "merged_") || strings.Contains(strings.ToLower(f), "tmp") {
			_ = os.Remove(f)
		}
	}
	return nil
}
