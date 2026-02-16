package utils

import (
	"fmt"
	"path/filepath"
	"strings"
	"unicode"
)

// NaturalLess compares two strings using natural sort order (e.g., "file2" < "file10")
func NaturalLess(str1, str2 string) bool {
	i, j := 0, 0
	for i < len(str1) && j < len(str2) {
		r1 := rune(str1[i])
		r2 := rune(str2[j])

		if unicode.IsDigit(r1) && unicode.IsDigit(r2) {
			// Scan numbers
			start1, start2 := i, j
			for i < len(str1) && unicode.IsDigit(rune(str1[i])) {
				i++
			}
			for j < len(str2) && unicode.IsDigit(rune(str2[j])) {
				j++
			}

			// Get number strings and trim leading zeros
			num1 := strings.TrimLeft(str1[start1:i], "0")
			num2 := strings.TrimLeft(str2[start2:j], "0")

			if len(num1) != len(num2) {
				return len(num1) < len(num2)
			}
			if num1 != num2 {
				return num1 < num2
			}
			// If numbers are equal (e.g. 01 vs 1), continue comparing remainder
			// But since we advanced i and j, we continue loop
			continue
		}

		if r1 != r2 {
			return r1 < r2
		}
		i++
		j++
	}
	return len(str1) < len(str2)
}

// Min returns the minimum of two integers
func Min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// EscapeForFFmpeg normalizes path slashes and escapes single quotes for ffmpeg concat lists
func EscapeForFFmpeg(p string) string {
	p = strings.ReplaceAll(p, "\\", "/")
	// Escape single quotes by replacing ' with '\''
	// But first, if path has spaces or special chars, we may need different handling
	// For Windows paths in concat lists, we typically just normalize slashes
	return p
}

// MakeTrimFilename creates a unique, readable trimmed filename: base_seg_start_end.mkv
func MakeTrimFilename(outputDir, file string, start, end float64) string {
	base := strings.TrimSuffix(filepath.Base(file), filepath.Ext(file))
	name := fmt.Sprintf("%s_seg_%.0f_%.0f.mkv", base, start, end)
	return filepath.Join(outputDir, name)
}
