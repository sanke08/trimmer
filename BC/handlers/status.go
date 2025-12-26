package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/sanke08/videoprocessor/models"
)

// StatusHandler handles the /api/status endpoint
func StatusHandler(w http.ResponseWriter, r *http.Request) {
	var snapshot models.Progress
	models.ProgressState.Get(func(p *models.Progress) {
		snapshot = *p
	})
	json.NewEncoder(w).Encode(snapshot)
}
