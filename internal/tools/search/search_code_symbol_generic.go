package search

import (
	"fmt"
	"strings"

	codeast "github.com/susugadx/xelyon-cli/internal/ast"
	"github.com/susugadx/xelyon-cli/internal/filefilter"
	"github.com/susugadx/xelyon-cli/internal/locator"
	"github.com/susugadx/xelyon-cli/internal/navigation"
	"github.com/susugadx/xelyon-cli/internal/repomap"
	"github.com/susugadx/xelyon-cli/internal/tools"
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
	Observation   *tools.RuntimeObservation
}

// genericSymbolDef は多言語のシンボル定義候補。
type genericSymbolDef struct {
	Name      string
	Kind      string
	File      string
	Line      int
	Character int
	Signature string
	Exported  bool
}

// genericSymbolRef は参照箇所。
type genericSymbolRef struct {
	File    string
	Line    int
	Snippet string
	IsTest  bool
	Class   codeast.MatchClass
}

type genericSymbolMatch struct {
	File    string
	Line    int
	Content string
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
			Observation:   observationForGenericDefinitions(defs, opts),
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
		Output:      formatSymbolBundle(bundle, opts.LocatorRegistry, nil),
		Status:      genericSymbolSingle,
		Bundle:      bundle,
		Observation: observationForSymbolBundle(bundle, opts),
	}
}

// findGenericDefinitions は ripgrep + signaturePatterns でシンボルの定義行を見つける。
func findGenericDefinitions(symbol string, opts SearchOptions) []genericSymbolDef {
	matches := findGenericSymbolMatches(symbol, opts, 0)
	return genericDefinitionsFromMatches(symbol, opts, matches)
}

func genericDefinitionsFromMatches(symbol string, opts SearchOptions, matches []genericSymbolMatch) []genericSymbolDef {
	lang := resolvePatternLang(filefilter.RepresentativeToken(opts.FileType, opts.FilePattern))

	defs := make([]genericSymbolDef, 0, len(matches))
	for _, match := range matches {
		name, kind, ok := repomap.ExtractSignatureMetadataForLang(match.Content, lang)
		if !ok || name != symbol {
			continue
		}

		defs = append(defs, genericSymbolDef{
			Name:      name,
			Kind:      kind,
			File:      match.File,
			Line:      match.Line,
			Signature: match.Content,
		})
	}

	return defs
}

const maxGenericRefs = 50

// findGenericReferences は ripgrep -w でシンボルの参照箇所を見つける。
func findGenericReferences(symbol string, opts SearchOptions) []genericSymbolRef {
	matches := findGenericSymbolMatches(symbol, opts, maxGenericRefs)
	refs := make([]genericSymbolRef, 0, len(matches))
	for _, match := range matches {
		refs = append(refs, genericSymbolRefFromMatch(match))
	}

	return refs
}

func genericSymbolRefFromMatch(match genericSymbolMatch) genericSymbolRef {
	return genericSymbolRef{
		File:    match.File,
		Line:    match.Line,
		Snippet: match.Content,
		IsTest:  repomap.IsTestFile(match.File),
	}
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
