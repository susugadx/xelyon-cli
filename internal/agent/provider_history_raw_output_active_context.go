package agent

import (
	"context"
	"io"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/rawoutputs"
	"github.com/susugadx/xelyon-cli/internal/token"
)

const (
	providerHistoryRawOutputActiveContextName                       = "provider_history_raw_output_context"
	providerHistoryRawOutputContextHeader                           = "Provider History Raw Output Context"
	providerHistoryRawOutputRequiredRefsMissingReason               = "raw_output_active_context_required_refs_missing"
	providerHistoryRawOutputActiveContextCoverageInsufficientReason = "raw_output_active_context_coverage_insufficient"
)

type rawOutputArtifactResolver interface {
	Resolve(ctx context.Context, ref rawoutputs.RawOutputRef) (rawoutputs.ResolvedArtifact, error)
}

type providerHistoryRawOutputActiveContextBuild struct {
	Blocks                     []api.ActiveContextBlock
	RequiredRefCount           int
	InjectedRefCount           int
	MissingRefIDs              []string
	CoverageInsufficientRefIDs []string
}

func (b providerHistoryRawOutputActiveContextBuild) missingRequiredRefs() bool {
	return b.RequiredRefCount > b.InjectedRefCount || len(b.MissingRefIDs) > 0 || len(b.CoverageInsufficientRefIDs) > 0
}

func (b providerHistoryRawOutputActiveContextBuild) failClosedReason() string {
	if len(b.CoverageInsufficientRefIDs) > 0 {
		return providerHistoryRawOutputActiveContextCoverageInsufficientReason
	}
	return providerHistoryRawOutputRequiredRefsMissingReason
}

func (a *Agent) buildProviderHistoryRawOutputActiveContext(ctx context.Context, report ProviderHistoryProjectionReport, raw []api.Message) providerHistoryRawOutputActiveContextBuild {
	refs := providerHistoryAppliedRawOutputRefs(report)
	result := providerHistoryRawOutputActiveContextBuild{RequiredRefCount: len(refs)}
	if len(refs) == 0 {
		return result
	}
	if !a.shouldBuildProviderHistoryRawOutputActiveContext() {
		result.MissingRefIDs = providerHistoryRawOutputRefIDs(refs)
		return result
	}
	resolver := a.providerHistoryRawOutputArtifactResolver()
	if resolver == nil {
		result.MissingRefIDs = providerHistoryRawOutputRefIDs(refs)
		return result
	}

	budget := providerHistoryRawOutputActiveContextBudget(a.Runtime)
	var b strings.Builder
	b.WriteString(providerHistoryRawOutputContextHeader)
	usedTokens := token.EstimateTokenCount(providerHistoryRawOutputContextHeader)
	injected := 0
	hints := providerHistoryRawOutputRehydrateHintsFromRaw(raw)
	for _, ref := range refs {
		if strings.EqualFold(ref.SemanticRole, "sensitive") {
			result.MissingRefIDs = append(result.MissingRefIDs, ref.RefID)
			continue
		}
		resolved, err := resolver.Resolve(ctx, ref)
		if err != nil {
			result.MissingRefIDs = append(result.MissingRefIDs, ref.RefID)
			continue
		}
		body, readErr := io.ReadAll(resolved.Body)
		_ = resolved.Body.Close()
		if readErr != nil {
			result.MissingRefIDs = append(result.MissingRefIDs, ref.RefID)
			continue
		}
		entry, reason := renderProviderHistoryRawOutputContextEntry(ref, string(body), budget-usedTokens, hints)
		entryTokens := token.EstimateTokenCount(entry)
		if entry == "" || entryTokens <= 0 || usedTokens+entryTokens > budget {
			if reason == providerHistoryRawOutputActiveContextCoverageInsufficientReason {
				result.CoverageInsufficientRefIDs = append(result.CoverageInsufficientRefIDs, ref.RefID)
			} else {
				result.MissingRefIDs = append(result.MissingRefIDs, ref.RefID)
			}
			continue
		}
		b.WriteString("\n")
		b.WriteString(entry)
		usedTokens += entryTokens
		injected++
	}
	result.InjectedRefCount = injected
	if injected == 0 {
		return result
	}
	result.Blocks = []api.ActiveContextBlock{{
		Name:    providerHistoryRawOutputActiveContextName,
		Content: b.String(),
	}}
	return result
}

func (a *Agent) shouldBuildProviderHistoryRawOutputActiveContext() bool {
	if a == nil || a.Runtime == nil {
		return false
	}
	return a.Runtime.Options.EnableProviderHistoryRehydrateContext && a.providerCanConsumeActiveContext()
}

func (a *Agent) providerHistoryRawOutputArtifactResolver() rawOutputArtifactResolver {
	if a == nil || a.Runtime == nil {
		return nil
	}
	if resolver, ok := a.Runtime.RawOutputArtifactStore.(rawOutputArtifactResolver); ok {
		return resolver
	}
	if resolver, ok := a.providerHistoryRawOutputArtifactStore().(rawOutputArtifactResolver); ok {
		return resolver
	}
	return nil
}
