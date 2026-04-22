package repomap

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/susugadx/xelyon-cli/internal/tools/common"
)

func (pm *ProjectMap) scanSymbols(states []fileState) (map[string][]Symbol, error) {
	return newSymbolScanStrategy(pm).scan(states)
}

func (pm *ProjectMap) runSymbolScan(def languagePattern, targets []string) (string, error) {
	args := []string{"-n", "-H", "--color", "never"}
	for _, pattern := range def.Patterns {
		args = append(args, "-e", pattern)
	}
	for _, ext := range def.Extensions {
		args = append(args, "--glob", "*"+ext)
	}
	args = append(args, "--")
	args = append(args, targets...)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, common.RipgrepPath(), args...)
	cmd.Dir = pm.RootPath

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 && stdout.Len() == 0 {
			return "", nil
		}
		if stderr.Len() > 0 {
			return "", fmt.Errorf("rg symbol scan failed: %s", strings.TrimSpace(stderr.String()))
		}
		return "", fmt.Errorf("rg symbol scan failed: %w", err)
	}
	return stdout.String(), nil
}
