package services

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/sanke08/videoprocessor/ffmpeg"
	"github.com/sanke08/videoprocessor/models"
	"github.com/sanke08/videoprocessor/utils"
)

// ShiftChapterInAll shifts the START of the chapter named `chapterName` by
// `deltaSeconds` (signed) in every .mkv under `input`. The result's chapter
// table is then renormalised: chapters sorted by START and each END set to
// the next chapter's START (last END = file duration). Out-of-range START
// values are clamped to [0, duration].
func ShiftChapterInAll(input, output, chapterName string, deltaSeconds float64) error {
	files, _ := filepath.Glob(filepath.Join(input, "*.mkv"))
	sort.Slice(files, func(i, j int) bool {
		return utils.NaturalLess(files[i], files[j])
	})

	inPlace := output == "" || filepath.Clean(output) == filepath.Clean(input)
	if !inPlace {
		os.MkdirAll(output, 0755)
	}

	episodes := make([]models.EpisodeStatus, len(files))
	for i, f := range files {
		episodes[i] = models.EpisodeStatus{Name: filepath.Base(f), State: "pending"}
	}

	models.ProgressState.Update(func(p *models.Progress) {
		p.Total = len(files)
		p.Completed = 0
		p.Percent = 0
		p.Status = "shifting_chapter"
		p.Done = false
		p.Episodes = episodes
		p.Parts = nil
	})

	var wg sync.WaitGroup
	for i, f := range files {
		wg.Add(1)
		go func(idx int, file string) {
			defer wg.Done()
			log.Printf("▶️ [%02d] Shifting chapter %q by %.3fs -> %s", idx+1, chapterName, deltaSeconds, file)

			models.ProgressState.Update(func(p *models.Progress) {
				if idx < len(p.Episodes) {
					p.Episodes[idx].State = "processing"
				}
			})

			if err := shiftChapterInOne(file, output, chapterName, deltaSeconds, inPlace); err != nil {
				log.Printf("❌ [%02d] Failed: %v", idx+1, err)
				models.ProgressState.Update(func(p *models.Progress) {
					if idx < len(p.Episodes) {
						p.Episodes[idx].State = "failed"
						p.Episodes[idx].Error = err.Error()
					}
					p.Completed++
					if p.Total > 0 {
						p.Percent = float64(p.Completed) / float64(p.Total) * 100
					}
				})
				return
			}

			log.Printf("✅ [%02d] Chapter shifted -> %s", idx+1, file)
			models.ProgressState.Update(func(p *models.Progress) {
				if idx < len(p.Episodes) {
					p.Episodes[idx].State = "done"
				}
				p.Completed++
				if p.Total > 0 {
					p.Percent = float64(p.Completed) / float64(p.Total) * 100
				}
			})
		}(i, f)
	}

	wg.Wait()

	models.ProgressState.Update(func(p *models.Progress) {
		p.Status = "done"
		p.Percent = 100
		p.Done = true
	})

	return nil
}

func shiftChapterInOne(file, output, chapterName string, deltaSeconds float64, inPlace bool) error {
	duration, err := ffmpeg.GetDuration(file)
	if err != nil {
		return fmt.Errorf("get duration failed: %v", err)
	}

	tmpParent := filepath.Dir(file)
	if !inPlace {
		tmpParent = output
	}
	tmpDir, err := os.MkdirTemp(tmpParent, "tmp_shiftch_*")
	if err != nil {
		return fmt.Errorf("create temp dir failed: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	origMeta := filepath.Join(tmpDir, "orig.ffmeta")
	newMetaPath := filepath.Join(tmpDir, "new.ffmeta")
	tempOut := filepath.Join(tmpDir, "out.mkv")

	if err := ffmpeg.ExtractMetadata(file, origMeta); err != nil {
		return fmt.Errorf("extract metadata failed: %v", err)
	}
	mf, err := ffmpeg.ParseFFMetadata(origMeta)
	if err != nil || mf == nil {
		return fmt.Errorf("parse metadata failed: %v", err)
	}
	if mf.TimebaseNum <= 0 || mf.TimebaseDen <= 0 {
		mf.TimebaseNum = 1
		mf.TimebaseDen = 1000
	}

	// Find the chapter by exact name match.
	idx := -1
	for i, ch := range mf.Chapters {
		if ch.Title == chapterName {
			idx = i
			break
		}
	}
	if idx < 0 {
		return fmt.Errorf("chapter %q not found in %s", chapterName, filepath.Base(file))
	}

	durUnits := ffmpeg.SecondsToUnits(duration, mf.TimebaseNum, mf.TimebaseDen)
	deltaUnits := ffmpeg.SecondsToUnits(deltaSeconds, mf.TimebaseNum, mf.TimebaseDen)

	newStart := mf.Chapters[idx].Start + deltaUnits
	if newStart < 0 {
		newStart = 0
	}
	if newStart > durUnits {
		newStart = durUnits
	}
	mf.Chapters[idx].Start = newStart

	// Sort and recompute ENDs.
	sort.Slice(mf.Chapters, func(i, j int) bool {
		return mf.Chapters[i].Start < mf.Chapters[j].Start
	})
	for i := 0; i < len(mf.Chapters)-1; i++ {
		mf.Chapters[i].End = mf.Chapters[i+1].Start
	}
	if len(mf.Chapters) > 0 {
		mf.Chapters[len(mf.Chapters)-1].End = durUnits
	}

	if err := ffmpeg.WriteFFMetadata(newMetaPath, mf); err != nil {
		return fmt.Errorf("write ffmetadata failed: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	out, err := ffmpeg.RunCmd(ctx, "ffmpeg", "-y",
		"-i", file,
		"-i", newMetaPath,
		"-map", "0:v?",
		"-map", "0:a?",
		"-map", "0:s?",
		"-ignore_unknown",
		"-map_metadata", "1",
		"-map_chapters", "1",
		"-c", "copy",
		tempOut,
	)
	if err != nil {
		return fmt.Errorf("ffmpeg shift-chapter failed: %v (%s)", err, string(out))
	}

	if inPlace {
		bak := file + ".shiftch.bak"
		_ = os.Remove(bak)
		if err := os.Rename(file, bak); err != nil {
			return fmt.Errorf("rename source to bak failed: %v", err)
		}
		if err := os.Rename(tempOut, file); err != nil {
			if rbErr := os.Rename(bak, file); rbErr != nil {
				return fmt.Errorf("rename temp to source failed: %v (rollback also failed: %v)", err, rbErr)
			}
			return fmt.Errorf("rename temp to source failed: %v", err)
		}
		if err := os.Remove(bak); err != nil {
			log.Printf("⚠️  Could not remove backup %s: %v", bak, err)
		}
	} else {
		dst := filepath.Join(output, filepath.Base(file))
		_ = os.Remove(dst)
		if err := os.Rename(tempOut, dst); err != nil {
			return fmt.Errorf("rename to output failed: %v", err)
		}
	}

	return nil
}
