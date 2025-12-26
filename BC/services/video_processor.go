package services

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"sync"

	"github.com/sanke08/videoprocessor/ffmpeg"
	"github.com/sanke08/videoprocessor/models"
	"github.com/sanke08/videoprocessor/utils"
)

// ProcessEpisodes is the main orchestrator for processing all episodes
func ProcessEpisodes(input, output string, opts models.TrimOptions) error {
	files, _ := filepath.Glob(filepath.Join(input, "*.mkv"))
	sort.Strings(files)
	os.MkdirAll(output, 0755)

	models.ProgressState.Update(func(p *models.Progress) {
		p.Total = len(files)
		p.Completed = 0
		p.Percent = 0
		p.Status = "processing"
		p.Done = false
	})

	type Result struct {
		Index    int
		File     string
		Meta     string
		Duration float64
		Err      error
	}

	results := make(chan Result, len(files))
	var wg sync.WaitGroup

	for i, f := range files {
		wg.Add(1)
		go func(idx int, file string) {
			defer wg.Done()
			log.Printf("▶️ [%02d] Starting -> %s", idx+1, file)

			ch, err := ffmpeg.ScanChapters(file)
			if err != nil {
				results <- Result{idx, "", "", 0, fmt.Errorf("scan failed: %v", err)}
				return
			}

			finalFile, metaFile, dur, err := ProcessSingleEpisode(file, output, ch, opts)
			if err != nil {
				results <- Result{idx, "", "", 0, fmt.Errorf("process failed: %v", err)}
				return
			}

			results <- Result{idx, finalFile, metaFile, dur, nil}
		}(i, f)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	allResults := make([]Result, len(files))
	for r := range results {
		allResults[r.Index] = r
	}

	processedFiles := make([]string, 0, len(files))
	metaFiles := make([]string, 0, len(files))
	durations := make([]float64, 0, len(files))

	for _, r := range allResults {
		if r.Err != nil {
			log.Printf("❌ [%02d] Failed: %v", r.Index+1, r.Err)
			continue
		}
		log.Printf("✅ [%02d] Trim success → %s", r.Index+1, r.File)
		processedFiles = append(processedFiles, r.File)
		metaFiles = append(metaFiles, r.Meta)
		durations = append(durations, r.Duration)

		models.ProgressState.Update(func(p *models.Progress) {
			p.Completed++
			if p.Total > 0 {
				p.Percent = (float64(p.Completed) / float64(p.Total)) * 100
			}
		})
	}

	// Merge processed files (parts)
	models.ProgressState.Update(func(p *models.Progress) {
		p.Status = "merging"
		p.Completed = 0
		p.Total = opts.Parts
		p.Percent = 0
	})

	if err := MergeEpisodes(processedFiles, metaFiles, durations, output, opts.Parts); err != nil {
		log.Println("⚠️ Merge error:", err)
	}

	models.ProgressState.Update(func(p *models.Progress) {
		p.Status = "done"
		p.Percent = 100
		p.Completed = p.Total
		p.Done = true
	})

	utils.CleanupTempFolders(output)
	return nil
}
