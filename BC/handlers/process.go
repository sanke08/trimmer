package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/sanke08/videoprocessor/models"
	"github.com/sanke08/videoprocessor/services"
)

// ProcessHandler handles the /api/process endpoint
func ProcessHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Input   string             `json:"input"`
		Output  string             `json:"output"`
		Options models.TrimOptions `json:"options"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON body", 400)
		return
	}

	go services.ProcessEpisodes(req.Input, req.Output, req.Options)
	w.Write([]byte(`{"status":"started"}`))
}
