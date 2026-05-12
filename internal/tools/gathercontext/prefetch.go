package gathercontext

import (
	"strconv"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/locator"
	"github.com/susugadx/xelyon-cli/internal/tools"
	filetool "github.com/susugadx/xelyon-cli/internal/tools/file"
	"github.com/susugadx/xelyon-cli/internal/tools/search"
)

const maxPrefetchedReads = 3

type prefetchResult struct {
	output      string
	observation *tools.RuntimeObservation
}

func prefetchRecommendedEvidence(execCtx tools.ExecutionContext, artifact search.SearchExecutionArtifact) prefetchResult {
	if !artifact.Metadata.StructuredImpact || artifact.Metadata.Ambiguous || artifact.Metadata.Bundle == nil || artifact.Metadata.Bundle.Impact == nil {
		return prefetchResult{}
	}
	items := boundedRecommendedReads(artifact.Metadata.Bundle.Impact.RecommendedReads, maxPrefetchedReads)
	if len(items) == 0 {
		return prefetchResult{}
	}

	var sections []string
	var observations []*tools.RuntimeObservation
	reg := execCtx.EffectiveLocatorRegistry()
	for _, item := range items {
		target := registerPrefetchLocator(reg, item)
		if target == "" {
			continue
		}
		for _, section := range filetool.ExecuteReadTargetsWithDetailSections(execCtx, target, "compact") {
			if section.Failed || strings.TrimSpace(section.Output) == "" {
				continue
			}
			sections = append(sections, section.Output)
			observations = append(observations, section.Observation)
		}
	}
	return prefetchResult{
		output:      strings.Join(sections, "\n\n"),
		observation: tools.MergeRuntimeObservations(observations...),
	}
}

func boundedRecommendedReads(items []search.SymbolBundleItem, limit int) []search.SymbolBundleItem {
	if limit <= 0 || len(items) == 0 {
		return nil
	}
	result := make([]search.SymbolBundleItem, 0, min(limit, len(items)))
	seen := make(map[string]struct{}, limit)
	for _, item := range items {
		if item.File == "" || item.Line <= 0 {
			continue
		}
		key := item.File + "\x00" + item.ResolvedPath + "\x00" + item.Kind + "\x00" + item.Name + "\x00" + strconv.Itoa(item.Line)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, item)
		if len(result) >= limit {
			break
		}
	}
	return result
}

func registerPrefetchLocator(reg *locator.Registry, item search.SymbolBundleItem) string {
	if reg == nil {
		return ""
	}
	name := strings.TrimSpace(item.Name)
	if name == "" {
		name = strings.TrimSpace(item.Kind)
	}
	return reg.Register(locator.Location{
		FilePath:     item.File,
		ResolvedPath: item.ResolvedPath,
		Line:         item.Line,
		EndLine:      item.EndLine,
		Name:         name,
	})
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
