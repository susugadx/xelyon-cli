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
	"github.com/susugadx/xelyon-cli/internal/navigation"
	"github.com/susugadx/xelyon-cli/internal/repomap"
	"github.com/susugadx/xelyon-cli/internal/tools/common"
)

type genericSymbolStatus string

const (
	genericSymbolSingle   genericSymbolStatus = "single"
	genericSymbolMultiple genericSymbolStatus = "multiple"
	genericSymbolNone     genericSymbolStatus = "none"
)

type genericResolveResult struct {
	Output        string
	Status        genericSymbolStatus
	Bundle        *SymbolBundle
	AffectedFiles []string
}

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
func resolveGenericSymbol(symbol string, opts SearchOptions) genericResolveResult {
	defs := findGenericDefinitions(symbol, opts)
	if len(defs) == 0 {
		return genericResolveResult{Status: genericSymbolNone}
	}

	if len(defs) > 1 {
		return genericResolveResult{
			Output:        formatGenericMultipleDefsWithOptions(symbol, defs, opts.LocatorRegistry, opts),
			Status:        genericSymbolMultiple,
			AffectedFiles: collectGenericDefAffectedFiles(defs, opts),
		}
	}

	def := defs[0]
	refs := findGenericReferences(symbol, opts)
	filteredRefs := filterGenericRefs(refs, def)

	var normalRefs, testRefs []genericSymbolRef
	for _, ref := range filteredRefs {
		if ref.IsTest {
			testRefs = append(testRefs, ref)
		} else {
			normalRefs = append(normalRefs, ref)
		}
	}

	bundle := buildGenericSymbolBundle(resolveLanguage(opts), symbol, def, []string{
		fmt.Sprintf("%d: %s", def.Line, def.Signature),
	}, []symbolBundleSectionInput{
		{Kind: "references", Title: "References", Items: normalRefs, Limit: genericRefLimit},
		{Kind: "tests", Title: "Related Tests", Items: testRefs, Limit: genericTestLimit, IsTest: true},
	})
	bundle.Debug.FileRootPath = invocationCWDOrGetwd(opts)
	return genericResolveResult{
		Output: formatSymbolBundle(bundle, opts.LocatorRegistry, nil),
		Status: genericSymbolSingle,
		Bundle: bundle,
	}
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

func formatGenericMultipleDefsWithOptions(symbol string, defs []genericSymbolDef, reg *locator.Registry, opts SearchOptions) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "Multiple definitions found for %q:\n", symbol)
	for i, d := range defs {
		line := fmt.Sprintf("  %d. %s %s (L%d) in %s", i+1, d.Kind, d.Name, d.Line, d.File)
		if reg != nil {
			id := reg.Register(newTextSearchLocator(d.File, d.Line, 0, fmt.Sprintf("%s %s", d.Kind, d.Name), opts))
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

func formatGenericSymbolResult(bundle *SymbolBundle, reg *locator.Registry) string {
	return formatSymbolBundle(bundle, reg, nil)
}

func collectNavigationCandidatesAffectedFiles(candidates []navigation.SymbolCandidate, opts SearchOptions) []string {
	paths := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		if absPath := absoluteAffectedFilePathForSymbol(candidate.File, opts, candidate.RootPath); absPath != "" {
			paths = append(paths, absPath)
		}
	}
	return dedupePaths(paths)
}

func collectGenericDefAffectedFiles(defs []genericSymbolDef, opts SearchOptions) []string {
	paths := make([]string, 0, len(defs))
	basePath := invocationCWDOrGetwd(opts)
	for _, def := range defs {
		if absPath := absoluteAffectedFilePathWithBase(def.File, basePath); absPath != "" {
			paths = append(paths, absPath)
		}
	}
	return dedupePaths(paths)
}
