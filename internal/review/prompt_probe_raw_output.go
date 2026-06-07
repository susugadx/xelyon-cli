package review

import (
	"context"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/rawoutputs"
	reviewmodelinput "github.com/susugadx/xelyon-cli/internal/review/modelinput"
)

const (
	reviewProbeRawOutputContextHeader                 = "Review Probe Raw Output Context"
	reviewProbeRawOutputDefaultBudgetTokens           = 4096
	reviewProbeRawOutputDefaultBudgetMaxTokens        = 8192
	reviewProbeRawOutputMetadataReserveTokens         = 512
	reviewProbeRawOutputMetadataReservePercent        = 15
	reviewProbeRawOutputRequiredRefBodyMinTokens      = 512
	reviewProbeRawOutputOptionalRefBodyMinTokens      = 160
	reviewProbeRawOutputPerCommandBodyMaxPercent      = 50
	reviewProbeRawOutputSingleExplicitRefMaxPercent   = 80
	reviewProbeRawOutputCommandPreviewRunes           = 160
	reviewProbeRawOutputReasonArtifactMissing         = "review_probe_raw_output_artifact_missing"
	reviewProbeRawOutputReasonRehydrateUnavailable    = "review_probe_raw_output_rehydrate_not_available"
	reviewProbeRawOutputReasonRequiredRefMissing      = "review_probe_required_ref_missing"
	reviewProbeRawOutputReasonRequiredRefMetadataOnly = "review_probe_required_ref_metadata_only"
	reviewProbeRawOutputReasonRequiredRefBodyTooSmall = "review_probe_required_ref_body_budget_too_small"
	reviewProbeRawOutputReasonRequiredRefHashInvalid  = "review_probe_required_ref_hash_invalid"
	reviewProbeRawOutputReasonRequiredRefQuarantined  = "review_probe_required_ref_quarantined"
	reviewProbeRawOutputReasonBudgetRequiresRevision  = "review_probe_budget_requires_blocked_or_needs_revision"
	reviewProbeRawOutputReasonSaturatedRejected       = "review_probe_saturated_rejected_by_rehydrate_ledger"
	reviewProbeRawOutputReasonArtifactsDryRun         = "review_probe_raw_output_artifacts_dry_run"
	reviewProbeRawOutputReasonSensitiveOrPrivateKeep  = "review_probe_sensitive_or_private_keep"
	reviewProbeRawOutputReasonUnreflectedEvidenceKeep = "review_probe_unreflected_evidence_keep"
)

// ReviewRawOutputArtifactStore は review prompt reduction が使う rawoutputs store 境界。
type ReviewRawOutputArtifactStore interface {
	Create(ctx context.Context, req rawoutputs.CreateRequest) (rawoutputs.CreateResult, error)
	Verify(ctx context.Context, ref rawoutputs.RawOutputRef) (rawoutputs.VerifyResult, error)
	Resolve(ctx context.Context, ref rawoutputs.RawOutputRef) (rawoutputs.ResolvedArtifact, error)
}

type reviewProbeRawOutputBuild struct {
	probeRefs      map[string]rawoutputs.RawOutputRef
	commandRefs    map[reviewmodelinput.ProbeCommandResultKey]rawoutputs.RawOutputRef
	context        string
	ledger         ReviewProbeRawOutputLedger
	applyAllowed   bool
	disabledReason string
}

type reviewProbeRawOutputSource struct {
	probeID          string
	commandIndex     *int
	command          ReviewProbeCommandResult
	body             string
	required         bool
	absorbedBy       []string
	originalBytes    int
	replacementBytes int
}

func (r *ReviewRunner) buildReviewProbeRawOutputForCandidates(ctx context.Context, phase ReviewModelPhase, promptKind string, candidates reviewProbeResultAbsorptionCandidates, probeResults []ReviewProbeResult, redactor reviewmodelinput.Redactor) reviewProbeRawOutputBuild {
	build := reviewProbeRawOutputBuild{
		probeRefs:   map[string]rawoutputs.RawOutputRef{},
		commandRefs: map[reviewmodelinput.ProbeCommandResultKey]rawoutputs.RawOutputRef{},
		ledger: ReviewProbeRawOutputLedger{
			ReviewRunID:           r.reviewRunID,
			Phase:                 phase,
			PromptKind:            promptKind,
			BudgetTokens:          r.reviewProbeRawOutputBudget(),
			MetadataReserveTokens: reviewProbeRawOutputMetadataReserve(r.reviewProbeRawOutputBudget()),
			CanAcceptSaturated:    true,
		},
	}
	build.ledger.BodyBudgetTokens = max(0, build.ledger.BudgetTokens-build.ledger.MetadataReserveTokens)
	if candidates.empty() {
		return build
	}
	if normalizeReviewRawOutputArtifactsMode(r.rawOutputArtifactsMode) == ReviewRawOutputArtifactsModeOff {
		build.disabledReason = reviewProbeRawOutputReasonArtifactMissing
		return build
	}
	if strings.TrimSpace(r.rawOutputSessionID) == "" || r.rawOutputArtifactStore == nil {
		build.disabledReason = reviewProbeRawOutputReasonArtifactMissing
		return build
	}

	sources := reviewProbeRawOutputSources(candidates, probeResults)
	if len(sources) == 0 {
		build.disabledReason = reviewProbeRawOutputReasonUnreflectedEvidenceKeep
		return build
	}
	for _, source := range sources {
		ref, reason, ok := r.createReviewProbeRawOutputArtifact(ctx, phase, source)
		ledgerRef := reviewProbeRawOutputLedgerRefFromSource(source, ref)
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
	build.applyAllowed = normalizeReviewPromptReductionMode(r.promptReductionMode) == ReviewPromptReductionModeApply &&
		normalizeReviewRawOutputArtifactsMode(r.rawOutputArtifactsMode) == ReviewRawOutputArtifactsModeApply &&
		strings.TrimSpace(build.context) != ""
	if !build.applyAllowed && normalizeReviewRawOutputArtifactsMode(r.rawOutputArtifactsMode) == ReviewRawOutputArtifactsModeDryRun {
		build.disabledReason = reviewProbeRawOutputReasonArtifactsDryRun
	}
	return build
}

func reviewProbeRawOutputSources(candidates reviewProbeResultAbsorptionCandidates, probeResults []ReviewProbeResult) []reviewProbeRawOutputSource {
	resultsByID := make(map[string]ReviewProbeResult, len(probeResults))
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
