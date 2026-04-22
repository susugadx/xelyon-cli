package repomap

import (
	"bytes"
	"context"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func (pm *ProjectMap) loadGitStatus() []GitChange {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", "status", "--porcelain")
	cmd.Dir = pm.RootPath

	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		return nil
	}

	return parseGitStatusOutput(stdout.String())
}

func parseGitStatusOutput(output string) []GitChange {
	var changes []GitChange
	for _, line := range strings.Split(output, "\n") {
		if len(line) < 3 {
			continue
		}
		status := strings.TrimSpace(line[:2])
		path := strings.TrimSpace(line[3:])
		if idx := strings.LastIndex(path, " -> "); idx >= 0 {
			path = path[idx+4:]
		}
		if status == "" || path == "" {
			continue
		}
		changes = append(changes, GitChange{
			Status: status,
			Path:   filepath.ToSlash(path),
		})
	}
	return changes
}
