package history

import (
	"fmt"
	"os"
	"path/filepath"
)

func replaceFileAtomically(filePath string, write func(*os.File) error) error {
	dir := filepath.Dir(filePath)
	tmp, err := os.CreateTemp(dir, filepath.Base(filePath)+".tmp-*")
	if err != nil {
		return fmt.Errorf("failed to create temporary file: %w", err)
	}

	tmpPath := tmp.Name()
	closed := false
	defer func() {
		if !closed {
			_ = tmp.Close()
		}
		_ = os.Remove(tmpPath)
	}()

	if err := write(tmp); err != nil {
		return err
	}
	if err := tmp.Sync(); err != nil {
		return fmt.Errorf("failed to sync temporary file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		closed = true
		return fmt.Errorf("failed to close temporary file: %w", err)
	}
	closed = true

	if err := os.Rename(tmpPath, filePath); err != nil {
		return fmt.Errorf("failed to replace file: %w", err)
	}
	return nil
}
