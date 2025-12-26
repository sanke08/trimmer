package services

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// CombineSubtitles combines subtitle files from multiple episodes with proper timing offsets
func CombineSubtitles(episodeSubsDir string, durations []float64, outputDir string, partNum int) error {
	subsDir := filepath.Join(episodeSubsDir, "subtitles")
	if _, err := os.Stat(subsDir); os.IsNotExist(err) {
		log.Printf("No subtitles directory found, skipping subtitle combination")
		return nil
	}

	// Find all subtitle files
	files, err := filepath.Glob(filepath.Join(subsDir, "*.srt"))
	if err != nil || len(files) == 0 {
		log.Printf("No subtitle files found to combine")
		return nil
	}

	// Group subtitle files by language/track
	trackGroups := make(map[string][]string)
	for _, file := range files {
		base := filepath.Base(file)
		// Extract language from filename (format: episodeName_lang_trackIndex.srt)
		parts := strings.Split(base, "_")
		if len(parts) >= 2 {
			lang := parts[len(parts)-2] // Get language
			trackGroups[lang] = append(trackGroups[lang], file)
		}
	}

	// Create output subtitles directory
	outSubsDir := filepath.Join(outputDir, "subtitles")
	os.MkdirAll(outSubsDir, 0755)

	// Combine each track group
	for lang, trackFiles := range trackGroups {
		sort.Strings(trackFiles)

		combinedFile := filepath.Join(outSubsDir, fmt.Sprintf("Part%d_%s.srt", partNum, lang))

		err := combineSRTFiles(trackFiles, combinedFile, durations)
		if err != nil {
			log.Printf("⚠️ Failed to combine %s subtitles: %v", lang, err)
			continue
		}

		log.Printf("✅ Combined subtitle track: Part%d_%s.srt", partNum, lang)
	}

	return nil
}

func combineSRTFiles(files []string, outputFile string, durations []float64) error {
	var allContent strings.Builder
	currentOffset := 0.0
	subtitleIndex := 1

	for i, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			log.Printf("⚠️ Failed to read %s: %v", file, err)
			continue
		}

		lines := strings.Split(string(content), "\n")

		for j := 0; j < len(lines); j++ {
			line := strings.TrimSpace(lines[j])

			// Skip empty lines between subtitles at the start
			if line == "" && allContent.Len() == 0 {
				continue
			}

			// Renumber subtitle indices
			if isSubtitleIndex(line) {
				allContent.WriteString(fmt.Sprintf("%d\n", subtitleIndex))
				subtitleIndex++
				continue
			}

			// Adjust timestamps
			if strings.Contains(line, " --> ") {
				parts := strings.Split(line, " --> ")
				if len(parts) == 2 {
					start := shiftSRTTimestampStr(strings.TrimSpace(parts[0]), currentOffset)
					end := shiftSRTTimestampStr(strings.TrimSpace(parts[1]), currentOffset)
					allContent.WriteString(fmt.Sprintf("%s --> %s\n", start, end))
					continue
				}
			}

			// Regular line (subtitle text)
			allContent.WriteString(line + "\n")
		}

		// Add offset for next episode
		if i < len(durations) {
			currentOffset += durations[i]
		}

		// Add blank line between episodes
		allContent.WriteString("\n")
	}

	return os.WriteFile(outputFile, []byte(allContent.String()), 0644)
}

func isSubtitleIndex(line string) bool {
	// Check if line is just a number (subtitle index)
	if line == "" {
		return false
	}
	for _, c := range line {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

func shiftSRTTimestampStr(timestamp string, offset float64) string {
	// Parse and shift SRT timestamp
	parts := strings.Split(timestamp, ",")
	if len(parts) != 2 {
		return timestamp
	}

	timeParts := strings.Split(parts[0], ":")
	if len(timeParts) != 3 {
		return timestamp
	}

	var hours, minutes, seconds, millis int
	fmt.Sscanf(timeParts[0], "%d", &hours)
	fmt.Sscanf(timeParts[1], "%d", &minutes)
	fmt.Sscanf(timeParts[2], "%d", &seconds)
	fmt.Sscanf(parts[1], "%d", &millis)

	totalSeconds := float64(hours*3600+minutes*60+seconds) + float64(millis)/1000.0
	totalSeconds += offset

	if totalSeconds < 0 {
		totalSeconds = 0
	}

	newHours := int(totalSeconds) / 3600
	newMinutes := (int(totalSeconds) % 3600) / 60
	newSeconds := int(totalSeconds) % 60
	newMillis := int((totalSeconds - float64(int(totalSeconds))) * 1000)

	return fmt.Sprintf("%02d:%02d:%02d,%03d", newHours, newMinutes, newSeconds, newMillis)
}
