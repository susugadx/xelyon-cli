package search

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/susugadx/xelyon-cli/internal/locator"
	"github.com/susugadx/xelyon-cli/internal/repomap"
	"github.com/susugadx/xelyon-cli/internal/tools/common"
)

// shouldTrySymbolResolve は symbol 自動解決を試みるべきか判定する。
func shouldTrySymbolResolve(pattern string, opts SearchOptions) bool {
	// 明示的に regex 指定されている場合はスキップ
	if opts.IsRegex {
		return false
	}
	if opts.FilePattern != "" {
		return false // glob 指定は text search
	}
	if !looksLikeIdentifier(pattern) {
		return false
	}

	// Go: AST ベースの正確な解決（Phase 1）
	if opts.FileType == "" || opts.FileType == "go" {
		return true
	}

	// 他言語: signaturePatterns に対応している言語なら試みる
	return isSymbolResolvableLanguage(opts.FileType)
}

// looksLikeIdentifier は文字列がシンボル名に見えるか判定する。
// 例: "NewAgent", "Config.Build", "(*Config).Build", "authenticate", "UserService"
func looksLikeIdentifier(s string) bool {
	if s == "" {
		return false
	}

	// regex メタ文字を含む → identifier ではない
	if containsRegexMeta(s) {
		return false
	}

	// 空白を含む → identifier ではない
	if strings.ContainsAny(s, " \t\n") {
		return false
	}

	// ".*" や ".+" は regex パターン
	if strings.Contains(s, ".*") || strings.Contains(s, ".+") {
		return false
	}

	// "(" を含むが "(*T).Method" 形式でない → 関数呼び出しパターン
	if strings.Contains(s, "(") && !strings.HasPrefix(s, "(*") {
		return false
	}

	// 残りは英数字 + _ + . + * + ( + ) のみで構成される
	for _, r := range s {
		if !isIdentRune(r) && r != '.' && r != '*' && r != '(' && r != ')' {
			return false
		}
	}

	return true
}

func isIdentRune(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_'
}

// containsRegexMeta は regex メタ文字を含むか判定する。
// '.', '*', '(', ')' はシンボルで使われるため除外する。
func containsRegexMeta(s string) bool {
	return strings.ContainsAny(s, "+?[]{}|\\^$")
}

// isSymbolResolvableLanguage は指定言語でシンボル解決が可能か返す。
func isSymbolResolvableLanguage(fileType string) bool {
	switch fileType {
	case "py", "python":
		return true
	case "ts", "tsx", "js", "jsx", "mjs":
		return true
	case "rs", "rust":
		return true
	case "java", "kt", "kts", "kotlin":
		return true
	case "cs", "csharp":
		return true
	case "rb", "ruby":
		return true
	case "php":
		return true
	case "c", "cpp", "cc", "h", "hpp":
		return true
	case "swift":
		return true
	case "scala":
		return true
	case "ex", "exs", "elixir":
		return true
	case "lua":
		return true
	case "sh", "bash", "zsh":
		return true
	default:
		return false
	}
}

// resolveLanguage は SearchOptions から正規化された言語名を返す。
func resolveLanguage(opts SearchOptions) string {
	if opts.FileType == "" {
		return "go"
	}
	switch opts.FileType {
	case "py", "python":
		return "python"
	case "ts", "tsx", "js", "jsx", "mjs":
		return "js"
	case "rs", "rust":
		return "rust"
	case "java", "kt", "kts", "kotlin":
		return "java"
	case "cs", "csharp":
		return "csharp"
	case "php":
		return "php"
	case "rb", "ruby":
		return "ruby"
	case "c", "cpp", "cc", "h", "hpp":
		return "cpp"
	case "swift":
		return "swift"
	case "scala":
		return "scala"
	case "ex", "exs", "elixir":
		return "elixir"
	case "lua":
		return "lua"
	default:
		return opts.FileType
	}
}

// ── 多言語シンボル解決 ──

type genericSymbolStatus string

const (
	genericSymbolSingle   genericSymbolStatus = "single"
	genericSymbolMultiple genericSymbolStatus = "multiple"
	genericSymbolNone     genericSymbolStatus = "none"
)

// genericSymbolDef は多言語のシンボル定義候補。
type genericSymbolDef struct {
	Name      string
	Kind      string
	File      string
	Line      int
	Signature string
}

// genericSymbolRef は参照箇所。
type genericSymbolRef struct {
	File    string
	Line    int
	Snippet string
	IsTest  bool
}

// resolveGenericSymbol は Go 以外の言語でシンボル解決を試みる。
func resolveGenericSymbol(symbol string, opts SearchOptions) (string, genericSymbolStatus) {
	// Step 1: 定義を見つける（signaturePatterns を使う）
	defs := findGenericDefinitions(symbol, opts)
	if len(defs) == 0 {
		return "", genericSymbolNone
	}

	// Step 2: 複数候補
	if len(defs) > 1 {
		return formatGenericMultipleDefs(symbol, defs, opts.LocatorRegistry), genericSymbolMultiple
	}

	// Step 3: 単一候補 → 参照箇所を検索
	def := defs[0]
	refs := findGenericReferences(symbol, opts)
	filteredRefs := filterGenericRefs(refs, def)

	// テスト分離
	var normalRefs, testRefs []genericSymbolRef
	for _, ref := range filteredRefs {
		if ref.IsTest {
			testRefs = append(testRefs, ref)
		} else {
			normalRefs = append(normalRefs, ref)
		}
	}

	return formatGenericSymbolResult(def, normalRefs, testRefs, opts.LocatorRegistry), genericSymbolSingle
}

// findGenericDefinitions は ripgrep + signaturePatterns でシンボルの定義行を見つける。
func findGenericDefinitions(symbol string, opts SearchOptions) []genericSymbolDef {
	if !common.IsRipgrepAvailable() {
		return nil
	}

	args := buildGenericRgArgs(symbol, opts)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, common.RipgrepPath(), args...)
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	_ = cmd.Run()

	lang := resolvePatternLang(opts.FileType)

	var defs []genericSymbolDef
	scanner := bufio.NewScanner(&stdout)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, ":", 3)
		if len(parts) < 3 {
			continue
		}
		file := parts[0]
		lineNum, err := strconv.Atoi(parts[1])
		if err != nil {
			continue
		}
		content := strings.TrimSpace(parts[2])

		// 言語フィルタ付きで定義行かチェック（クロス言語偽陽性を防止）
		name, kind, ok := repomap.ExtractSignatureMetadataForLang(content, lang)
		if !ok || name != symbol {
			continue
		}

		defs = append(defs, genericSymbolDef{
			Name:      name,
			Kind:      kind,
			File:      file,
			Line:      lineNum,
			Signature: content,
		})
	}

	return defs
}

const maxGenericRefs = 50

// findGenericReferences は ripgrep -w でシンボルの参照箇所を見つける。
func findGenericReferences(symbol string, opts SearchOptions) []genericSymbolRef {
	if !common.IsRipgrepAvailable() {
		return nil
	}

	args := buildGenericRgArgs(symbol, opts)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, common.RipgrepPath(), args...)
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	_ = cmd.Run()

	var refs []genericSymbolRef
	scanner := bufio.NewScanner(&stdout)
	for scanner.Scan() && len(refs) < maxGenericRefs {
		line := scanner.Text()
		parts := strings.SplitN(line, ":", 3)
		if len(parts) < 3 {
			continue
		}
		file := parts[0]
		lineNum, err := strconv.Atoi(parts[1])
		if err != nil {
			continue
		}
		refs = append(refs, genericSymbolRef{
			File:    file,
			Line:    lineNum,
			Snippet: strings.TrimSpace(parts[2]),
			IsTest:  repomap.IsTestFile(file),
		})
	}

	return refs
}

// buildGenericRgArgs は多言語シンボル検索用の ripgrep 引数を構築する。
func buildGenericRgArgs(symbol string, opts SearchOptions) []string {
	args := []string{
		"-n", "--no-heading", "--color", "never",
		"-w",
	}
	if opts.FileType != "" {
		args = append(args, "--type", normalizeRgType(opts.FileType))
	}
	searchPath := "."
	if opts.Path != "" {
		searchPath = opts.Path
	}
	args = append(args, symbol, searchPath)
	return args
}

// resolvePatternLang は FileType を signaturePattern の lang に変換する。
func resolvePatternLang(fileType string) string {
	switch fileType {
	case "go":
		return "go"
	case "py", "python":
		return "py"
	case "ts", "tsx", "js", "jsx", "mjs", "typescript", "javascript":
		return "js"
	case "rs", "rust":
		return "rs"
	case "java", "kt", "kts", "kotlin":
		return "java"
	case "cs", "csharp":
		return "csharp"
	case "rb", "ruby":
		return "rb"
	case "php":
		return "php"
	case "c", "cpp", "cc", "h", "hpp":
		return "c"
	case "swift":
		return "swift"
	case "scala":
		return "scala"
	case "ex", "exs", "elixir":
		return "elixir"
	case "lua":
		return "lua"
	case "sh", "bash", "zsh":
		return "sh"
	default:
		return ""
	}
}

func filterGenericRefs(refs []genericSymbolRef, def genericSymbolDef) []genericSymbolRef {
	var filtered []genericSymbolRef
	for _, ref := range refs {
		if ref.File == def.File && ref.Line == def.Line {
			continue
		}
		filtered = append(filtered, ref)
	}
	return filtered
}

func normalizeRgType(fileType string) string {
	switch fileType {
	case "python":
		return "py"
	case "typescript", "tsx", "jsx", "mjs":
		return "ts"
	case "rs", "rust":
		return "rust"
	case "kt", "kotlin", "kts":
		return "kotlin"
	case "cs", "csharp":
		return "csharp"
	case "cc", "hpp":
		return "cpp"
	case "ex", "exs", "elixir":
		return "elixir"
	case "rb", "ruby":
		return "ruby"
	default:
		return fileType
	}
}

// ── フォーマッター ──

func formatGenericMultipleDefs(symbol string, defs []genericSymbolDef, reg *locator.Registry) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "Multiple definitions found for %q:\n", symbol)
	for i, d := range defs {
		line := fmt.Sprintf("  %d. %s %s (L%d) in %s", i+1, d.Kind, d.Name, d.Line, d.File)
		if reg != nil {
			id := reg.Register(locator.Location{FilePath: d.File, Line: d.Line, Name: fmt.Sprintf("%s %s", d.Kind, d.Name)})
			line += " " + id
		}
		fmt.Fprintf(&sb, "%s\n", line)
	}
	sb.WriteString("\nRefine with path to disambiguate (e.g. path=\"src/models/\").")
	return sb.String()
}

const (
	genericRefLimit  = 15
	genericTestLimit = 5
)

func formatGenericSymbolResult(def genericSymbolDef, refs, tests []genericSymbolRef, reg *locator.Registry) string {
	var sb strings.Builder

	header := fmt.Sprintf("── %s %s (L%d) in %s", def.Kind, def.Name, def.Line, def.File)
	if reg != nil {
		id := reg.Register(locator.Location{FilePath: def.File, Line: def.Line, Name: fmt.Sprintf("%s %s", def.Kind, def.Name)})
		header += " " + id
	}
	fmt.Fprintf(&sb, "%s ──\n", header)
	fmt.Fprintf(&sb, "%d: %s\n", def.Line, def.Signature)

	if len(refs) > 0 {
		sb.WriteString("\n── References ──\n")
		for i, ref := range refs {
			if i >= genericRefLimit {
				fmt.Fprintf(&sb, "  ... (%d more references)\n", len(refs)-genericRefLimit)
				break
			}
			line := fmt.Sprintf("  %s:%d  %s", ref.File, ref.Line, ref.Snippet)
			if reg != nil {
				id := reg.Register(locator.Location{FilePath: ref.File, Line: ref.Line})
				line += " " + id
			}
			fmt.Fprintf(&sb, "%s\n", line)
		}
	}

	if len(tests) > 0 {
		sb.WriteString("\n── Related Tests ──\n")
		for i, test := range tests {
			if i >= genericTestLimit {
				fmt.Fprintf(&sb, "  ... (%d more tests)\n", len(tests)-genericTestLimit)
				break
			}
			line := fmt.Sprintf("  %s:%d  %s", test.File, test.Line, test.Snippet)
			if reg != nil {
				id := reg.Register(locator.Location{FilePath: test.File, Line: test.Line})
				line += " " + id
			}
			fmt.Fprintf(&sb, "%s\n", line)
		}
	}

	if len(refs) == 0 && len(tests) == 0 {
		sb.WriteString("\nNo references found.\n")
	}

	return sb.String()
}
