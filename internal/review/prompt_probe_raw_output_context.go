package review

import (
	"context"
	"io"

	"github.com/susugadx/xelyon-cli/internal/rawoutputs"
	reviewmodelinput "github.com/susugadx/xelyon-cli/internal/review/modelinput"
	reviewpromptreduction "github.com/susugadx/xelyon-cli/internal/review/promptreduction"
)

func (r *ReviewRunner) renderReviewProbeRawOutputContext(ctx context.Context, ledger reviewpromptreduction.ReviewProbeRawOutputLedger, sources []reviewProbeRawOutputSource, probeRefs map[string]rawoutputs.RawOutputRef, commandRefs map[reviewmodelinput.ProbeCommandResultKey]rawoutputs.RawOutputRef, redactor reviewmodelinput.Redactor) (string, reviewpromptreduction.ReviewProbeRawOutputLedger) {
	if len(sources) == 0 {
		return "", ledger
	}
	entries := make([]reviewpromptreduction.ReviewProbeRawOutputContextEntry, 0, len(sources))
	for _, source := range sources {
		contextSource := reviewProbeRawOutputContextSource(source)
		ref, ok := reviewProbeRawOutputRefForSource(source, probeRefs, commandRefs)
		if !ok {
			ledger.FailClosedReason = reviewpromptreduction.ReviewProbeRawOutputReasonRequiredRefMissing
			ledger.CanAcceptSaturated = false
			ledger.MissingRefs = append(ledger.MissingRefs, reviewpromptreduction.NewReviewProbeRawOutputLedgerRef(contextSource, rawoutputs.RawOutputRef{
				Surface: string(rawoutputs.SurfaceReviewProbeResult),
			}))
			continue
		}
		resolved, err := r.rawOutputArtifactStore.Resolve(ctx, ref)
		if err != nil {
			reason := reviewProbeRawOutputResolveReason(err)
			ledger.FailClosedReason = reason
			ledger.CanAcceptSaturated = false
			ledger.MissingRefs = append(ledger.MissingRefs, reviewpromptreduction.ReviewProbeRawOutputLedgerRefWithStatus(reviewpromptreduction.NewReviewProbeRawOutputLedgerRef(contextSource, ref), "missing", reason))
			continue
		}
		body, readErr := io.ReadAll(resolved.Body)
		_ = resolved.Body.Close()
		if readErr != nil {
			ledger.FailClosedReason = reviewpromptreduction.ReviewProbeRawOutputReasonRequiredRefMissing
			ledger.CanAcceptSaturated = false
			ledger.MissingRefs = append(ledger.MissingRefs, reviewpromptreduction.ReviewProbeRawOutputLedgerRefWithStatus(reviewpromptreduction.NewReviewProbeRawOutputLedgerRef(contextSource, ref), "missing", reviewpromptreduction.ReviewProbeRawOutputReasonRequiredRefMissing))
			continue
		}
		entries = append(entries, reviewpromptreduction.ReviewProbeRawOutputContextEntry{
			Ref:    ref,
			Source: contextSource,
			Body:   string(body),
		})
	}
	return reviewpromptreduction.RenderReviewProbeRawOutputContext(reviewpromptreduction.ReviewProbeRawOutputContextInput{
		Ledger:   ledger,
		Entries:  entries,
		Redactor: redactor,
	})
}

func reviewProbeRawOutputRefForSource(source reviewProbeRawOutputSource, probeRefs map[string]rawoutputs.RawOutputRef, commandRefs map[reviewmodelinput.ProbeCommandResultKey]rawoutputs.RawOutputRef) (rawoutputs.RawOutputRef, bool) {
	if source.commandIndex == nil {
		ref, ok := probeRefs[source.probeID]
		return ref, ok
	}
	ref, ok := commandRefs[reviewmodelinput.ProbeCommandResultKey{ProbeID: source.probeID, CommandIndex: *source.commandIndex}]
	return ref, ok
}

func reviewProbeRawOutputContextSource(source reviewProbeRawOutputSource) reviewpromptreduction.ReviewProbeRawOutputContextSource {
	var commandIndex *int
	if source.commandIndex != nil {
		cloned := *source.commandIndex
		commandIndex = &cloned
	}
	return reviewpromptreduction.ReviewProbeRawOutputContextSource{
		ProbeID:        source.probeID,
		CommandIndex:   commandIndex,
		CommandPreview: reviewProbeRawOutputCommandPreview(source),
		Status:         string(source.command.Status),
		ExitCode:       source.command.ExitCode,
		AbsorbedBy:     append([]string(nil), source.absorbedBy...),
	}
}

func (r *ReviewRunner) reviewProbeRawOutputBudget() int {
	return reviewpromptreduction.NormalizeReviewProbeRawOutputBudget(r.rawOutputRehydrateBudgetTokens, r.rawOutputRehydrateBudgetMaxTokens)
}
