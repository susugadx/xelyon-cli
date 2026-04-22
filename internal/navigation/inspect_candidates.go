package navigation

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/susugadx/xelyon-cli/internal/ast"
	"github.com/susugadx/xelyon-cli/internal/tools/common"
)

// resolveSymbolCandidates はプロジェクト内からシンボル候補を検索する。
func resolveSymbolCandidates(symbol, pathHint string) []SymbolCandidate {
	return resolveSymbolCandidatesWithRuntime(symbol, pathHint, GoSymbolRuntime{})
}

// listGoFiles はプロジェクト内の Go ファイルを一覧する。
// pathHint が指定されている場合はそのパス配下に限定する。
func listGoFiles(pathHint string) []string {
	if !common.IsRipgrepAvailable() {
		return nil
	}

	searchPath := "."
	if pathHint != "" {
		// pathHint がファイルの場合はそのファイルだけ。
		if info, err := os.Stat(pathHint); err == nil && !info.IsDir() {
			if strings.HasSuffix(pathHint, ".go") {
				abs, _ := filepath.Abs(pathHint)
				return []string{abs}
			}
			return nil
		}
		searchPath = pathHint
	}

	args := []string{
		"--files",
		"--type", "go",
		"--glob", "!vendor/",
		searchPath,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, common.RipgrepPath(), args...)
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil && stdout.Len() == 0 {
		return nil
	}
	out := stdout.String()

	var files []string
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		abs, err := filepath.Abs(line)
		if err != nil {
			continue
		}
		files = append(files, abs)
	}
	return files
}

func extractASTSymbols(path string) ([]ast.Symbol, error) {
	return ast.ExtractSymbols(path)
}

func sortSymbolCandidates(candidates []SymbolCandidate) {
	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].File != candidates[j].File {
			return candidates[i].File < candidates[j].File
		}
		if candidates[i].Line != candidates[j].Line {
			return candidates[i].Line < candidates[j].Line
		}
		if candidates[i].EndLine != candidates[j].EndLine {
			return candidates[i].EndLine < candidates[j].EndLine
		}
		return candidateDisplayName(candidates[i]) < candidateDisplayName(candidates[j])
	})
}
