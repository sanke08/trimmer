package utils

import (
	"os"
	"path/filepath"
	"strings"
)

// CleanupTempFolders removes temporary files and folders from the output directory
func CleanupTempFolders(output string) {
	filepath.Walk(output, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		// Remove temp directories
		if info.IsDir() && strings.Contains(info.Name(), "tmp") {
			os.RemoveAll(path)
			return nil
		}

		// Remove list.txt or any leftover FFmpeg metadata files
		if !info.IsDir() && (strings.HasSuffix(info.Name(), ".txt") || strings.HasSuffix(info.Name(), ".log")) {
			os.Remove(path)
			return nil
		}

		return nil
	})
}
