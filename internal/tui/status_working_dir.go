package tui

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/susugadx/xelyon-cli/internal/tui/termtext"
	"github.com/susugadx/xelyon-cli/internal/tui/theme"
)

const (
	workingDirStatusPrefix   = "cwd: "
	minWorkingDirStatusWidth = 12
	pathTruncationMarker     = "..."
)

func currentWorkingDirForStatus() string {
	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}
	return filepath.Clean(cwd)
}

func (m Model) renderWorkingDirStatusSegment(maxWidth int) string {
	if maxWidth < minWorkingDirStatusWidth || strings.TrimSpace(m.workingDir) == "" {
		return ""
	}

	pathMaxWidth := maxWidth - lipgloss.Width(workingDirStatusPrefix)
	if pathMaxWidth <= lipgloss.Width(pathTruncationMarker) {
		return ""
	}

	displayPath := formatWorkingDirForStatus(m.workingDir)
	if displayPath == "" {
		return ""
	}
	displayPath = sanitizeWorkingDirStatusPath(displayPath)
	displayPath = truncateWorkingDirStatusPath(displayPath, pathMaxWidth)
	if displayPath == "" {
		return ""
	}
	return theme.Chrome.HintFg + workingDirStatusPrefix + displayPath + theme.Chrome.Reset
}

func sanitizeWorkingDirStatusPath(path string) string {
	return termtext.SanitizeSingleLineANSI(termtext.StripANSI(path))
}

func formatWorkingDirForStatus(cwd string) string {
	home, _ := os.UserHomeDir()
	return formatWorkingDirForStatusWithHome(cwd, home)
}

func formatWorkingDirForStatusWithHome(cwd, home string) string {
	cwd = filepath.Clean(strings.TrimSpace(cwd))
	if cwd == "." || cwd == "" {
		return ""
	}

	home = filepath.Clean(strings.TrimSpace(home))
	if home != "." && home != "" && pathWithinOrEqual(cwd, home) {
		if cwd == home {
			return "~"
		}
		rel, err := filepath.Rel(home, cwd)
		if err == nil && rel != "." && !strings.HasPrefix(rel, "..") {
			return filepath.ToSlash(filepath.Join("~", rel))
		}
	}
	return filepath.ToSlash(cwd)
}

func pathWithinOrEqual(path, root string) bool {
	if path == root {
		return true
	}
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func truncateWorkingDirStatusPath(path string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}
	if termtext.PlainTextDisplayWidth(path) <= maxWidth {
		return path
	}

	markerWidth := lipgloss.Width(pathTruncationMarker)
	if maxWidth <= markerWidth {
		return termtext.TruncateWithANSI(path, maxWidth)
	}

	suffixWidth := maxWidth - markerWidth
	return pathTruncationMarker + displaySuffix(path, suffixWidth)
}

func displaySuffix(s string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}

	runes := []rune(s)
	start := len(runes)
	width := 0
	for start > 0 {
		next := string(runes[start-1 : start])
		nextWidth := termtext.PlainTextDisplayWidth(next)
		if width+nextWidth > maxWidth {
			break
		}
		start--
		width += nextWidth
	}
	return string(runes[start:])
}
