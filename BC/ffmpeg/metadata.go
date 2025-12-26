package ffmpeg

import (
	"bufio"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"

	"github.com/sanke08/videoprocessor/models"
)

// UnitsToSeconds converts timebase units to seconds: seconds = units * (num/den)
func UnitsToSeconds(units int64, num, den int64) float64 {
	return float64(units) * (float64(num) / float64(den))
}

// SecondsToUnits converts seconds to timebase units: units = round(seconds * den/num)
func SecondsToUnits(sec float64, num, den int64) int64 {
	if num == 0 || den == 0 {
		return int64(math.Round(sec * 1000.0))
	}
	return int64(math.Round(sec * (float64(den) / float64(num))))
}

// ParseFFMetadata parses ffmetadata file created by: ffmpeg -i file -f ffmetadata out.txt
// It returns MetaFile; if TIMEBASE not present, default to 1/1000
func ParseFFMetadata(path string) (*models.MetaFile, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	mf := &models.MetaFile{
		TimebaseNum: 1,
		TimebaseDen: 1000,
		Chapters:    []models.MetaChapter{},
	}

	var currentChapter *models.MetaChapter
	inChapter := false

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, ";") {
			continue
		}

		if line == "[CHAPTER]" {
			if currentChapter != nil {
				mf.Chapters = append(mf.Chapters, *currentChapter)
			}
			currentChapter = &models.MetaChapter{}
			inChapter = true
			continue
		}

		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			if currentChapter != nil && inChapter {
				mf.Chapters = append(mf.Chapters, *currentChapter)
				currentChapter = nil
			}
			inChapter = false
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])

		if inChapter && currentChapter != nil {
			switch key {
			case "TIMEBASE":
				nums := strings.Split(val, "/")
				if len(nums) == 2 {
					n, _ := strconv.ParseInt(nums[0], 10, 64)
					d, _ := strconv.ParseInt(nums[1], 10, 64)
					if n > 0 && d > 0 {
						mf.TimebaseNum = n
						mf.TimebaseDen = d
					}
				}
			case "START":
				v, _ := strconv.ParseInt(val, 10, 64)
				currentChapter.Start = v
			case "END":
				v, _ := strconv.ParseInt(val, 10, 64)
				currentChapter.End = v
			case "title":
				currentChapter.Title = strings.ReplaceAll(val, "\\n", "\n")
			}
		} else {
			if key == "TIMEBASE" {
				nums := strings.Split(val, "/")
				if len(nums) == 2 {
					n, _ := strconv.ParseInt(nums[0], 10, 64)
					d, _ := strconv.ParseInt(nums[1], 10, 64)
					if n > 0 && d > 0 {
						mf.TimebaseNum = n
						mf.TimebaseDen = d
					}
				}
			}
		}
	}

	if currentChapter != nil && inChapter {
		mf.Chapters = append(mf.Chapters, *currentChapter)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return mf, nil
}

// WriteFFMetadata writes ffmetadata file from chapters and timebase
func WriteFFMetadata(path string, mf *models.MetaFile) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	w := bufio.NewWriter(f)
	fmt.Fprintln(w, ";FFMETADATA1")
	// global TIMEBASE - ffmetadata uses per-chapter TIMEBASE entries
	for _, ch := range mf.Chapters {
		fmt.Fprintln(w, "[CHAPTER]")
		fmt.Fprintf(w, "TIMEBASE=%d/%d\n", mf.TimebaseNum, mf.TimebaseDen)
		fmt.Fprintf(w, "START=%d\n", ch.Start)
		fmt.Fprintf(w, "END=%d\n", ch.End)
		if ch.Title != "" {
			// escape newlines in title
			title := strings.ReplaceAll(ch.Title, "\n", "\\n")
			fmt.Fprintf(w, "title=%s\n", title)
		}
	}
	w.Flush()
	return nil
}

// CreateShiftedMetadata filters & shifts metadata chapters for segment [segStart, segEnd] seconds
// origMetaPath -> shiftedMetaPath
func CreateShiftedMetadata(origMetaPath, shiftedMetaPath string, segStart, segEnd float64) error {
	mf, err := ParseFFMetadata(origMetaPath)
	if err != nil {
		return fmt.Errorf("parse ffmetadata failed: %v", err)
	}
	outMF := &models.MetaFile{
		TimebaseNum: mf.TimebaseNum,
		TimebaseDen: mf.TimebaseDen,
		Chapters:    []models.MetaChapter{},
	}
	for _, ch := range mf.Chapters {
		// convert to seconds
		chStartSec := UnitsToSeconds(ch.Start, mf.TimebaseNum, mf.TimebaseDen)
		chEndSec := UnitsToSeconds(ch.End, mf.TimebaseNum, mf.TimebaseDen)
		// check overlap with [segStart, segEnd)
		if chEndSec <= segStart || chStartSec >= segEnd {
			continue
		}
		// clip to segment bounds
		newStartSec := math.Max(chStartSec, segStart) - segStart
		newEndSec := math.Min(chEndSec, segEnd) - segStart
		// convert back to units with same timebase
		newStartUnits := SecondsToUnits(newStartSec, mf.TimebaseNum, mf.TimebaseDen)
		newEndUnits := SecondsToUnits(newEndSec, mf.TimebaseNum, mf.TimebaseDen)
		outMF.Chapters = append(outMF.Chapters, models.MetaChapter{
			Start: newStartUnits,
			End:   newEndUnits,
			Title: ch.Title,
		})
	}
	// if no chapters remain, leave empty file (ffmpeg will just ignore)
	return WriteFFMetadata(shiftedMetaPath, outMF)
}

// BuildCombinedChapters combines chapters from multiple metadata files with duration offsets
func BuildCombinedChapters(metaFiles []string, durations []float64, outMeta string) error {
	// metaFiles correspond to trimmed episodes (some may be empty)
	combined := &models.MetaFile{
		TimebaseNum: 1,
		TimebaseDen: 1000,
		Chapters:    []models.MetaChapter{},
	}
	offset := 0.0
	for i, mfPath := range metaFiles {
		if mfPath == "" {
			// no chapter metadata for this episode, skip
			offset += durations[i]
			continue
		}
		parsed, err := ParseFFMetadata(mfPath)
		if err != nil {
			// skip this meta file but add duration
			offset += durations[i]
			continue
		}
		// for each chapter add with offset
		for _, ch := range parsed.Chapters {
			// convert to seconds
			chStartSec := UnitsToSeconds(ch.Start, parsed.TimebaseNum, parsed.TimebaseDen)
			chEndSec := UnitsToSeconds(ch.End, parsed.TimebaseNum, parsed.TimebaseDen)
			newStartSec := offset + chStartSec
			newEndSec := offset + chEndSec
			// convert to default timebase units (1/1000)
			newStartUnits := SecondsToUnits(newStartSec, combined.TimebaseNum, combined.TimebaseDen)
			newEndUnits := SecondsToUnits(newEndSec, combined.TimebaseNum, combined.TimebaseDen)
			combined.Chapters = append(combined.Chapters, models.MetaChapter{
				Start: newStartUnits,
				End:   newEndUnits,
				Title: ch.Title,
			})
		}
		offset += durations[i]
	}

	// write combined meta
	return WriteFFMetadata(outMeta, combined)
}
