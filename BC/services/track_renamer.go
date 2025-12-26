package services

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// renameExtractedTracks renames extracted tracks to follow Part naming convention
func renameExtractedTracks(trackMap map[int]string, outputDir, trackType string, partNum int, ext string) {
	tracksDir := filepath.Join(outputDir, trackType)

	for _, oldPath := range trackMap {
		// Get the language from the filename
		base := filepath.Base(oldPath)

		// Extract language code from filename (format: originalfilename_lang_*.ext)
		parts := strings.Split(base, "_")
		var lang string
		if len(parts) >= 2 {
			lang = parts[len(parts)-2] // Get language part
		} else {
			lang = "unknown"
		}

		// Create new filename: Part{partNum}_{lang}.{ext}
		newFilename := fmt.Sprintf("Part%d_%s.%s", partNum, lang, ext)
		newPath := filepath.Join(tracksDir, newFilename)

		// Rename the file
		err := os.Rename(oldPath, newPath)
		if err != nil {
			log.Printf("⚠️ Failed to rename %s to %s: %v", oldPath, newPath, err)
		} else {
			log.Printf("📝 Renamed: %s → %s", filepath.Base(oldPath), newFilename)
		}
	}
}
