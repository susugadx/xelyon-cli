package navigation

import (
	"os"
	"path/filepath"
	"strings"
)

func contextAbsPath(filePath string) string {
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return filePath
	}
	return absPath
}

func readContextLines(filePath string) ([]string, error) {
	content, err := os.ReadFile(contextAbsPath(filePath))
	if err != nil {
		return nil, err
	}
	return strings.Split(string(content), "\n"), nil
}
