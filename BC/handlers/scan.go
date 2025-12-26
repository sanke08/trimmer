package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/sanke08/videoprocessor/ffmpeg"
)

// ScanHandler handles the /api/scan endpoint
func ScanHandler(w http.ResponseWriter, r *http.Request) {
	folder := r.URL.Query().Get("path")
	result, err := ffmpeg.ScanFirstTwoEpisodes(folder)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	json.NewEncoder(w).Encode(result)
}
