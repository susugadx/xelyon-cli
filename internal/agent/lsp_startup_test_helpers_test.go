package agent

import (
	"os"
	"path/filepath"
	"testing"
)

func withLSPCommandAvailability(t *testing.T, available map[string]bool) {
	t.Helper()

	binDir := t.TempDir()
	original := lspLookPath
	lspLookPath = func(file string) (string, error) {
		if available[file] {
			return filepath.Join(binDir, file), nil
		}
		return "", os.ErrNotExist
	}
	t.Cleanup(func() {
		lspLookPath = original
	})
}
