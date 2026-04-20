package repomap

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/susugadx/xelyon-cli/internal/ast"
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

func (pm *ProjectMap) buildFileStates(paths []string, cache *MapCache) ([]fileState, error) {
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
		if cached, ok := cache.Files[relPath]; ok && cached != nil && state.modTime.Equal(cached.ModTime) {
			state.cached = cloneCacheFile(cached)
		}
		states = append(states, state)
	}
	return states, nil
}

func (pm *ProjectMap) scanSymbols(states []fileState) (map[string][]Symbol, error) {
	results := make(map[string][]Symbol)

	for _, state := range states {
		if state.cached != nil || !state.supportsSym || !ast.IsSupportedFile(state.path) {
			continue
		}

		astSymbols, err := ast.ExtractSymbols(state.absPath)
		if err != nil {
			continue
		}

		repoSymbols := make([]Symbol, 0, len(astSymbols))
		for _, symbol := range astSymbols {
			repoSymbols = append(repoSymbols, Symbol{
				Name:      symbol.Name,
				Kind:      string(symbol.Kind),
				Line:      symbol.Line,
				EndLine:   symbol.EndLine,
				Signature: symbol.Signature,
				Exported:  symbol.Exported,
			})
		}
		results[state.path] = repoSymbols
	}

	targetsByExt := make(map[string][]string)
	for _, state := range states {
		if state.cached != nil || !state.supportsSym {
			continue
		}
		if _, done := results[state.path]; done {
			continue
		}
		ext := extensionForPath(state.path)
		if ext == "" {
			continue
		}
		targetsByExt[ext] = append(targetsByExt[ext], state.path)
	}
	if len(targetsByExt) == 0 {
		sortSymbolsByLocation(results)
		return results, nil
	}

	seen := make(map[string]map[int]struct{})

	for _, def := range defaultPatterns {
		var targets []string
		for _, ext := range def.Extensions {
			targets = append(targets, targetsByExt[ext]...)
		}
		if len(targets) == 0 {
			continue
		}

		symbols, err := pm.runRgAndParse(def, targets, seen)
		if err != nil {
			return nil, err
		}
		for path, syms := range symbols {
			results[path] = append(results[path], syms...)
		}
	}

	sortSymbolsByLocation(results)
	return results, nil
}

func (pm *ProjectMap) runRgAndParse(def languagePattern, targets []string, seen map[string]map[int]struct{}) (map[string][]Symbol, error) {
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
			return map[string][]Symbol{}, nil
		}
		if stderr.Len() > 0 {
			return nil, fmt.Errorf("rg symbol scan failed: %s", strings.TrimSpace(stderr.String()))
		}
		return nil, fmt.Errorf("rg symbol scan failed: %w", err)
	}

	results := make(map[string][]Symbol)
	for _, line := range strings.Split(stdout.String(), "\n") {
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

	return results, nil
}

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

	var changes []GitChange
	for _, line := range strings.Split(stdout.String(), "\n") {
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

func (pm *ProjectMap) ignorePatterns() []string {
	patterns := append([]string{}, pathmatch.DefaultIgnorePatterns()...)
	patterns = append(patterns, pm.additionalIgnoreDirs...)
	return pathmatch.NormalizePatterns(patterns)
}

func countLines(path string) (int, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	count := 0
	reader := bufio.NewReaderSize(f, 32*1024)
	for {
		b, err := reader.ReadByte()
		if err != nil {
			if err == io.EOF {
				return count, nil
			}
			return 0, err
		}
		if b == '\n' {
			count++
		}
	}
}
