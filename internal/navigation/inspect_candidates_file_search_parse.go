package navigation

import (
	"path/filepath"
	"strings"
)

func parseGoFileSearchOutput(output string) []string {
	var files []string
	for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		absPath, err := filepath.Abs(line)
		if err != nil {
			continue
		}
		files = append(files, absPath)
	}
	return files
}
