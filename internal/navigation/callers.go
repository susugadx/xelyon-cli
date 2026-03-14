package navigation

import (
	"bufio"
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/susugadx/xelyon-cli/internal/ast"
	"github.com/susugadx/xelyon-cli/internal/tools/common"
)

// maxRipgrepResults は findReferences が返す参照の上限。
// これを超えた場合は truncated=true を返す。
const maxRipgrepResults = 200

const (
	ripgrepScannerInitialBufferSize = 64 * 1024
	ripgrepScannerMaxBufferSize     = 1024 * 1024
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

	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 0, ripgrepScannerInitialBufferSize), ripgrepScannerMaxBufferSize)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		ref := parseRipgrepLine(line, symbol)
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

// parseRipgrepLine は "file:line:content" 形式の行をパースする。
func parseRipgrepLine(line, symbol string) *Reference {
	// file:line:content
	firstColon := strings.Index(line, ":")
	if firstColon < 0 {
		return nil
	}
	rest := line[firstColon+1:]
	secondColon := strings.Index(rest, ":")
	if secondColon < 0 {
		return nil
	}

	filePath := line[:firstColon]
	lineNumStr := rest[:secondColon]
	content := rest[secondColon+1:]

	lineNum, err := strconv.Atoi(lineNumStr)
	if err != nil || lineNum <= 0 {
		return nil
	}

	relPath := toRelativePath(mustAbs(filePath))
	isTest := strings.HasSuffix(filePath, "_test.go")

	// AST 分類を試みる
	scope := ""
	class := ast.ClassUnknown
	absPath := mustAbs(filePath)
	if ast.IsSupportedFile(absPath) {
		src, err := os.ReadFile(absPath)
		if err == nil {
			if info, err := ast.ClassifyLine(absPath, src, lineNum, symbol); err == nil && info != nil {
				scope = info.Scope
				class = info.Class
			}
		}
	}

	return &Reference{
		File:    relPath,
		Line:    lineNum,
		Scope:   scope,
		Snippet: strings.TrimSpace(content),
		IsTest:  isTest,
		Class:   class,
	}
}

// classifyCallers は参照一覧からコール箇所を抽出する。
// AST 分類（ClassCall）を使い、定義行自体・テスト・定義宣言を除外する。
func classifyCallers(refs []Reference, def SymbolCandidate, limit int) ([]Reference, bool) {
	var callers []Reference

	for _, ref := range refs {
		// 定義行自体を除外
		if ref.File == def.File && ref.Line >= def.Line && ref.Line <= def.EndLine {
			continue
		}
		// テストファイルは tests で扱う
		if ref.IsTest {
			continue
		}
		// AST 分類が ClassCall のもののみ caller
		if ref.Class == ast.ClassCall {
			callers = append(callers, ref)
		}
	}

	if len(callers) > limit {
		return callers[:limit], true
	}
	return callers, false
}

// classifyRefs は参照一覧から caller でもテストでもないものを抽出する。
// ClassDef（他シンボルの定義）、ClassCall（callers で扱い済み）、
// ClassImport、ClassComment、ClassString を除外する。
// テストファイルの参照は Related tests で扱うため除外する。
func classifyRefs(refs []Reference, def SymbolCandidate, limit int) ([]Reference, bool) {
	var filtered []Reference

	for _, ref := range refs {
		// 定義行自体を除外
		if ref.File == def.File && ref.Line >= def.Line && ref.Line <= def.EndLine {
			continue
		}
		// caller は callers で扱い済み
		if ref.Class == ast.ClassCall {
			continue
		}
		// 他シンボルの定義、import、コメント、文字列は除外
		switch ref.Class {
		case ast.ClassDef, ast.ClassImport, ast.ClassComment, ast.ClassString:
			continue
		}
		// テストファイルの参照は Related tests で扱う
		if ref.IsTest {
			continue
		}
		filtered = append(filtered, ref)
	}

	if len(filtered) > limit {
		return filtered[:limit], true
	}
	return filtered, false
}

// mustAbs はパスを絶対パスに変換する。エラー時はそのまま返す。
func mustAbs(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		return path
	}
	return abs
}
