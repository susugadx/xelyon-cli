package lsp

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	lsplib "github.com/susugadx/xelyon-cli/internal/lsp"
	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/tools/common"
)

// ===== lsp_find Tool =====

// LSPFindTool はシンボル名で LSP を叩けるラッパーツール。
// AI が行番号を間違える問題を解決するため、grep でシンボル位置を特定してから LSP を呼ぶ。
type LSPFindTool struct{}

func (t *LSPFindTool) Name() string { return "lsp_find" }

func (t *LSPFindTool) Description() string {
	return tools.ToolDescriptions[t.Name()]
}

func (t *LSPFindTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"symbol": map[string]interface{}{
				"type":        "string",
				"description": "Symbol name to find (function, type, variable, constant name)",
			},
			"action": map[string]interface{}{
				"type":        "string",
				"description": "LSP action: 'definition' (default), 'references', or 'implementations'",
				"enum":        []string{"definition", "references", "implementations"},
			},
			"scope": map[string]interface{}{
				"type":        "string",
				"description": "Directory to search in (default: project root '.')",
			},
		},
		"required":             []string{"symbol"},
		"additionalProperties": false,
	}
}

func (t *LSPFindTool) Run(args map[string]string) (string, *tools.FileChange, error) {
	symbol := args["symbol"]
	if symbol == "" {
		return "Error: symbol is required", nil, fmt.Errorf("symbol is required")
	}

	action := args["action"]
	if action == "" {
		action = "definition"
	}
	if action != "definition" && action != "references" && action != "implementations" {
		return "Error: action must be 'definition', 'references', or 'implementations'", nil, fmt.Errorf("invalid action: %s", action)
	}

	scope := args["scope"]
	if scope == "" {
		scope = "."
	}

	return executeLSPFind(symbol, action, scope)
}

// symbolMatch はシンボル検索のマッチ結果を表す
type symbolMatch struct {
	FilePath  string
	Line      int
	Column    int // シンボルのカラム位置（1-indexed）
	Content   string
	MatchType matchType
}

// matchType はマッチの種別（定義サイト優先に使う）
type matchType int

const (
	matchDefinition matchType = iota // func, type, var, const 定義
	matchAssignment                  // := 代入
	matchUsage                       // その他の使用箇所
)

// executeLSPFind はシンボル名から LSP アクションを実行する。
func executeLSPFind(symbol, action, scope string) (string, *tools.FileChange, error) {
	// 1. grep でシンボルの位置を特定
	matches := findSymbolLocations(symbol, scope)
	if len(matches) == 0 {
		return fmt.Sprintf("Symbol '%s' not found in %s", symbol, scope), nil, nil
	}

	// 2. 定義箇所を優先してソート
	sortMatchesByPriority(matches)

	// 3. LSP が使えない場合は grep 結果だけ返す（フォールバック）
	if LSPClient == nil {
		return formatGrepFallback(symbol, action, matches), nil, nil
	}

	// 4. 最も確度の高い候補で LSP を呼ぶ
	best := matches[0]

	absPath, err := common.ValidatePath(best.FilePath)
	if err != nil {
		return fmt.Sprintf("Error: %v", err), nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), LSPToolTimeout)
	defer cancel()

	switch action {
	case "definition":
		return executeLSPDefinitionForSymbol(ctx, absPath, best, symbol, matches)
	case "references":
		return executeLSPReferencesForSymbol(ctx, absPath, best, symbol, matches)
	case "implementations":
		// implementations は references で代用（Go の LSP は textDocument/implementation をサポート）
		return executeLSPReferencesForSymbol(ctx, absPath, best, symbol, matches)
	}

	return "Error: unexpected action", nil, fmt.Errorf("unexpected action: %s", action)
}

// findSymbolLocations は grep でシンボルの出現箇所を全て見つける。
func findSymbolLocations(symbol, scope string) []symbolMatch {
	// シンボル名を単語境界で検索（正規表現）
	pattern := `\b` + regexp.QuoteMeta(symbol) + `\b`

	cmd := exec.Command("grep", "-rn", "-I",
		"--include=*.go", "--include=*.js", "--include=*.ts",
		"--include=*.py", "--include=*.rs", "--include=*.java",
		"-E", pattern, scope)

	output, err := cmd.CombinedOutput()
	if err != nil && len(output) == 0 {
		return nil
	}

	var matches []symbolMatch
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := scanner.Text()
		m := parseGrepLine(line, symbol)
		if m != nil {
			matches = append(matches, *m)
		}
	}

	return matches
}

// parseGrepLine は grep 出力の1行をパースする。
// 形式: filepath:linenum:content
func parseGrepLine(line, symbol string) *symbolMatch {
	// filepath:linenum:content の形式をパース
	// Windows パス対策: 最初のコロンの後に数字が来るパターンを探す
	parts := strings.SplitN(line, ":", 3)
	if len(parts) < 3 {
		return nil
	}

	filePath := parts[0]
	lineNum, err := strconv.Atoi(parts[1])
	if err != nil {
		return nil
	}
	content := parts[2]

	// シンボルのカラム位置を特定（1-indexed）
	col := findSymbolColumn(content, symbol)
	if col == 0 {
		col = 1 // フォールバック
	}

	return &symbolMatch{
		FilePath:  filePath,
		Line:      lineNum,
		Column:    col,
		Content:   strings.TrimSpace(content),
		MatchType: classifyMatch(content, symbol),
	}
}

// findSymbolColumn はコンテンツ内のシンボルの正確なカラム位置を返す（1-indexed）。
func findSymbolColumn(content, symbol string) int {
	// 単語境界を考慮して検索
	re, err := regexp.Compile(`\b` + regexp.QuoteMeta(symbol) + `\b`)
	if err != nil {
		return strings.Index(content, symbol) + 1
	}
	loc := re.FindStringIndex(content)
	if loc == nil {
		return strings.Index(content, symbol) + 1
	}
	return loc[0] + 1 // 1-indexed
}

// classifyMatch はマッチ行の種別を判定する。
func classifyMatch(content, symbol string) matchType {
	trimmed := strings.TrimSpace(content)

	// Go の定義パターン
	goDefPatterns := []string{
		`func ` + regexp.QuoteMeta(symbol) + `\b`,
		`func \(.*\) ` + regexp.QuoteMeta(symbol) + `\b`,
		`type ` + regexp.QuoteMeta(symbol) + `\b`,
		`var ` + regexp.QuoteMeta(symbol) + `\b`,
		`const ` + regexp.QuoteMeta(symbol) + `\b`,
	}

	for _, p := range goDefPatterns {
		if matched, _ := regexp.MatchString(p, trimmed); matched {
			return matchDefinition
		}
	}

	// JS/TS の定義パターン
	jsDefPatterns := []string{
		`function ` + regexp.QuoteMeta(symbol) + `\b`,
		`class ` + regexp.QuoteMeta(symbol) + `\b`,
		`interface ` + regexp.QuoteMeta(symbol) + `\b`,
		`const ` + regexp.QuoteMeta(symbol) + `\b`,
		`let ` + regexp.QuoteMeta(symbol) + `\b`,
		`export `,
	}

	for _, p := range jsDefPatterns {
		if matched, _ := regexp.MatchString(p, trimmed); matched {
			return matchDefinition
		}
	}

	// Python の定義パターン
	pyDefPatterns := []string{
		`def ` + regexp.QuoteMeta(symbol) + `\b`,
		`class ` + regexp.QuoteMeta(symbol) + `\b`,
	}

	for _, p := range pyDefPatterns {
		if matched, _ := regexp.MatchString(p, trimmed); matched {
			return matchDefinition
		}
	}

	// 代入パターン
	if matched, _ := regexp.MatchString(regexp.QuoteMeta(symbol)+`\s*:=`, trimmed); matched {
		return matchAssignment
	}
	if matched, _ := regexp.MatchString(regexp.QuoteMeta(symbol)+`\s*=\s`, trimmed); matched {
		return matchAssignment
	}

	return matchUsage
}

// sortMatchesByPriority は定義サイトを優先してソートする。
func sortMatchesByPriority(matches []symbolMatch) {
	sort.SliceStable(matches, func(i, j int) bool {
		// 定義 > 代入 > 使用
		if matches[i].MatchType != matches[j].MatchType {
			return matches[i].MatchType < matches[j].MatchType
		}
		// テストファイルは後回し
		iTest := isTestFile(matches[i].FilePath)
		jTest := isTestFile(matches[j].FilePath)
		if iTest != jTest {
			return !iTest
		}
		return false
	})
}

// isTestFile はテストファイルかどうか判定する。
func isTestFile(path string) bool {
	base := filepath.Base(path)
	return strings.HasSuffix(base, "_test.go") ||
		strings.HasSuffix(base, ".test.js") ||
		strings.HasSuffix(base, ".test.ts") ||
		strings.HasSuffix(base, ".spec.js") ||
		strings.HasSuffix(base, ".spec.ts") ||
		strings.Contains(base, "test_")
}

// executeLSPDefinitionForSymbol は LSP definition を実行する。
func executeLSPDefinitionForSymbol(ctx context.Context, absPath string, best symbolMatch, symbol string, allMatches []symbolMatch) (string, *tools.FileChange, error) {
	locations, err := LSPClient.GoToDefinition(ctx, absPath, best.Line, best.Column)
	if err != nil {
		// LSP 失敗 → grep フォールバック
		return formatGrepFallback(symbol, "definition", allMatches), nil, nil
	}

	if len(locations) == 0 {
		// 候補が複数ある場合、別の候補で再試行
		for _, m := range allMatches[1:] {
			altPath, altErr := common.ValidatePath(m.FilePath)
			if altErr != nil {
				continue
			}
			locations, err = LSPClient.GoToDefinition(ctx, altPath, m.Line, m.Column)
			if err == nil && len(locations) > 0 {
				break
			}
		}
	}

	if len(locations) == 0 {
		return formatGrepFallback(symbol, "definition", allMatches), nil, nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Definition of '%s':\n\n", symbol))
	for _, loc := range locations {
		filePath := lsplib.URIToFile(loc.URI)
		sb.WriteString(fmt.Sprintf("  %s:%d:%d\n", filePath, loc.Range.Start.Line+1, loc.Range.Start.Character+1))

		// 定義行の内容を読んで表示
		lineContent := readFileLine(filePath, loc.Range.Start.Line+1)
		if lineContent != "" {
			sb.WriteString(fmt.Sprintf("  → %s\n", strings.TrimSpace(lineContent)))
		}
	}

	green.Printf("🔍 lsp_find: Found definition of '%s'\n", symbol)
	return sb.String(), nil, nil
}

// executeLSPReferencesForSymbol は LSP references を実行する。
func executeLSPReferencesForSymbol(ctx context.Context, absPath string, best symbolMatch, symbol string, allMatches []symbolMatch) (string, *tools.FileChange, error) {
	locations, err := LSPClient.FindReferences(ctx, absPath, best.Line, best.Column, true)
	if err != nil {
		return formatGrepFallback(symbol, "references", allMatches), nil, nil
	}

	if len(locations) == 0 {
		// 別の候補で再試行
		for _, m := range allMatches[1:] {
			altPath, altErr := common.ValidatePath(m.FilePath)
			if altErr != nil {
				continue
			}
			locations, err = LSPClient.FindReferences(ctx, altPath, m.Line, m.Column, true)
			if err == nil && len(locations) > 0 {
				break
			}
		}
	}

	if len(locations) == 0 {
		return formatGrepFallback(symbol, "references", allMatches), nil, nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Found %d references for '%s':\n\n", len(locations), symbol))
	for i, loc := range locations {
		if i >= 50 {
			sb.WriteString(fmt.Sprintf("\n... and %d more\n", len(locations)-50))
			break
		}
		filePath := lsplib.URIToFile(loc.URI)
		sb.WriteString(fmt.Sprintf("  %s:%d:%d\n", filePath, loc.Range.Start.Line+1, loc.Range.Start.Character+1))
	}

	green.Printf("🔍 lsp_find: Found %d references for '%s'\n", len(locations), symbol)
	return sb.String(), nil, nil
}

// formatGrepFallback は LSP 未起動・失敗時の grep ベースフォールバック。
func formatGrepFallback(symbol, action string, matches []symbolMatch) string {
	var sb strings.Builder

	if LSPClient == nil {
		sb.WriteString(fmt.Sprintf("LSP not available. Showing grep results for '%s' (%s):\n\n", symbol, action))
	} else {
		sb.WriteString(fmt.Sprintf("LSP could not resolve '%s'. Showing grep results (%s):\n\n", symbol, action))
	}

	// 定義候補を先に表示
	defs := 0
	for _, m := range matches {
		if m.MatchType == matchDefinition {
			sb.WriteString(fmt.Sprintf("  [DEF] %s:%d:%d  %s\n", m.FilePath, m.Line, m.Column, m.Content))
			defs++
		}
	}

	if defs > 0 {
		sb.WriteString("\nOther occurrences:\n")
	}

	shown := 0
	for _, m := range matches {
		if m.MatchType == matchDefinition {
			continue
		}
		if shown >= 30 {
			remaining := len(matches) - defs - shown
			if remaining > 0 {
				sb.WriteString(fmt.Sprintf("\n... and %d more occurrences\n", remaining))
			}
			break
		}
		sb.WriteString(fmt.Sprintf("  %s:%d:%d  %s\n", m.FilePath, m.Line, m.Column, m.Content))
		shown++
	}

	return sb.String()
}

// readFileLine はファイルの指定行を読み取る。
func readFileLine(filePath string, lineNum int) string {
	f, err := os.Open(filePath)
	if err != nil {
		return ""
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	current := 0
	for scanner.Scan() {
		current++
		if current == lineNum {
			return scanner.Text()
		}
	}
	return ""
}
