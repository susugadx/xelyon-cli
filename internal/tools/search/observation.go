package search

import (
	"fmt"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/navigation"
	"github.com/susugadx/xelyon-cli/internal/tools"
)

func observationForSearchResults(results []SearchResult, opts SearchOptions) *tools.RuntimeObservation {
	if len(results) == 0 {
		return nil
	}
	observation := &tools.RuntimeObservation{}
	for _, result := range results {
		path := result.FilePath
		resolved := absoluteAffectedFilePath(path, opts, affectedFileSourceText)
		observation.TouchedFiles = append(observation.TouchedFiles, tools.ObservationPath{
			Path:         path,
			ResolvedPath: resolved,
		})
		for _, match := range sortMatchesForSearchResultBody(result.Matches) {
			if match.LineNum <= 0 {
				continue
			}
			observation.Evidence = append(observation.Evidence, tools.ObservationEvidence{
				Path:         path,
				ResolvedPath: resolved,
				StartLine:    match.LineNum,
				EndLine:      match.LineNum,
				Excerpt:      match.Line,
			})
		}
	}
	return nonEmptyObservation(observation)
}

func observationForSymbolBundle(bundle *SymbolBundle, opts SearchOptions) *tools.RuntimeObservation {
	if bundle == nil {
		return nil
	}
	observation := &tools.RuntimeObservation{}
	appendBundleDefinitionObservation(observation, bundle, opts)
	for _, section := range bundle.Sections {
		for _, item := range section.Items {
			appendBundleItemEvidenceObservation(observation, bundle, opts, item)
		}
	}
	if bundle.Impact != nil {
		for _, item := range bundle.Impact.RecommendedReads {
			appendBundleRecommendedReadObservation(observation, bundle, opts, item)
		}
	}
	for _, file := range bundle.Debug.DependencyFiles {
		path := observationPathForBundleFile(file, bundle, opts)
		observation.TouchedFiles = append(observation.TouchedFiles, path)
	}
	return nonEmptyObservation(observation)
}

func appendBundleDefinitionObservation(observation *tools.RuntimeObservation, bundle *SymbolBundle, opts SearchOptions) {
	path := strings.TrimSpace(bundle.Definition.File)
	if path == "" {
		path = bundle.Identity.File
	}
	line := bundle.Definition.Line
	if line == 0 {
		line = bundle.Identity.Line
	}
	endLine := bundle.Definition.EndLine
	if endLine == 0 {
		endLine = bundle.Identity.EndLine
	}
	location := observationPathForBundleFile(path, bundle, opts)
	observation.TouchedFiles = append(observation.TouchedFiles, location)
	observation.Evidence = append(observation.Evidence, tools.ObservationEvidence{
		Path:         location.Path,
		ResolvedPath: location.ResolvedPath,
		StartLine:    line,
		EndLine:      endLine,
		Excerpt:      symbolBundleDefinitionExcerpt(bundle),
	})
}

func appendBundleItemEvidenceObservation(observation *tools.RuntimeObservation, bundle *SymbolBundle, opts SearchOptions, item SymbolBundleItem) {
	location := observationPathForBundleItem(item, bundle, opts)
	observation.TouchedFiles = append(observation.TouchedFiles, location)
	observation.Evidence = append(observation.Evidence, tools.ObservationEvidence{
		Path:         location.Path,
		ResolvedPath: location.ResolvedPath,
		StartLine:    item.Line,
		EndLine:      item.EndLine,
		Excerpt:      symbolBundleItemExcerpt(item),
	})
}

func appendBundleRecommendedReadObservation(observation *tools.RuntimeObservation, bundle *SymbolBundle, opts SearchOptions, item SymbolBundleItem) {
	location := observationPathForBundleItem(item, bundle, opts)
	observation.RecommendedReads = append(observation.RecommendedReads, tools.ObservationRecommendedRead{
		Path:         location.Path,
		ResolvedPath: location.ResolvedPath,
		Reason:       recommendedReadReason(item),
	})
}

func observationPathForBundleItem(item SymbolBundleItem, bundle *SymbolBundle, opts SearchOptions) tools.ObservationPath {
	if resolved := cleanResolvedLocatorPath(item.ResolvedPath); resolved != "" {
		return tools.ObservationPath{Path: item.File, ResolvedPath: resolved}
	}
	return observationPathForBundleFile(item.File, bundle, opts)
}

func observationPathForBundleFile(file string, bundle *SymbolBundle, opts SearchOptions) tools.ObservationPath {
	rootPath := ""
	if bundle != nil {
		rootPath = bundle.Debug.FileRootPath
	}
	return tools.ObservationPath{
		Path:         file,
		ResolvedPath: absoluteAffectedFilePathForBundle(file, opts, rootPath),
	}
}

func symbolBundleDefinitionExcerpt(bundle *SymbolBundle) string {
	if bundle == nil {
		return ""
	}
	if signature := strings.TrimSpace(bundle.Definition.Signature); signature != "" {
		return signature
	}
	for _, line := range bundle.Definition.Body {
		if trimmed := strings.TrimSpace(line); trimmed != "" {
			return trimmed
		}
	}
	if bundle.Identity.DisplayName != "" {
		return fmt.Sprintf("%s %s", bundle.Identity.Kind, bundle.Identity.DisplayName)
	}
	return bundle.Identity.Canonical
}

func symbolBundleItemExcerpt(item SymbolBundleItem) string {
	if snippet := strings.TrimSpace(item.Snippet); snippet != "" {
		return snippet
	}
	if item.IsTest && item.Name != "" {
		return "func " + item.Name
	}
	if item.Name != "" {
		return item.Name
	}
	return item.Scope
}

func recommendedReadReason(item SymbolBundleItem) string {
	if reason := symbolBundleItemExcerpt(item); reason != "" {
		return reason
	}
	return "recommended read"
}

func observationForGenericDefinitions(defs []genericSymbolDef, opts SearchOptions) *tools.RuntimeObservation {
	if len(defs) == 0 {
		return nil
	}
	observation := &tools.RuntimeObservation{}
	basePath := invocationCWDOrGetwd(opts)
	for _, def := range defs {
		resolved := absoluteAffectedFilePathWithBase(def.File, basePath)
		observation.TouchedFiles = append(observation.TouchedFiles, tools.ObservationPath{
			Path:         def.File,
			ResolvedPath: resolved,
		})
		observation.Evidence = append(observation.Evidence, tools.ObservationEvidence{
			Path:         def.File,
			ResolvedPath: resolved,
			StartLine:    def.Line,
			EndLine:      def.Line,
			Excerpt:      strings.TrimSpace(def.Signature),
		})
	}
	return nonEmptyObservation(observation)
}

func observationForNavigationCandidates(candidates []navigation.SymbolCandidate, opts SearchOptions) *tools.RuntimeObservation {
	if len(candidates) == 0 {
		return nil
	}
	observation := &tools.RuntimeObservation{}
	for _, candidate := range candidates {
		resolved := absoluteAffectedFilePathForSymbol(candidate.File, opts, candidate.RootPath)
		observation.TouchedFiles = append(observation.TouchedFiles, tools.ObservationPath{
			Path:         candidate.File,
			ResolvedPath: resolved,
		})
		observation.Evidence = append(observation.Evidence, tools.ObservationEvidence{
			Path:         candidate.File,
			ResolvedPath: resolved,
			StartLine:    candidate.Line,
			EndLine:      candidate.EndLine,
			Excerpt:      navigationCandidateExcerpt(candidate),
		})
	}
	return nonEmptyObservation(observation)
}

func navigationCandidateExcerpt(candidate navigation.SymbolCandidate) string {
	name := strings.TrimSpace(candidate.Name)
	if name == "" {
		return strings.TrimSpace(candidate.File)
	}
	if receiver := strings.TrimSpace(candidate.Receiver); receiver != "" {
		name = receiver + "." + name
	}
	if kind := strings.TrimSpace(candidate.Kind); kind != "" {
		return kind + " " + name
	}
	return name
}

func singlePatternObservationMap(pattern string, observation *tools.RuntimeObservation) map[string]*tools.RuntimeObservation {
	cloned := tools.CloneRuntimeObservation(observation)
	if cloned == nil {
		return nil
	}
	return map[string]*tools.RuntimeObservation{pattern: cloned}
}

func loadCachedPatternObservations(contexts []singlePatternExecutionContext) map[string]*tools.RuntimeObservation {
	groups := make(map[string]*tools.RuntimeObservation)
	for _, ctx := range contexts {
		bundle := loadSinglePatternBundle(ctx.Pattern, ctx.CacheKey)
		if observation := loadCachedSinglePatternObservation(ctx, bundle); observation != nil {
			groups[ctx.Pattern] = observation
		}
	}
	if len(groups) == 0 {
		return nil
	}
	return groups
}

func nonEmptyObservation(item *tools.RuntimeObservation) *tools.RuntimeObservation {
	if item == nil || item.Empty() {
		return nil
	}
	return item
}
