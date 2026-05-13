package search

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/navigation"
)

type symbolBundleSectionInput struct {
	Kind   string
	Title  string
	Items  []genericSymbolRef
	Limit  int
	IsTest bool
}

type goSymbolBundleBuildOptions struct {
	implementationLimit int
	impact              *SymbolBundleImpact
}

var searchCodeGoSymbolBudget = navigation.Budget{
	BodyLines:   18,
	CallerLimit: 3,
	RefLimit:    3,
	TestLimit:   2,
}

const goImplementationLimit = 4
const goSymbolBundleLSPSource = "gopls"

func buildGoSymbolBundle(query string, result navigation.InspectResult) *SymbolBundle {
	return buildGoSymbolBundleWithOptions(query, result, goSymbolBundleBuildOptions{
		implementationLimit: goImplementationLimit,
	})
}

func buildGoSymbolBundleWithOptions(query string, result navigation.InspectResult, opts goSymbolBundleBuildOptions) *SymbolBundle {
	if result.Symbol == nil {
		return nil
	}
	if opts.implementationLimit <= 0 {
		opts.implementationLimit = goImplementationLimit
	}

	bundle := newGoSymbolBundle(query, result, opts)

	if len(result.Body) > 0 {
		bundle.Definition.Signature = result.Body[0]
	}

	addNavigationSection(bundle, "callers", "Callers", result.Callers, result.TotalCallers, result.MoreCallers)
	addNavigationSection(bundle, "references", "References", result.Refs, result.TotalRefs, result.MoreRefs)
	addNavigationTestSection(bundle, result.Tests, result.TotalTests, result.MoreTests)
	appendGoImplementationSection(bundle, result.Implementations, opts.implementationLimit)

	return bundle
}

func newGoSymbolBundle(query string, result navigation.InspectResult, opts goSymbolBundleBuildOptions) *SymbolBundle {
	displayName := goSymbolBundleDisplayName(*result.Symbol)
	return &SymbolBundle{
		Identity: SymbolBundleIdentity{
			Language:    "go",
			Query:       query,
			Canonical:   canonicalGoSymbolBundleKey(*result.Symbol),
			DisplayName: displayName,
			Kind:        result.Symbol.Kind,
			File:        result.Symbol.File,
			Line:        result.Symbol.Line,
			EndLine:     result.Symbol.EndLine,
		},
		Diagnostics: SymbolBundleDiagnostics{
			ResolvedViaLSP:     result.ResolvedViaLSP,
			LSPSource:          goSymbolBundleLSPSource,
			UpstreamTruncated:  result.UpstreamTruncated,
			UpstreamIncomplete: result.UpstreamIncomplete,
		},
		Definition: SymbolBundleDefinition{
			File:    result.Symbol.File,
			Line:    result.Symbol.Line,
			EndLine: result.Symbol.EndLine,
			Body:    append([]string(nil), result.Body...),
		},
		Impact: cloneSymbolBundleImpact(opts.impact),
		Debug: SymbolBundleDebug{
			Source:       "go-inspect",
			FileRootPath: result.Symbol.RootPath,
		},
	}
}

func goSymbolBundleDisplayName(symbol navigation.SymbolCandidate) string {
	if symbol.Kind != "method" || symbol.Receiver == "" {
		return symbol.Name
	}
	if strings.HasPrefix(symbol.Receiver, "*") {
		return fmt.Sprintf("(%s).%s", symbol.Receiver, symbol.Name)
	}
	return symbol.Receiver + "." + symbol.Name
}

func appendGoImplementationSection(bundle *SymbolBundle, impls []navigation.ImplementationRef, implementationLimit int) {
	if len(impls) == 0 || implementationLimit <= 0 {
		return
	}

	limit := min(implementationLimit, len(impls))
	items := make([]SymbolBundleItem, 0, limit)
	for _, impl := range impls[:limit] {
		items = append(items, SymbolBundleItem{
			Kind:         "implementations",
			File:         impl.File,
			ResolvedPath: impl.ResolvedPath,
			Line:         impl.Line,
			Snippet:      strings.TrimSpace(impl.Name),
			Name:         impl.Name,
		})
	}

	bundle.Sections = append(bundle.Sections, SymbolBundleSection{
		Kind:  "implementations",
		Title: "Related Implementations",
		Items: items,
		Total: len(impls),
		More:  len(impls) > len(items),
	})
}

func addNavigationSection(bundle *SymbolBundle, kind, title string, refs []navigation.Reference, total int, more bool) {
	if len(refs) == 0 {
		return
	}
	items := make([]SymbolBundleItem, 0, len(refs))
	for _, ref := range refs {
		snippet := strings.TrimSpace(ref.Snippet)
		if snippet == "" && ref.Scope != "" {
			snippet = ref.Scope
		}
		items = append(items, SymbolBundleItem{
			Kind:         kind,
			File:         ref.File,
			ResolvedPath: ref.ResolvedPath,
			Line:         ref.Line,
			Snippet:      snippet,
			Scope:        ref.Scope,
			IsTest:       ref.IsTest,
		})
	}
	bundle.Sections = append(bundle.Sections, SymbolBundleSection{
		Kind:  kind,
		Title: title,
		Items: items,
		Total: total,
		More:  more,
	})
}

func addNavigationTestSection(bundle *SymbolBundle, tests []navigation.TestRef, total int, more bool) {
	if len(tests) == 0 {
		return
	}
	items := make([]SymbolBundleItem, 0, len(tests))
	for _, test := range tests {
		items = append(items, SymbolBundleItem{
			Kind:         "tests",
			File:         test.File,
			ResolvedPath: test.ResolvedPath,
			Line:         test.Line,
			Name:         test.Name,
			IsTest:       true,
		})
	}
	bundle.Sections = append(bundle.Sections, SymbolBundleSection{
		Kind:  "tests",
		Title: "Related Tests",
		Items: items,
		Total: total,
		More:  more,
	})
}

func buildGenericSymbolBundle(lang, query string, def genericSymbolDef, body []string, inputs []symbolBundleSectionInput) *SymbolBundle {
	displayName := def.Name
	if displayName == "" {
		displayName = query
	}

	bundle := &SymbolBundle{
		Identity: SymbolBundleIdentity{
			Language:    lang,
			Query:       query,
			Canonical:   canonicalSymbolBundleKey(lang, def.File, def.Line, displayName),
			DisplayName: displayName,
			Kind:        def.Kind,
			File:        def.File,
			Line:        def.Line,
			EndLine:     def.Line,
		},
		Definition: SymbolBundleDefinition{
			File:      def.File,
			Line:      def.Line,
			EndLine:   def.Line,
			Signature: def.Signature,
			Body:      append([]string(nil), body...),
		},
		Debug: SymbolBundleDebug{Source: "generic-resolver"},
	}

	for _, input := range inputs {
		section := buildGenericBundleSection(def, input)
		if section != nil {
			bundle.Sections = append(bundle.Sections, *section)
		}
	}

	return bundle
}

func stableGoSymbolBundleKey(packageDir, receiverNorm, name, kind, signature string) string {
	sigHash := sha256.Sum256([]byte(strings.TrimSpace(signature)))
	return fmt.Sprintf("%s|%s|%s|%s|%s|%s",
		"go",
		filepath.ToSlash(filepath.Clean(packageDir)),
		strings.TrimSpace(receiverNorm),
		strings.TrimSpace(name),
		strings.TrimSpace(kind),
		hex.EncodeToString(sigHash[:8]),
	)
}

func canonicalGoSymbolBundleKey(symbol navigation.SymbolCandidate) string {
	key := strings.TrimSpace(symbol.StableKey)
	if key == "" {
		key = stableGoSymbolBundleKey(symbol.PackageDir, symbol.ReceiverNorm, symbol.Name, symbol.Kind, symbol.Signature)
	}
	if symbol.StableKeyCollision && strings.TrimSpace(symbol.File) != "" {
		return key + "|file=" + filepath.ToSlash(filepath.Clean(symbol.File))
	}
	return key
}

func canonicalSymbolBundleKey(lang, file string, line int, displayName string) string {
	return fmt.Sprintf("%s|%s|%d|%s", lang, file, line, displayName)
}

func attachBundleRoute(bundle *SymbolBundle, route searchRouteTrace) *SymbolBundle {
	if bundle == nil {
		return nil
	}
	bundle.Debug.Route = route
	if bundle.Debug.MatchedPatterns == nil && route.SymbolCandidates != nil {
		bundle.Debug.MatchedPatterns = append([]string(nil), route.SymbolCandidates...)
	}
	return bundle
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
