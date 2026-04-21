package navigation

import (
	"bufio"
	"context"
	"io"
	"os/exec"
	"strings"
	"time"

	"github.com/susugadx/xelyon-cli/internal/tools/common"
)

// maxRipgrepResults は findReferences が返す参照の上限。
// これを超えた場合は truncated=true を返す。
const maxRipgrepResults = 500

const (
	ripgrepScannerInitialBufferSize = 64 * 1024
	ripgrepScannerMaxBufferSize     = 1024 * 1024
	lspReferenceTimeout             = 5 * time.Second
)

// referenceSearchResult は ripgrep 参照検索の内部状態を保持する。
type referenceSearchResult struct {
	Refs          []Reference
	Truncated     bool
	Incomplete    bool
	StopRequested bool
}

// findReferences は ripgrep でシンボル名を検索し、全参照を返す。
// StdoutPipe + scanner で逐次読み取りし、201件目を検出したら早期停止する。
// truncated が true の場合、上流の検索結果が上限を超えたことを示す。
// incomplete が true の場合、読み取り失敗や異常終了により結果が不完全であることを示す。
func findReferences(symbol string) (refs []Reference, truncated bool, incomplete bool) {
	if !common.IsRipgrepAvailable() {
		return nil, false, false
	}

	args := []string{
		"-n",
		"--no-heading",
		"--color", "never",
		"-w", // 単語境界
		"--type", "go",
		"--glob", "!vendor/",
		symbol,
		".",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, common.RipgrepPath(), args...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, false, true
	}
	if err := cmd.Start(); err != nil {
		return nil, false, true
	}

	return runReferenceSearch(stdout, symbol, cancel, cmd.Wait)
}

// collectReferenceSearchResult は ripgrep の標準出力を読み取り、参照一覧を構築する。
func collectReferenceSearchResult(reader io.Reader, symbol string) referenceSearchResult {
	result := referenceSearchResult{}
	if reader == nil {
		result.Incomplete = true
		return result
	}

	cache := newReferenceParseCache()
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 0, ripgrepScannerInitialBufferSize), ripgrepScannerMaxBufferSize)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		ref := parseRipgrepLine(line, symbol, cache)
		if ref == nil {
			continue
		}
		result.Refs = append(result.Refs, *ref)

		// 201件目を検出したら truncated=true にし、先頭200件のみ保持して早期停止
		if len(result.Refs) > maxRipgrepResults {
			result.Truncated = true
			result.Refs = result.Refs[:maxRipgrepResults]
			result.StopRequested = true
			break
		}
	}

	if err := scanner.Err(); err != nil {
		result.Incomplete = true
	}

	return result
}

// runReferenceSearch は参照ストリームの読み取りと終了待機をまとめて処理する。
func runReferenceSearch(reader io.Reader, symbol string, cancel func(), wait func() error) ([]Reference, bool, bool) {
	result := collectReferenceSearchResult(reader, symbol)
	if result.StopRequested && cancel != nil {
		cancel()
	}
	if wait != nil {
		if err := wait(); err != nil && !result.StopRequested {
			result.Incomplete = true
		}
	}
	return result.Refs, result.Truncated, result.Incomplete
}

func findReferencesWithFallbackRuntime(baseName string, cand SymbolCandidate, runtime GoSymbolRuntime) ([]Reference, bool, bool) {
	ambiguousFiles := findAmbiguousFilesWithRuntime(baseName, cand, runtime)
	allRefs, truncated, incomplete := findReferences(baseName)
	return filterRefsByCandidate(allRefs, cand, ambiguousFiles), truncated, incomplete
}
