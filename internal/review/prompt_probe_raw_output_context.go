package review

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/rawoutputs"
	reviewmodelinput "github.com/susugadx/xelyon-cli/internal/review/modelinput"
	"github.com/susugadx/xelyon-cli/internal/token"
)

func (r *ReviewRunner) renderReviewProbeRawOutputContext(ctx context.Context, ledger ReviewProbeRawOutputLedger, sources []reviewProbeRawOutputSource, probeRefs map[string]rawoutputs.RawOutputRef, commandRefs map[reviewmodelinput.ProbeCommandResultKey]rawoutputs.RawOutputRef, redactor reviewmodelinput.Redactor) (string, ReviewProbeRawOutputLedger) {
	if len(sources) == 0 {
		return "", ledger
	}
	if redactor == nil {
		redactor = reviewProbeRawOutputNoopRedactor{}
	}
	budget := ledger.BudgetTokens
	if budget <= 0 {
		budget = reviewProbeRawOutputDefaultBudgetTokens
	}
	metadataReserve := reviewProbeRawOutputMetadataReserve(budget)
	ledger.MetadataReserveTokens = metadataReserve
	ledger.BodyBudgetTokens = max(0, budget-metadataReserve)
	remainingBodyTokens := ledger.BodyBudgetTokens
	var b strings.Builder
	b.WriteString(reviewProbeRawOutputContextHeader)
	for _, source := range sources {
		ref, ok := reviewProbeRawOutputRefForSource(source, probeRefs, commandRefs)
		if !ok {
			ledger.FailClosedReason = reviewProbeRawOutputReasonRequiredRefMissing
			ledger.CanAcceptSaturated = false
			ledger.MissingRefs = append(ledger.MissingRefs, reviewProbeRawOutputLedgerRefFromSource(source, rawoutputs.RawOutputRef{
				Surface: string(rawoutputs.SurfaceReviewProbeResult),
			}))
			continue
		}
		resolved, err := r.rawOutputArtifactStore.Resolve(ctx, ref)
		if err != nil {
			reason := reviewProbeRawOutputResolveReason(err)
			ledger.FailClosedReason = reason
			ledger.CanAcceptSaturated = false
			ledger.MissingRefs = append(ledger.MissingRefs, reviewProbeRawOutputLedgerRefFromSource(source, ref).withStatus("missing", reason))
			continue
		}
		body, readErr := io.ReadAll(resolved.Body)
		_ = resolved.Body.Close()
		if readErr != nil {
			ledger.FailClosedReason = reviewProbeRawOutputReasonRequiredRefMissing
			ledger.CanAcceptSaturated = false
			ledger.MissingRefs = append(ledger.MissingRefs, reviewProbeRawOutputLedgerRefFromSource(source, ref).withStatus("missing", reviewProbeRawOutputReasonRequiredRefMissing))
			continue
		}
		entry, bodyTokens, reason := renderReviewProbeRawOutputContextEntry(ref, source, string(body), remainingBodyTokens, len(sources) == 1, redactor)
		ledgerRef := reviewProbeRawOutputLedgerRefFromSource(source, ref)
		ledgerRef.BodyTokens = bodyTokens
		switch {
		case reason != "":
			ledgerRef.Status = "budget_exhausted"
			ledgerRef.Reason = reason
			ledger.BudgetExhaustedRefs = append(ledger.BudgetExhaustedRefs, ledgerRef)
			ledger.FailClosedReason = reason
			ledger.CanAcceptSaturated = false
		case strings.TrimSpace(entry) == "":
			ledgerRef.Status = "metadata_only"
			ledgerRef.Reason = reviewProbeRawOutputReasonRequiredRefMetadataOnly
			ledger.MetadataOnlyRefs = append(ledger.MetadataOnlyRefs, ledgerRef)
			ledger.FailClosedReason = reviewProbeRawOutputReasonRequiredRefMetadataOnly
			ledger.CanAcceptSaturated = false
		default:
			b.WriteString("\n")
			b.WriteString(entry)
			remainingBodyTokens -= bodyTokens
			ledgerRef.Status = "rehydrated"
			ledger.RehydratedRefs = append(ledger.RehydratedRefs, ledgerRef)
		}
	}
	if len(ledger.RehydratedRefs) == 0 {
		ledger.CanAcceptSaturated = false
		if ledger.FailClosedReason == "" {
			ledger.FailClosedReason = reviewProbeRawOutputReasonRehydrateUnavailable
		}
		return "", ledger
	}
	return b.String(), ledger
}

func renderReviewProbeRawOutputContextEntry(ref rawoutputs.RawOutputRef, source reviewProbeRawOutputSource, body string, availableBodyTokens int, singleExplicitRef bool, redactor reviewmodelinput.Redactor) (string, int, string) {
	if availableBodyTokens <= 0 {
		return "", 0, reviewProbeRawOutputReasonRequiredRefBodyTooSmall
	}
	maxBodyTokens := reviewProbeRawOutputPerCommandBodyMaxTokens(availableBodyTokens)
	if singleExplicitRef {
		maxBodyTokens = reviewProbeRawOutputSingleExplicitRefMaxTokens(availableBodyTokens)
	}
	bodyBudget := min(availableBodyTokens, maxBodyTokens)
	if bodyBudget < reviewProbeRawOutputRequiredRefBodyMinTokens && token.EstimateTokenCount(body) > bodyBudget {
		return "", bodyBudget, reviewProbeRawOutputReasonRequiredRefBodyTooSmall
	}
	body = rawoutputs.RedactDisplaySecrets(redactor.RedactText(body))
	excerpt := reviewProbeRawOutputBodyExcerpt(body, bodyBudget)
	if strings.TrimSpace(excerpt) == "" {
		return "", 0, reviewProbeRawOutputReasonRequiredRefMetadataOnly
	}
	bodyTokens := token.EstimateTokenCount(excerpt)
	if token.EstimateTokenCount(body) > bodyTokens && bodyTokens < reviewProbeRawOutputRequiredRefBodyMinTokens {
		return "", bodyTokens, reviewProbeRawOutputReasonRequiredRefBodyTooSmall
	}
	var b strings.Builder
	fmt.Fprintf(&b, "- ref: %s\n", ref.RefID)
	fmt.Fprintf(&b, "  surface: %s\n", ref.Surface)
	fmt.Fprintf(&b, "  probe_id: %s\n", source.probeID)
	if source.commandIndex != nil {
		fmt.Fprintf(&b, "  command_index: %d\n", *source.commandIndex)
	}
	fmt.Fprintf(&b, "  command_preview: %s\n", rawoutputs.SanitizeDisplayPreview(redactor.RedactText(reviewProbeRawOutputCommandPreview(source)), reviewProbeRawOutputCommandPreviewRunes))
	fmt.Fprintf(&b, "  status: %s\n", source.command.Status)
	fmt.Fprintf(&b, "  exit_code: %d\n", source.command.ExitCode)
	fmt.Fprintf(&b, "  byte_size: %d\n", ref.ByteSize)
	fmt.Fprintf(&b, "  content_hash: %s\n", ref.ContentHash)
	fmt.Fprintf(&b, "  absorbed_by: %s\n", strings.Join(source.absorbedBy, ", "))
	b.WriteString("  body:\n")
	b.WriteString(indentReviewProbeRawOutputBody(excerpt))
	return b.String(), bodyTokens, ""
}

func reviewProbeRawOutputBodyExcerpt(body string, budgetTokens int) string {
	body = strings.TrimSpace(body)
	if body == "" || budgetTokens <= 0 {
		return ""
	}
	if token.EstimateTokenCount(body) <= budgetTokens {
		return body
	}
	maxRunes := budgetTokens * 2
	if maxRunes < 256 {
		maxRunes = 256
	}
	runes := []rune(body)
	if len(runes) <= maxRunes {
		return body
	}
	headRunes := maxRunes / 2
	tailRunes := maxRunes - headRunes
	head := strings.TrimSpace(string(runes[:headRunes]))
	tail := strings.TrimSpace(string(runes[len(runes)-tailRunes:]))
	return head + "\n...\n" + tail
}

func indentReviewProbeRawOutputBody(body string) string {
	lines := strings.Split(strings.TrimSpace(body), "\n")
	for i, line := range lines {
		lines[i] = "    " + line
	}
	return strings.Join(lines, "\n")
}

func reviewProbeRawOutputRefForSource(source reviewProbeRawOutputSource, probeRefs map[string]rawoutputs.RawOutputRef, commandRefs map[reviewmodelinput.ProbeCommandResultKey]rawoutputs.RawOutputRef) (rawoutputs.RawOutputRef, bool) {
	if source.commandIndex == nil {
		ref, ok := probeRefs[source.probeID]
		return ref, ok
	}
	ref, ok := commandRefs[reviewmodelinput.ProbeCommandResultKey{ProbeID: source.probeID, CommandIndex: *source.commandIndex}]
	return ref, ok
}

func reviewProbeRawOutputLedgerRefFromSource(source reviewProbeRawOutputSource, ref rawoutputs.RawOutputRef) ReviewProbeRawOutputLedgerRef {
	return ReviewProbeRawOutputLedgerRef{
		RefID:        ref.RefID,
		ProbeID:      source.probeID,
		CommandIndex: cloneReviewProbeCommandIndex(source.commandIndex),
		ContentHash:  ref.ContentHash,
		ByteSize:     ref.ByteSize,
		ApproxTokens: ref.ApproxTokens,
	}
}

func cloneReviewProbeCommandIndex(index *int) *int {
	if index == nil {
		return nil
	}
	cloned := *index
	return &cloned
}

func (r ReviewProbeRawOutputLedgerRef) withStatus(status, reason string) ReviewProbeRawOutputLedgerRef {
	r.Status = status
	r.Reason = reason
	return r
}

func (r *ReviewRunner) reviewProbeRawOutputBudget() int {
	budget := r.rawOutputRehydrateBudgetTokens
	if budget <= 0 {
		budget = reviewProbeRawOutputDefaultBudgetTokens
	}
	maxBudget := r.rawOutputRehydrateBudgetMaxTokens
	if maxBudget <= 0 {
		maxBudget = reviewProbeRawOutputDefaultBudgetMaxTokens
	}
	if budget > maxBudget {
		return maxBudget
	}
	return budget
}

func reviewProbeRawOutputMetadataReserve(budget int) int {
	percent := budget * reviewProbeRawOutputMetadataReservePercent / 100
	if percent > reviewProbeRawOutputMetadataReserveTokens {
		return percent
	}
	return reviewProbeRawOutputMetadataReserveTokens
}

func reviewProbeRawOutputPerCommandBodyMaxTokens(available int) int {
	return max(1, available*reviewProbeRawOutputPerCommandBodyMaxPercent/100)
}

func reviewProbeRawOutputSingleExplicitRefMaxTokens(available int) int {
	return max(1, available*reviewProbeRawOutputSingleExplicitRefMaxPercent/100)
}

type reviewProbeRawOutputNoopRedactor struct{}

func (reviewProbeRawOutputNoopRedactor) RedactText(text string) string { return text }

func (reviewProbeRawOutputNoopRedactor) RedactTexts(values []string) []string {
	if len(values) == 0 {
		return []string{}
	}
	return append([]string(nil), values...)
}

func (reviewProbeRawOutputNoopRedactor) RedactPath(path string) string { return path }

func (reviewProbeRawOutputNoopRedactor) RedactPaths(paths []string) []string {
	if len(paths) == 0 {
		return []string{}
	}
	return append([]string(nil), paths...)
}
