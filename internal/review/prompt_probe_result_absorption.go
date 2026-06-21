package review

import (
	"context"

	reviewmodelinput "github.com/susugadx/xelyon-cli/internal/review/modelinput"
	reviewprobe "github.com/susugadx/xelyon-cli/internal/review/probe"
	reviewpromptreduction "github.com/susugadx/xelyon-cli/internal/review/promptreduction"
	reviewreport "github.com/susugadx/xelyon-cli/internal/review/report"
)

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
	plan := reviewpromptreduction.BuildProbeResultAbsorptionPlan(report, probeResults)
	if plan.Empty() {
		return reviewProbeResultPromptContextBuild{options: opts}
	}
	opts = plan.CandidatePromptContextOptions(opts)
	if r.promptReductionStats == nil {
		r.promptReductionStats = reviewpromptreduction.NewStats(mode)
	}
	rawOutput := r.buildReviewProbeRawOutputForCandidates(ctx, phase, promptKind, plan, probeResults, redactor)
	applied := rawOutput.applyAllowed
	artifactRefs := reviewProbeResultAbsorptionArtifactRefs(rawOutput)
	for _, record := range plan.ReductionRecords(reviewPromptReductionPhase(phase), applied, artifactRefs) {
		r.promptReductionStats.RecordCandidate(record.Classifier, record.SavedBytes, record.SavedTokens, applied)
		if !applied {
			r.promptReductionStats.RecordKeepReason(reviewProbeResultAbsorptionKeepReason(rawOutput))
		}
		r.recordPromptReductionItem(record.Item)
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
	opts = plan.AbsorbedPromptContextOptions(opts, artifactRefs)
	return reviewProbeResultPromptContextBuild{
		options:          opts,
		rawOutputContext: rawOutput.context,
		rawOutputLedger:  rawOutputLedger,
	}
}

func reviewProbeResultAbsorptionArtifactRefs(rawOutput reviewProbeRawOutputBuild) reviewpromptreduction.ProbeResultAbsorptionArtifactRefs {
	refs := reviewpromptreduction.ProbeResultAbsorptionArtifactRefs{}
	if len(rawOutput.probeRefs) > 0 {
		refs.ProbeResults = make(map[string]string, len(rawOutput.probeRefs))
		for probeID, ref := range rawOutput.probeRefs {
			refs.ProbeResults[probeID] = ref.RefID
		}
	}
	if len(rawOutput.commandRefs) > 0 {
		refs.ProbeCommands = make(map[reviewmodelinput.ProbeCommandResultKey]string, len(rawOutput.commandRefs))
		for key, ref := range rawOutput.commandRefs {
			refs.ProbeCommands[key] = ref.RefID
		}
	}
	return refs
}
