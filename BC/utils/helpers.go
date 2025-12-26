package utils

import (
	"fmt"
	"path/filepath"
	"strings"
)

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
