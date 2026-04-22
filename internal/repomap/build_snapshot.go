package repomap

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/susugadx/xelyon-cli/internal/pathmatch"
	"github.com/susugadx/xelyon-cli/internal/tools/common"
)

func (pm *ProjectMap) listFiles() ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	args := []string{"--files"}
	for _, glob := range pathmatch.BuildRGIgnoreGlobs(pm.ignorePatterns()) {
		args = append(args, "--glob", glob)
	}

	cmd := exec.CommandContext(ctx, common.RipgrepPath(), args...)
	cmd.Dir = pm.RootPath

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 && stdout.Len() == 0 {
			return nil, nil
		}
		return nil, fmt.Errorf("rg --files failed: %w: %s", err, strings.TrimSpace(stderr.String()))
	}

	var paths []string
	for _, line := range strings.Split(stdout.String(), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		paths = append(paths, filepath.Clean(filepath.ToSlash(line)))
	}
	sort.Strings(paths)
	return paths, nil
}

func (pm *ProjectMap) buildFileStates(paths []string) ([]fileState, error) {
	states := make([]fileState, 0, len(paths))
	for _, relPath := range paths {
		absPath := filepath.Join(pm.RootPath, relPath)
		info, err := os.Stat(absPath)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("stat %s: %w", relPath, err)
		}

		state := fileState{
			path:        relPath,
			absPath:     absPath,
			modTime:     info.ModTime().UTC(),
			supportsSym: supportsSymbols(relPath),
		}
		states = append(states, state)
	}
	return states, nil
}

func (pm *ProjectMap) ignorePatterns() []string {
	patterns := append([]string{}, pathmatch.DefaultIgnorePatterns()...)
	patterns = append(patterns, pm.additionalIgnoreDirs...)
	return pathmatch.NormalizePatterns(patterns)
}
