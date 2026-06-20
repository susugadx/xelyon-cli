package review

import (
	"context"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/rawoutputs"
	reviewmodelinput "github.com/susugadx/xelyon-cli/internal/review/modelinput"
	reviewprobe "github.com/susugadx/xelyon-cli/internal/review/probe"
	reviewpromptreduction "github.com/susugadx/xelyon-cli/internal/review/promptreduction"
)

type reviewProbeRawOutputBuild struct {
	probeRefs      map[string]rawoutputs.RawOutputRef
	commandRefs    map[reviewmodelinput.ProbeCommandResultKey]rawoutputs.RawOutputRef
	context        string
	ledger         reviewpromptreduction.ReviewProbeRawOutputLedger
	applyAllowed   bool
	disabledReason string
}

type reviewProbeRawOutputSource struct {
	probeID          string
	commandIndex     *int
	command          reviewprobe.ReviewProbeCommandResult
	body             string
	required         bool
	absorbedBy       []string
	originalBytes    int
	replacementBytes int
}

func (r *ReviewRunner) buildReviewProbeRawOutputForCandidates(ctx context.Context, phase ReviewModelPhase, promptKind string, candidates reviewProbeResultAbsorptionCandidates, probeResults []reviewprobe.ReviewProbeResult, redactor reviewmodelinput.Redactor) reviewProbeRawOutputBuild {
	build := reviewProbeRawOutputBuild{
		probeRefs:   map[string]rawoutputs.RawOutputRef{},
		commandRefs: map[reviewmodelinput.ProbeCommandResultKey]rawoutputs.RawOutputRef{},
		ledger: reviewpromptreduction.ReviewProbeRawOutputLedger{
			ReviewRunID:           r.reviewRunID,
			Phase:                 reviewPromptReductionPhase(phase),
			PromptKind:            promptKind,
			BudgetTokens:          r.reviewProbeRawOutputBudget(),
			MetadataReserveTokens: reviewpromptreduction.ReviewProbeRawOutputMetadataReserve(r.reviewProbeRawOutputBudget()),
			CanAcceptSaturated:    true,
		},
	}
	build.ledger.BodyBudgetTokens = max(0, build.ledger.BudgetTokens-build.ledger.MetadataReserveTokens)
	if candidates.empty() {
		return build
	}
	if reviewpromptreduction.NormalizeReviewRawOutputArtifactsMode(r.rawOutputArtifactsMode) == reviewpromptreduction.ReviewRawOutputArtifactsModeOff {
		build.disabledReason = reviewpromptreduction.ReviewProbeRawOutputReasonArtifactMissing
		return build
	}
	if strings.TrimSpace(r.rawOutputSessionID) == "" || r.rawOutputArtifactStore == nil {
		build.disabledReason = reviewpromptreduction.ReviewProbeRawOutputReasonArtifactMissing
		return build
	}

	sources := reviewProbeRawOutputSources(candidates, probeResults)
	if len(sources) == 0 {
		build.disabledReason = reviewpromptreduction.ReviewProbeRawOutputReasonUnreflectedEvidenceKeep
		return build
	}
	for _, source := range sources {
		ref, reason, ok := r.createReviewProbeRawOutputArtifact(ctx, phase, source)
		ledgerRef := reviewpromptreduction.NewReviewProbeRawOutputLedgerRef(reviewProbeRawOutputContextSource(source), ref)
		if !ok {
			ledgerRef.Status = "missing"
			ledgerRef.Reason = reason
			build.ledger.MissingRefs = append(build.ledger.MissingRefs, ledgerRef)
			build.ledger.FailClosedReason = reason
			build.ledger.CanAcceptSaturated = false
			build.disabledReason = reason
			continue
		}
		ledgerRef.Status = "required"
		build.ledger.RequiredRefs = append(build.ledger.RequiredRefs, ledgerRef)
		if source.commandIndex == nil {
			build.probeRefs[source.probeID] = ref
		} else {
			build.commandRefs[reviewmodelinput.ProbeCommandResultKey{ProbeID: source.probeID, CommandIndex: *source.commandIndex}] = ref
		}
	}
	if len(build.ledger.MissingRefs) > 0 {
		return build
	}

	contextText, ledger := r.renderReviewProbeRawOutputContext(ctx, build.ledger, sources, build.probeRefs, build.commandRefs, redactor)
	build.context = contextText
	build.ledger = ledger
	if !ledger.CanAcceptSaturated {
		build.disabledReason = ledger.FailClosedReason
		return build
	}
	build.applyAllowed = reviewpromptreduction.NormalizeReviewPromptReductionMode(r.promptReductionMode) == reviewpromptreduction.ReviewPromptReductionModeApply &&
		reviewpromptreduction.NormalizeReviewRawOutputArtifactsMode(r.rawOutputArtifactsMode) == reviewpromptreduction.ReviewRawOutputArtifactsModeApply &&
		strings.TrimSpace(build.context) != ""
	if !build.applyAllowed && reviewpromptreduction.NormalizeReviewRawOutputArtifactsMode(r.rawOutputArtifactsMode) == reviewpromptreduction.ReviewRawOutputArtifactsModeDryRun {
		build.disabledReason = reviewpromptreduction.ReviewProbeRawOutputReasonArtifactsDryRun
	}
	return build
}

func reviewProbeRawOutputSources(candidates reviewProbeResultAbsorptionCandidates, probeResults []reviewprobe.ReviewProbeResult) []reviewProbeRawOutputSource {
	resultsByID := make(map[string]reviewprobe.ReviewProbeResult, len(probeResults))
	for _, result := range probeResults {
		if strings.TrimSpace(result.ID) != "" {
			resultsByID[result.ID] = result
		}
	}
	sources := make([]reviewProbeRawOutputSource, 0, len(candidates.probes)+len(candidates.commands))
	for _, probeID := range sortedReviewProbeAbsorptionProbeIDs(candidates.probes) {
		result, ok := resultsByID[probeID]
		if !ok {
			continue
		}
		candidate := candidates.probes[probeID]
		sources = append(sources, reviewProbeRawOutputSource{
			probeID:          probeID,
			body:             reviewProbeRawOutputBodyForProbe(result),
			required:         true,
			absorbedBy:       candidate.summary.AbsorbedBy,
			originalBytes:    candidate.originalBytes,
			replacementBytes: candidate.replacementBytes,
		})
	}
	for _, key := range sortedReviewProbeCommandAbsorptionKeys(candidates.commands) {
		result, ok := resultsByID[key.ProbeID]
		if !ok || key.CommandIndex < 0 || key.CommandIndex >= len(result.CommandResults) {
			continue
		}
		candidate := candidates.commands[key]
		commandIndex := key.CommandIndex
		command := result.CommandResults[commandIndex]
		sources = append(sources, reviewProbeRawOutputSource{
			probeID:          key.ProbeID,
			commandIndex:     &commandIndex,
			command:          command,
			body:             reviewProbeRawOutputBodyForCommand(command),
			required:         true,
			absorbedBy:       candidate.summary.AbsorbedBy,
			originalBytes:    candidate.originalBytes,
			replacementBytes: candidate.replacementBytes,
		})
	}
	return sources
}
