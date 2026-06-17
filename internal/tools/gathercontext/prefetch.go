package gathercontext

import (
	"strings"

	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/tools/search"
)

const maxPrefetchedReads = 3

type prefetchResult struct {
	output        string
	observation   *tools.RuntimeObservation
	discoveryNote string
}

func prefetchRecommendedEvidence(execCtx tools.ExecutionContext, artifact search.SearchExecutionArtifact) prefetchResult {
	policy := prefetchPolicyForArtifact(artifact.Metadata)
	if !policy.shouldPrefetch {
		if policy.reason != "" {
			return prefetchResult{discoveryNote: prefetchSkippedNote(policy.reason)}
		}
		return prefetchResult{}
	}
	items := boundedRecommendedReads(artifact.Metadata.Bundle.Impact.RecommendedReads, policy.limit)
	if len(items) == 0 {
		return prefetchResult{}
	}

	sections, observations := executePrefetchReads(execCtx, items)
	if len(sections) == 0 {
		return prefetchResult{}
	}
	if policy.limited {
		sections = append([]string{prefetchLimitedNote(policy.reason)}, sections...)
	}
	return prefetchResult{
		output:      strings.Join(sections, "\n\n"),
		observation: mergePrefetchObservations(observations),
	}
}
