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

// AddChapterToAll adds one chapter at timeSeconds to every .mkv file in `input`.
// If `output` is empty or equals `input`, the result replaces the source file in place.
// Otherwise, results are written to `output/<original-basename>`.
func AddChapterToAll(input, output, chapterName string, timeSeconds float64) error {
	files, _ := filepath.Glob(filepath.Join(input, "*.mkv"))
	sort.Slice(files, func(i, j int) bool {
		return utils.NaturalLess(files[i], files[j])
	})

	// Determine if this is an in-place operation.
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
		p.Status = "adding_chapter"
		p.Done = false
		p.Episodes = episodes
		p.Parts = nil
	})

	var wg sync.WaitGroup
	for i, f := range files {
		wg.Add(1)
		go func(idx int, file string) {
			defer wg.Done()
			log.Printf("▶️ [%02d] Adding chapter -> %s", idx+1, file)

			models.ProgressState.Update(func(p *models.Progress) {
				if idx < len(p.Episodes) {
					p.Episodes[idx].State = "processing"
				}
			})

			if err := addChapterToOne(file, output, chapterName, timeSeconds, inPlace); err != nil {
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

			log.Printf("✅ [%02d] Chapter added -> %s", idx+1, file)
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

// addChapterToOne processes a single mkv file.
func addChapterToOne(file, output, chapterName string, timeSeconds float64, inPlace bool) error {
	duration, err := ffmpeg.GetDuration(file)
	if err != nil {
		return fmt.Errorf("get duration failed: %v", err)
	}
	if timeSeconds < 0 || timeSeconds >= duration {
		return fmt.Errorf("timeSeconds %.3f out of range [0, %.3f)", timeSeconds, duration)
	}

	// Pick a directory to host the temp dir: source dir for in-place, else output dir.
	tmpParent := filepath.Dir(file)
	if !inPlace {
		tmpParent = output
	}
	tmpDir, err := os.MkdirTemp(tmpParent, "tmp_addch_*")
	if err != nil {
		return fmt.Errorf("create temp dir failed: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	origMeta := filepath.Join(tmpDir, "orig.ffmeta")
	newMetaPath := filepath.Join(tmpDir, "new.ffmeta")
	tempOut := filepath.Join(tmpDir, "out.mkv")

	// Extract existing metadata. If it fails, fall back to empty chapters.
	var mf *models.MetaFile
	if extractErr := ffmpeg.ExtractMetadata(file, origMeta); extractErr != nil {
		log.Printf("⚠️  ExtractMetadata returned error for %s, proceeding with empty chapter list: %v", filepath.Base(file), extractErr)
		mf = &models.MetaFile{TimebaseNum: 1, TimebaseDen: 1000, Chapters: []models.MetaChapter{}}
	} else {
		parsed, parseErr := ffmpeg.ParseFFMetadata(origMeta)
		if parseErr != nil || parsed == nil {
			log.Printf("⚠️  ParseFFMetadata failed for %s, starting fresh: %v", filepath.Base(file), parseErr)
			mf = &models.MetaFile{TimebaseNum: 1, TimebaseDen: 1000, Chapters: []models.MetaChapter{}}
		} else {
			mf = parsed
			if mf.TimebaseNum <= 0 || mf.TimebaseDen <= 0 {
				mf.TimebaseNum = 1
				mf.TimebaseDen = 1000
			}
		}
	}

	// Append the new chapter (End placeholder, recomputed after sort).
	newUnits := ffmpeg.SecondsToUnits(timeSeconds, mf.TimebaseNum, mf.TimebaseDen)
	mf.Chapters = append(mf.Chapters, models.MetaChapter{
		Start: newUnits,
		End:   newUnits,
		Title: chapterName,
	})

	// Sort ascending by Start.
	sort.Slice(mf.Chapters, func(i, j int) bool {
		return mf.Chapters[i].Start < mf.Chapters[j].Start
	})

	// Recompute ENDs: each chapter ends where the next begins; last ends at duration.
	durUnits := ffmpeg.SecondsToUnits(duration, mf.TimebaseNum, mf.TimebaseDen)
	for i := 0; i < len(mf.Chapters)-1; i++ {
		mf.Chapters[i].End = mf.Chapters[i+1].Start
	}
	if len(mf.Chapters) > 0 {
		mf.Chapters[len(mf.Chapters)-1].End = durUnits
	}

	if err := ffmpeg.WriteFFMetadata(newMetaPath, mf); err != nil {
		return fmt.Errorf("write ffmetadata failed: %v", err)
	}

	// Re-mux with the merged metadata.
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
		return fmt.Errorf("ffmpeg add-chapter failed: %v (%s)", err, string(out))
	}

	// Place the result.
	if inPlace {
		// Safer 2-step swap on Windows: source -> .bak, tempOut -> source, remove .bak.
		bak := file + ".addch.bak"
		// Clean any stale bak.
		_ = os.Remove(bak)
		if err := os.Rename(file, bak); err != nil {
			return fmt.Errorf("rename source to bak failed: %v", err)
		}
		if err := os.Rename(tempOut, file); err != nil {
			// Roll back.
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
		// Remove any existing destination so Rename succeeds on Windows.
		_ = os.Remove(dst)
		if err := os.Rename(tempOut, dst); err != nil {
			return fmt.Errorf("rename to output failed: %v", err)
		}
	}

	return nil
}
