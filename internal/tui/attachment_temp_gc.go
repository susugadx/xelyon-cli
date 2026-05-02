package tui

import (
	"os"
	"path/filepath"
	"strings"
	"time"
)

const staleClipboardAttachmentTTL = 24 * time.Hour

var clipboardTempRootDir = os.TempDir

func cleanupStaleClipboardAttachmentTemps(now time.Time) int {
	root := strings.TrimSpace(clipboardTempRootDir())
	if root == "" {
		return 0
	}

	entries, err := os.ReadDir(root)
	if err != nil {
		return 0
	}

	cutoff := now.Add(-staleClipboardAttachmentTTL)
	removed := 0
	for _, entry := range entries {
		if !entry.IsDir() || !strings.HasPrefix(entry.Name(), clipboardAttachmentTempDirPrefix) {
			continue
		}
		info, infoErr := entry.Info()
		if infoErr != nil || info.ModTime().After(cutoff) {
			continue
		}
		if err := os.RemoveAll(filepath.Join(root, entry.Name())); err == nil {
			removed++
		}
	}

	return removed
}
