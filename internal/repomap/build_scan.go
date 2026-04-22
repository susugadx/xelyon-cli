package repomap

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
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

func parseSymbolScanOutput(output string, seen map[string]map[int]struct{}) map[string][]Symbol {
	results := make(map[string][]Symbol)
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimRight(line, "\r")
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, ":", 3)
		if len(parts) != 3 {
			continue
		}

		path := filepath.Clean(filepath.ToSlash(parts[0]))
		lineNum, convErr := strconv.Atoi(parts[1])
		if convErr != nil {
			continue
		}
		content := parts[2]
		if !matchesSymbolPattern(path, content) {
			continue
		}

		if seen[path] == nil {
			seen[path] = make(map[int]struct{})
		}
		if _, ok := seen[path][lineNum]; ok {
			continue
		}
		seen[path][lineNum] = struct{}{}

		signature := normalizeSignature(content)
		name, kind, exported := signatureMetadataForPath(path, signature)
		results[path] = append(results[path], Symbol{
			Name:      name,
			Kind:      kind,
			Line:      lineNum,
			Signature: signature,
			Exported:  exported,
		})
	}
	return results
}
