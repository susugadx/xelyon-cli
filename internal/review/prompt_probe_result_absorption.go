package review

import (
	"context"
	"fmt"
	"strings"

	reviewmodelinput "github.com/susugadx/xelyon-cli/internal/review/modelinput"
	reviewprobe "github.com/susugadx/xelyon-cli/internal/review/probe"
	reviewpromptreduction "github.com/susugadx/xelyon-cli/internal/review/promptreduction"
	reviewreport "github.com/susugadx/xelyon-cli/internal/review/report"
)

const (
	reviewPromptProbeResultAbsorptionMinSavedTokens = 128
)

type reviewProbeResultAbsorptionCandidate struct {
	summary          reviewmodelinput.ProbeResultAbsorptionSummary
	originalBytes    int
	replacementBytes int
	savedBytes       int
	savedTokens      int
}

type reviewProbeResultAbsorptionCandidates struct {
	probes   map[string]reviewProbeResultAbsorptionCandidate
	commands map[reviewmodelinput.ProbeCommandResultKey]reviewProbeResultAbsorptionCandidate
}

func (c reviewProbeResultAbsorptionCandidates) empty() bool {
	return len(c.probes) == 0 && len(c.commands) == 0
}

type reviewProbeResultPromptContextBuild struct {
	options          reviewmodelinput.ProbeResultPromptContextOptions
	rawOutputContext string
	rawOutputLedger  *reviewpromptreduction.ReviewProbeRawOutputLedger
}

func (r *ReviewRunner) probeResultPromptContextBuildForAbsorbedReport(ctx context.Context, phase ReviewModelPhase, promptKind string, report reviewreport.ReviewReport, probeResults []reviewprobe.ReviewProbeResult, redactor reviewmodelinput.Redactor) reviewProbeResultPromptContextBuild {
	opts := r.probeResultPromptContextOptions()
	if r == nil {
		return reviewProbeResultPromptContextBuild{options: opts}
	}
	mode := reviewpromptreduction.NormalizeReviewPromptReductionMode(r.promptReductionMode)
	if mode == reviewpromptreduction.ReviewPromptReductionModeOff {
		return reviewProbeResultPromptContextBuild{options: opts}
	}
	candidates := buildReviewProbeResultAbsorptionCandidates(report, probeResults)
	if candidates.empty() {
		return reviewProbeResultPromptContextBuild{options: opts}
	}
	if len(candidates.probes) > 0 {
		opts.AbsorptionCandidateProbeIDs = make(map[string]struct{}, len(candidates.probes))
		for probeID := range candidates.probes {
			opts.AbsorptionCandidateProbeIDs[probeID] = struct{}{}
		}
	}
	if len(candidates.commands) > 0 {
		opts.AbsorptionCandidateCommands = make(map[reviewmodelinput.ProbeCommandResultKey]struct{}, len(candidates.commands))
		for key := range candidates.commands {
			opts.AbsorptionCandidateCommands[key] = struct{}{}
		}
	}
	if r.promptReductionStats == nil {
		r.promptReductionStats = reviewpromptreduction.NewStats(mode)
	}
	rawOutput := r.buildReviewProbeRawOutputForCandidates(ctx, phase, promptKind, candidates, probeResults, redactor)
	applied := rawOutput.applyAllowed
	for _, probeID := range sortedReviewProbeAbsorptionProbeIDs(candidates.probes) {
		candidate := candidates.probes[probeID]
		if ref, ok := rawOutput.probeRefs[probeID]; ok {
			candidate.summary.RawArtifactRef = ref.RefID
			candidate.replacementBytes = len(candidate.summary.Summary) + len(strings.Join(candidate.summary.AbsorbedBy, "\n")) + len(candidate.summary.RawArtifactRef)
		}
		r.promptReductionStats.RecordCandidate("probe_result_absorption_candidate", candidate.savedBytes, candidate.savedTokens, applied)
		if !applied {
			r.promptReductionStats.RecordKeepReason(reviewProbeResultAbsorptionKeepReason(rawOutput))
		}
		r.recordPromptReductionItem(reviewpromptreduction.ReviewPromptReductionItem{
			ID:               "probe_result:" + probeID,
			Family:           reviewpromptreduction.ReviewPromptReductionFamilyProbeResult,
			Phase:            reviewPromptReductionPhase(phase),
			Status:           reviewPromptProbeResultAbsorptionStatus(applied),
			AbsorbedBy:       reviewpromptreduction.ReviewPromptAbsorptionRefsFromOwners(candidate.summary.AbsorbedBy),
			EvidenceRefs:     []reviewreport.ReviewEvidenceRef{{Kind: reviewreport.ReviewEvidenceKindProbe, ProbeID: probeID}},
			RawArtifactRef:   candidate.summary.RawArtifactRef,
			Summary:          candidate.summary.Summary,
			OriginalBytes:    candidate.originalBytes,
			ReplacementBytes: candidate.replacementBytes,
		})
	}
	for _, key := range sortedReviewProbeCommandAbsorptionKeys(candidates.commands) {
		candidate := candidates.commands[key]
		if ref, ok := rawOutput.commandRefs[key]; ok {
			candidate.summary.RawArtifactRef = ref.RefID
			candidate.replacementBytes = len(candidate.summary.Summary) + len(strings.Join(candidate.summary.AbsorbedBy, "\n")) + len(candidate.summary.RawArtifactRef)
		}
		r.promptReductionStats.RecordCandidate("probe_command_result_absorption_candidate", candidate.savedBytes, candidate.savedTokens, applied)
		if !applied {
			r.promptReductionStats.RecordKeepReason(reviewProbeResultAbsorptionKeepReason(rawOutput))
		}
		r.recordPromptReductionItem(reviewpromptreduction.ReviewPromptReductionItem{
			ID:               fmt.Sprintf("probe_result:%s:command:%d", key.ProbeID, key.CommandIndex),
			Family:           reviewpromptreduction.ReviewPromptReductionFamilyProbeResult,
			Phase:            reviewPromptReductionPhase(phase),
			Status:           reviewPromptProbeResultAbsorptionStatus(applied),
			AbsorbedBy:       reviewpromptreduction.ReviewPromptAbsorptionRefsFromOwners(candidate.summary.AbsorbedBy),
			EvidenceRefs:     []reviewreport.ReviewEvidenceRef{{Kind: reviewreport.ReviewEvidenceKindProbeCommand, ProbeID: key.ProbeID, CommandIndex: reviewreport.ReviewCommandIndex(key.CommandIndex)}},
			RawArtifactRef:   candidate.summary.RawArtifactRef,
			Summary:          candidate.summary.Summary,
			OriginalBytes:    candidate.originalBytes,
			ReplacementBytes: candidate.replacementBytes,
		})
	}
	rawOutputLedger := reviewpromptreduction.ReviewProbeRawOutputLedgerPtr(rawOutput.ledger)
	if rawOutputLedger != nil {
		r.promptReductionStats.RecordRawOutputLedger(*rawOutputLedger)
		if r.promptReductionState != nil {
			r.promptReductionState.Report = r.promptReductionStats.Report()
		}
	}
	if !applied {
		return reviewProbeResultPromptContextBuild{options: opts}
	}
	if len(candidates.probes) > 0 {
		opts.AbsorbedProbeResults = make(map[string]reviewmodelinput.ProbeResultAbsorptionSummary, len(candidates.probes))
		for probeID, candidate := range candidates.probes {
			if ref, ok := rawOutput.probeRefs[probeID]; ok {
				candidate.summary.RawArtifactRef = ref.RefID
				candidate.summary.Summary = reviewProbeResultAbsorptionAppliedSummary(probeID, false, 0)
			}
			opts.AbsorbedProbeResults[probeID] = candidate.summary
		}
	}
	if len(candidates.commands) > 0 {
		opts.AbsorbedProbeCommands = make(map[reviewmodelinput.ProbeCommandResultKey]reviewmodelinput.ProbeResultAbsorptionSummary, len(candidates.commands))
		for key, candidate := range candidates.commands {
			if ref, ok := rawOutput.commandRefs[key]; ok {
				candidate.summary.RawArtifactRef = ref.RefID
				candidate.summary.Summary = reviewProbeResultAbsorptionAppliedSummary(key.ProbeID, true, key.CommandIndex)
			}
			opts.AbsorbedProbeCommands[key] = candidate.summary
		}
	}
	return reviewProbeResultPromptContextBuild{
		options:          opts,
		rawOutputContext: rawOutput.context,
		rawOutputLedger:  rawOutputLedger,
	}
}

func reviewPromptProbeResultAbsorptionStatus(applied bool) reviewpromptreduction.ReviewPromptReductionItemStatus {
	if applied {
		return reviewpromptreduction.ReviewPromptReductionItemAbsorbed
	}
	return reviewpromptreduction.ReviewPromptReductionItemCandidate
}
