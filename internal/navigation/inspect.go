package navigation

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/susugadx/xelyon-cli/internal/ast"
	"github.com/susugadx/xelyon-cli/internal/tools/common"
)

// InspectMode は inspect_symbol の出力モード。
type InspectMode string

const (
	ModeSummary InspectMode = "summary"
	ModeFull    InspectMode = "full"
)

// Budget は inspect_symbol の出力上限。
type Budget struct {
	BodyLines   int
	CallerLimit int
	RefLimit    int
	TestLimit   int
}

// SummaryBudget は summary モードの出力上限。
var SummaryBudget = Budget{
	BodyLines:   15,
	CallerLimit: 2,
	RefLimit:    2,
	TestLimit:   1,
}

// FullBudget は full モードの出力上限。
var FullBudget = Budget{
	BodyLines:   120,
	CallerLimit: 12,
	RefLimit:    16,
	TestLimit:   6,
}

// SymbolCandidate はシンボル候補。
type SymbolCandidate struct {
	Name      string
	Kind      string
	File      string // プロジェクトルートからの相対パス
	Line      int
	EndLine   int
	Signature string
}

// InspectResult は inspect_symbol の結果。
type InspectResult struct {
	// 単一候補の場合
	Symbol  *SymbolCandidate
	Body    []string // 行番号付き本文
	Callers []Reference
	Refs    []Reference
	Tests   []TestRef

	// 複数候補の場合
	Candidates []SymbolCandidate

	// 打ち切り情報
	TotalCallers       int
	TotalRefs          int
	TotalTests         int
	MoreCallers        bool
	MoreRefs           bool
	MoreTests          bool
	UpstreamTruncated  bool
	UpstreamIncomplete bool
}

// Reference はシンボル参照。
type Reference struct {
	File    string
	Line    int
	Scope   string // 包含関数名
	Snippet string // マッチ行テキスト
	IsTest  bool
	Class   ast.MatchClass // AST 分類（ClassCall, ClassRef, ClassDef 等）
}

// TestRef はテスト参照。
type TestRef struct {
	File      string
	Name      string
	Line      int
	EndLine   int
	Signature string
}

// InspectSymbol は指定シンボルの定義・caller・ref・テストをまとめて返す。
func InspectSymbol(symbol, pathHint, mode string) string {
	if symbol == "" {
		return "Error: symbol is required"
	}

	inspectMode := ModeSummary
	if mode == "full" {
		inspectMode = ModeFull
	}

	budget := SummaryBudget
	if inspectMode == ModeFull {
		budget = FullBudget
	}

	// 1. シンボル候補を解決
	candidates := resolveSymbolCandidates(symbol, pathHint)
	if len(candidates) == 0 {
		return fmt.Sprintf("No symbol found: %q", symbol)
	}

	// 2. 複数候補 → 一覧のみ
	if len(candidates) > 1 {
		return formatMultipleCandidates(symbol, candidates)
	}

	// 3. 単一候補 → 詳細取得
	cand := candidates[0]
	result := InspectResult{Symbol: &cand}

	// 定義本文
	result.Body = readDefinitionBody(cand, budget.BodyLines)

	// 同名シンボルを持つ他ファイルを特定（曖昧ファイル）
	// pathHint で絞った場合でも、プロジェクト全体で同名定義がどこにあるか調べる
	ambiguousFiles := findAmbiguousFiles(symbol, cand)

	// Caller / Reference / Test
	allRefs, upstreamTruncated, upstreamIncomplete := findReferences(symbol)
	result.UpstreamTruncated = upstreamTruncated
	result.UpstreamIncomplete = upstreamIncomplete
	allRefs = filterRefsByCandidate(allRefs, cand, ambiguousFiles)
	result.Callers, result.TotalCallers, result.MoreCallers = classifyCallers(allRefs, cand, budget.CallerLimit)
	result.Refs, result.TotalRefs, result.MoreRefs = classifyRefs(allRefs, cand, budget.RefLimit)
	result.Tests, result.TotalTests, result.MoreTests = findRelatedTests(symbol, allRefs, budget.TestLimit)

	return formatInspectResult(result)
}

// resolveSymbolCandidates はプロジェクト内からシンボル候補を検索する。
func resolveSymbolCandidates(symbol, pathHint string) []SymbolCandidate {
	goFiles := listGoFiles(pathHint)
	if len(goFiles) == 0 {
		return nil
	}

	var candidates []SymbolCandidate
	for _, f := range goFiles {
		symbols, err := ast.ExtractSymbols(f)
		if err != nil {
			continue
		}
		for _, s := range symbols {
			if s.Name == symbol {
				relPath := toRelativePath(f)
				candidates = append(candidates, SymbolCandidate{
					Name:      s.Name,
					Kind:      string(s.Kind),
					File:      relPath,
					Line:      s.Line,
					EndLine:   s.EndLine,
					Signature: s.Signature,
				})
			}
		}
	}
	return candidates
}

// listGoFiles はプロジェクト内の Go ファイルを一覧する。
// pathHint が指定されている場合はそのパス配下に限定する。
func listGoFiles(pathHint string) []string {
	if !common.IsRipgrepAvailable() {
		return nil
	}

	searchPath := "."
	if pathHint != "" {
		// pathHint がファイルの場合はそのファイルだけ
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

// readDefinitionBody は定義本文を読み出す。
func readDefinitionBody(cand SymbolCandidate, maxLines int) []string {
	absPath, err := filepath.Abs(cand.File)
	if err != nil {
		return nil
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
		return nil
	}

	lines := strings.Split(string(data), "\n")
	start := cand.Line - 1 // 0-indexed
	end := cand.EndLine    // exclusive

	if start < 0 {
		start = 0
	}
	if end > len(lines) {
		end = len(lines)
	}

	totalLines := end - start
	truncated := totalLines > maxLines
	if truncated {
		end = start + maxLines
	}

	var result []string
	for i := start; i < end; i++ {
		result = append(result, fmt.Sprintf("%d: %s", i+1, lines[i]))
	}
	if truncated {
		result = append(result, fmt.Sprintf("... (%d more lines, L%d-L%d)", totalLines-maxLines, start+maxLines+1, cand.EndLine))
	}
	return result
}

// toRelativePath は絶対パスを作業ディレクトリからの相対パスに変換する。
func toRelativePath(absPath string) string {
	cwd, err := os.Getwd()
	if err != nil {
		return absPath
	}
	rel, err := filepath.Rel(cwd, absPath)
	if err != nil {
		return absPath
	}
	return rel
}

// findAmbiguousFiles は、候補と同名のシンボル定義を持つ他ファイルを特定する。
// これらのファイル内の call/ref は別のシンボルを参照している可能性が高いため、
// 保守的に除外対象とする。
func findAmbiguousFiles(symbol string, cand SymbolCandidate) map[string]bool {
	ambiguous := make(map[string]bool)

	// プロジェクト全体で同名シンボルを検索（pathHint なし）
	allCandidates := resolveSymbolCandidates(symbol, "")
	for _, c := range allCandidates {
		if c.File != cand.File {
			ambiguous[c.File] = true
		}
	}
	return ambiguous
}

// filterRefsByCandidate は、候補に安全に帰属できない参照を除外する。
// - 候補自身の定義行は除外（本文で表示するため）
// - 他ファイルの ClassDef は除外
// - ambiguousFiles（同名シンボルを定義するファイル）内の参照は全て除外
//
// AST-only では型解決ができないため、同名定義のあるファイル内の call/ref が
// どのシンボルを指すか判定できない。安全側に倒して除外する。
func filterRefsByCandidate(refs []Reference, cand SymbolCandidate, ambiguousFiles map[string]bool) []Reference {
	var filtered []Reference
	for _, ref := range refs {
		// 候補自身の定義行は除外（定義行は本文で表示するため）
		if ref.File == cand.File && ref.Line >= cand.Line && ref.Line <= cand.EndLine {
			continue
		}
		// 同名シンボルを定義するファイル内の参照は除外（誤帰属防止）
		if ambiguousFiles[ref.File] {
			continue
		}
		// 他ファイルの ClassDef は除外（同名シンボルの定義行自体）
		if ref.Class == ast.ClassDef && ref.File != cand.File {
			continue
		}
		filtered = append(filtered, ref)
	}
	return filtered
}

// formatMultipleCandidates は複数候補の一覧を整形する。
func formatMultipleCandidates(symbol string, candidates []SymbolCandidate) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "Multiple symbols matched %q:\n", symbol)
	for i, c := range candidates {
		fmt.Fprintf(&sb, "  %d. %-40s %s %s (L%d-L%d)\n",
			i+1, c.File, c.Kind, c.Name, c.Line, c.EndLine)
	}
	sb.WriteString("\nRefine with path to disambiguate.")
	return sb.String()
}

// formatInspectResult は単一候補の結果を整形する。
func formatInspectResult(r InspectResult) string {
	if r.Symbol == nil {
		return ""
	}

	var sb strings.Builder
	s := r.Symbol

	// ヘッダー
	fmt.Fprintf(&sb, "── %s %s (L%d-L%d) in %s ──\n",
		s.Kind, s.Name, s.Line, s.EndLine, s.File)

	// 本文
	for _, line := range r.Body {
		sb.WriteString(line)
		sb.WriteByte('\n')
	}

	// Callers
	if len(r.Callers) > 0 {
		if r.MoreCallers {
			fmt.Fprintf(&sb, "\nCallers: %d examples (of %d found)\n", len(r.Callers), r.TotalCallers)
		} else {
			fmt.Fprintf(&sb, "\nCallers (%d):\n", len(r.Callers))
		}
		for _, c := range r.Callers {
			scope := ""
			if c.Scope != "" && c.Scope != "package-level" {
				scope = " in " + c.Scope
			}
			fmt.Fprintf(&sb, "  - %s:%d%s\n", c.File, c.Line, scope)
		}
		if r.MoreCallers {
			sb.WriteString("  (+ more callers. Use mode=\"full\" or search_code)\n")
		}
	}

	// References
	if len(r.Refs) > 0 {
		if r.MoreRefs {
			fmt.Fprintf(&sb, "\nReferences: %d examples (of %d found)\n", len(r.Refs), r.TotalRefs)
		} else {
			fmt.Fprintf(&sb, "\nReferences (%d):\n", len(r.Refs))
		}
		for _, ref := range r.Refs {
			label := ""
			if ref.IsTest {
				label = " [test]"
			}
			fmt.Fprintf(&sb, "  - %s:%d | %s%s\n", ref.File, ref.Line, strings.TrimSpace(ref.Snippet), label)
		}
		if r.MoreRefs {
			sb.WriteString("  (+ more references. Use mode=\"full\" or search_code)\n")
		}
	}

	// Related tests
	if len(r.Tests) > 0 {
		if r.MoreTests {
			fmt.Fprintf(&sb, "\nRelated tests: %d examples (of %d found)\n", len(r.Tests), r.TotalTests)
		} else {
			fmt.Fprintf(&sb, "\nRelated tests (%d):\n", len(r.Tests))
		}
		for _, t := range r.Tests {
			if t.Name == "test-file" {
				fmt.Fprintf(&sb, "  - %s (has related tests)\n", t.File)
			} else {
				fmt.Fprintf(&sb, "  - %s:%d | func %s\n", t.File, t.Line, t.Name)
			}
		}
		if r.MoreTests {
			sb.WriteString("  (+ more tests. Use mode=\"full\" or search_code)\n")
		}
	}

	// 打ち切り情報（コンパクトな注記）
	if r.UpstreamIncomplete {
		sb.WriteString("\nWarning: Upstream search may be incomplete due to errors. Use narrower search scopes.\n")
	} else if r.UpstreamTruncated {
		sb.WriteString("\nNote: Some results were truncated upstream. For comprehensive search, use search_code.\n")
	}

	return sb.String()
}
