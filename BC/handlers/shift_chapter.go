package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"path/filepath"

	"github.com/sanke08/videoprocessor/models"
	"github.com/sanke08/videoprocessor/services"
)

// ShiftChapterHandler handles the /api/shift-chapter endpoint.
func ShiftChapterHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Input        string  `json:"input"`
		Output       string  `json:"output"`
		ChapterName  string  `json:"chapterName"`
		DeltaSeconds float64 `json:"deltaSeconds"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON body", 400)
		return
	}

	log.Printf("📥 /api/shift-chapter input=%q output=%q name=%q delta=%.3f",
		req.Input, req.Output, req.ChapterName, req.DeltaSeconds)

	files, globErr := filepath.Glob(filepath.Join(req.Input, "*.mkv"))
	inPlace := req.Output == "" || filepath.Clean(req.Output) == filepath.Clean(req.Input)
	log.Printf("   → found %d mkv files, inPlace=%v (globErr=%v)", len(files), inPlace, globErr)

	if len(files) == 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(400)
		json.NewEncoder(w).Encode(map[string]any{
			"status":  "no_files",
			"input":   req.Input,
			"message": fmt.Sprintf("no .mkv files found in %q", req.Input),
		})
		return
	}

	models.ProgressState.Update(func(p *models.Progress) {
		p.Status = "shifting_chapter"
		p.Done = false
		p.Total = len(files)
		p.Completed = 0
		p.Percent = 0
		p.Episodes = nil
		p.Parts = nil
	})
	go services.ShiftChapterInAll(req.Input, req.Output, req.ChapterName, req.DeltaSeconds)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"status":    "started",
		"fileCount": len(files),
		"inPlace":   inPlace,
	})
}
