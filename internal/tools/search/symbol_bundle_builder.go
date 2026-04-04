package search

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"sort"
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

var searchCodeGoSymbolBudget = navigation.Budget{
	BodyLines:   18,
	CallerLimit: 3,
	RefLimit:    3,
	TestLimit:   2,
}

const goImplementationLimit = 4

func buildGoSymbolBundle(query string, result navigation.InspectResult) *SymbolBundle {
	if result.Symbol == nil {
		return nil
	}

	displayName := result.Symbol.Name
	if result.Symbol.Kind == "method" && result.Symbol.Receiver != "" {
		if strings.HasPrefix(result.Symbol.Receiver, "*") {
			displayName = fmt.Sprintf("(%s).%s", result.Symbol.Receiver, result.Symbol.Name)
		} else {
			displayName = result.Symbol.Receiver + "." + result.Symbol.Name
		}
	}

	bundle := &SymbolBundle{
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
			UpstreamTruncated:  result.UpstreamTruncated,
			UpstreamIncomplete: result.UpstreamIncomplete,
		},
		Definition: SymbolBundleDefinition{
			File:    result.Symbol.File,
			Line:    result.Symbol.Line,
			EndLine: result.Symbol.EndLine,
			Body:    append([]string(nil), result.Body...),
		},
		Debug: SymbolBundleDebug{
			Source:       "go-inspect",
			FileRootPath: result.Symbol.RootPath,
		},
	}

	if len(result.Body) > 0 {
		bundle.Definition.Signature = result.Body[0]
	}

	addNavigationSection(bundle, "callers", "Callers", result.Callers, result.TotalCallers, result.MoreCallers)
	addNavigationSection(bundle, "references", "References", result.Refs, result.TotalRefs, result.MoreRefs)
	addNavigationTestSection(bundle, result.Tests, result.TotalTests, result.MoreTests)
	if len(result.Implementations) > 0 {
		limit := min(goImplementationLimit, len(result.Implementations))
		items := make([]SymbolBundleItem, 0, limit)
		for _, impl := range result.Implementations[:limit] {
			items = append(items, SymbolBundleItem{
				File:    impl.File,
				Line:    impl.Line,
				Snippet: strings.TrimSpace(impl.Name),
				Name:    impl.Name,
			})
		}
		bundle.Sections = append(bundle.Sections, SymbolBundleSection{
			Kind:  "implementations",
			Title: "Related Implementations",
			Items: items,
			Total: len(result.Implementations),
			More:  len(result.Implementations) > len(items),
		})
	}

	return bundle
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
			File:    ref.File,
			Line:    ref.Line,
			Snippet: snippet,
			Scope:   ref.Scope,
			IsTest:  ref.IsTest,
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
			File:   test.File,
			Line:   test.Line,
			Name:   test.Name,
			IsTest: true,
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

func buildGenericBundleSection(def genericSymbolDef, input symbolBundleSectionInput) *SymbolBundleSection {
	if len(input.Items) == 0 {
		return nil
	}

	total := len(dedupeGenericRefs(input.Items))
	items := prioritizeGenericRefs(def, input.Items, input.Limit, input.IsTest)
	if len(items) == 0 {
		return nil
	}

	sectionItems := make([]SymbolBundleItem, 0, len(items))
	for _, item := range items {
		sectionItems = append(sectionItems, SymbolBundleItem{
			File:    item.File,
			Line:    item.Line,
			Snippet: strings.TrimSpace(item.Snippet),
			IsTest:  input.IsTest || item.IsTest,
		})
	}

	return &SymbolBundleSection{
		Kind:  input.Kind,
		Title: input.Title,
		Items: sectionItems,
		Total: total,
		More:  total > len(sectionItems),
	}
}

func dedupeGenericRefs(refs []genericSymbolRef) []genericSymbolRef {
	seen := make(map[string]bool)
	result := make([]genericSymbolRef, 0, len(refs))
	for _, ref := range refs {
		key := fmt.Sprintf("%s:%d", ref.File, ref.Line)
		if seen[key] {
			continue
		}
		seen[key] = true
		result = append(result, ref)
	}
	return result
}

func prioritizeGenericRefs(def genericSymbolDef, refs []genericSymbolRef, limit int, testOnly bool) []genericSymbolRef {
	if limit <= 0 {
		return nil
	}

	uniq := dedupeGenericRefs(refs)
	sort.SliceStable(uniq, func(i, j int) bool {
		left := genericRefScore(def, uniq[i], testOnly)
		right := genericRefScore(def, uniq[j], testOnly)
		if left != right {
			return left > right
		}
		if uniq[i].File != uniq[j].File {
			return uniq[i].File < uniq[j].File
		}
		return uniq[i].Line < uniq[j].Line
	})

	selected := make([]genericSymbolRef, 0, min(limit, len(uniq)))
	seenFile := make(map[string]bool)
	for _, ref := range uniq {
		if len(selected) >= limit {
			break
		}
		if seenFile[ref.File] {
			continue
		}
		seenFile[ref.File] = true
		selected = append(selected, ref)
	}
	for _, ref := range uniq {
		if len(selected) >= limit {
			break
		}
		if containsGenericRef(selected, ref) {
			continue
		}
		selected = append(selected, ref)
	}
	return selected
}

func containsGenericRef(refs []genericSymbolRef, target genericSymbolRef) bool {
	for _, ref := range refs {
		if ref.File == target.File && ref.Line == target.Line {
			return true
		}
	}
	return false
}

func genericRefScore(def genericSymbolDef, ref genericSymbolRef, testOnly bool) int {
	score := 0
	defDir := filepath.Dir(def.File)
	refDir := filepath.Dir(ref.File)
	if defDir == refDir {
		score += 40
	}
	if ref.File == def.File {
		score += 30
	}
	if strings.Contains(strings.ToLower(filepath.Base(ref.File)), strings.ToLower(def.Name)) {
		score += 20
	}
	if strings.Contains(strings.ToLower(filepath.Base(ref.File)), strings.ToLower(strings.TrimSuffix(filepath.Base(def.File), filepath.Ext(def.File)))) {
		score += 15
	}
	if testOnly {
		if ref.IsTest {
			score += 50
		}
	} else if ref.IsTest {
		score -= 100
	}
	if classifyFilePath(ref.File) == "impl" {
		score += 5
	}
	score -= min(ref.Line, 200)
	return score
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
