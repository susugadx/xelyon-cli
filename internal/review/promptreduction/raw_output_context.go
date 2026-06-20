package promptreduction

import (
	"fmt"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/rawoutputs"
	reviewmodelinput "github.com/susugadx/xelyon-cli/internal/review/modelinput"
	"github.com/susugadx/xelyon-cli/internal/token"
)

const (
	reviewProbeRawOutputContextHeader               = "Review Probe Raw Output Context"
	reviewProbeRawOutputDefaultBudgetTokens         = 4096
	reviewProbeRawOutputDefaultBudgetMaxTokens      = 8192
	reviewProbeRawOutputMetadataReserveTokens       = 512
	reviewProbeRawOutputMetadataReservePercent      = 15
	reviewProbeRawOutputRequiredRefBodyMinTokens    = 512
	reviewProbeRawOutputPerCommandBodyMaxPercent    = 50
	reviewProbeRawOutputSingleExplicitRefMaxPercent = 80
)

const (
	// ReviewProbeRawOutputCommandPreviewRunes は raw output artifact の command preview 上限。
	ReviewProbeRawOutputCommandPreviewRunes = 160
	// ReviewProbeRawOutputReasonArtifactMissing は artifact store / session が使えない場合の保持理由。
	ReviewProbeRawOutputReasonArtifactMissing = "review_probe_raw_output_artifact_missing"
	// ReviewProbeRawOutputReasonRehydrateUnavailable は rehydrate context が成立しない場合の保持理由。
	ReviewProbeRawOutputReasonRehydrateUnavailable = "review_probe_raw_output_rehydrate_not_available"
	// ReviewProbeRawOutputReasonRequiredRefMissing は必須 raw output ref が欠けた場合の保持理由。
	ReviewProbeRawOutputReasonRequiredRefMissing = "review_probe_required_ref_missing"
	// ReviewProbeRawOutputReasonRequiredRefMetadataOnly は body が provider prompt に載らない場合の保持理由。
	ReviewProbeRawOutputReasonRequiredRefMetadataOnly = "review_probe_required_ref_metadata_only"
	// ReviewProbeRawOutputReasonRequiredRefBodyTooSmall は body budget が必須 ref に足りない場合の保持理由。
	ReviewProbeRawOutputReasonRequiredRefBodyTooSmall = "review_probe_required_ref_body_budget_too_small"
	// ReviewProbeRawOutputReasonRequiredRefHashInvalid は raw output artifact hash 検証失敗時の保持理由。
	ReviewProbeRawOutputReasonRequiredRefHashInvalid = "review_probe_required_ref_hash_invalid"
	// ReviewProbeRawOutputReasonRequiredRefQuarantined は quarantined artifact を prompt に戻せない場合の保持理由。
	ReviewProbeRawOutputReasonRequiredRefQuarantined = "review_probe_required_ref_quarantined"
	// ReviewProbeRawOutputReasonBudgetRequiresRevision は artifact quota / size で revision が必要な場合の保持理由。
	ReviewProbeRawOutputReasonBudgetRequiresRevision = "review_probe_budget_requires_blocked_or_needs_revision"
	// ReviewProbeRawOutputReasonSaturatedRejected は rehydrate ledger が fail-closed した saturated 判定の保持理由。
	ReviewProbeRawOutputReasonSaturatedRejected = "review_probe_saturated_rejected_by_rehydrate_ledger"
	// ReviewProbeRawOutputReasonArtifactsDryRun は artifact-backed absorption が dry-run の場合の保持理由。
	ReviewProbeRawOutputReasonArtifactsDryRun = "review_probe_raw_output_artifacts_dry_run"
	// ReviewProbeRawOutputReasonSensitiveOrPrivateKeep は secret/private らしい raw output を保持する理由。
	ReviewProbeRawOutputReasonSensitiveOrPrivateKeep = "review_probe_sensitive_or_private_keep"
	// ReviewProbeRawOutputReasonUnreflectedEvidenceKeep は report evidence に反映されていない raw output の保持理由。
	ReviewProbeRawOutputReasonUnreflectedEvidenceKeep = "review_probe_unreflected_evidence_keep"
)

// ReviewProbeRawOutputContextSource は raw output context の provider-facing source metadata。
type ReviewProbeRawOutputContextSource struct {
	ProbeID        string
	CommandIndex   *int
	CommandPreview string
	Status         string
	ExitCode       int
	AbsorbedBy     []string
}

// ReviewProbeRawOutputContextEntry は rehydrate 済み raw output body と ledger metadata の入力。
type ReviewProbeRawOutputContextEntry struct {
	Ref    rawoutputs.RawOutputRef
	Source ReviewProbeRawOutputContextSource
	Body   string
}

// ReviewProbeRawOutputContextInput は raw output context rendering の入力。
type ReviewProbeRawOutputContextInput struct {
	Ledger   ReviewProbeRawOutputLedger
	Entries  []ReviewProbeRawOutputContextEntry
	Redactor reviewmodelinput.Redactor
}

// NormalizeReviewProbeRawOutputBudget は configured budget と max budget を provider prompt 用に正規化する。
func NormalizeReviewProbeRawOutputBudget(budget, maxBudget int) int {
	if budget <= 0 {
		budget = reviewProbeRawOutputDefaultBudgetTokens
	}
	if maxBudget <= 0 {
		maxBudget = reviewProbeRawOutputDefaultBudgetMaxTokens
	}
	if budget > maxBudget {
		return maxBudget
	}
	return budget
}

// ReviewProbeRawOutputMetadataReserve は raw output context の metadata reserve token 数を返す。
func ReviewProbeRawOutputMetadataReserve(budget int) int {
	percent := budget * reviewProbeRawOutputMetadataReservePercent / 100
	if percent > reviewProbeRawOutputMetadataReserveTokens {
		return percent
	}
	return reviewProbeRawOutputMetadataReserveTokens
}

// ReviewProbeRawOutputLedgerIsEmpty は ledger が report に記録する内容を持たないか判定する。
func ReviewProbeRawOutputLedgerIsEmpty(ledger ReviewProbeRawOutputLedger) bool {
	return ledger.empty()
}

// ReviewProbeRawOutputLedgerPtr は空 ledger を nil に正規化する。
func ReviewProbeRawOutputLedgerPtr(ledger ReviewProbeRawOutputLedger) *ReviewProbeRawOutputLedger {
	if ReviewProbeRawOutputLedgerIsEmpty(ledger) {
		return nil
	}
	return &ledger
}

// NewReviewProbeRawOutputLedgerRef は raw output source と artifact ref から ledger ref を構築する。
func NewReviewProbeRawOutputLedgerRef(source ReviewProbeRawOutputContextSource, ref rawoutputs.RawOutputRef) ReviewProbeRawOutputLedgerRef {
	return ReviewProbeRawOutputLedgerRef{
		RefID:        ref.RefID,
		ProbeID:      source.ProbeID,
		CommandIndex: cloneReviewProbeCommandIndex(source.CommandIndex),
		ContentHash:  ref.ContentHash,
		ByteSize:     ref.ByteSize,
		ApproxTokens: ref.ApproxTokens,
	}
}

// ReviewProbeRawOutputLedgerRefWithStatus は ledger ref に status と reason を付与する。
func ReviewProbeRawOutputLedgerRefWithStatus(ref ReviewProbeRawOutputLedgerRef, status, reason string) ReviewProbeRawOutputLedgerRef {
	ref.Status = status
	ref.Reason = reason
	return ref
}

// RenderReviewProbeRawOutputContext は artifact body を provider-facing raw output context と ledger へ変換する。
func RenderReviewProbeRawOutputContext(input ReviewProbeRawOutputContextInput) (string, ReviewProbeRawOutputLedger) {
	ledger := input.Ledger
	if len(input.Entries) == 0 {
		return "", ledger
	}
	redactor := normalizeReviewProbeRawOutputRedactor(input.Redactor)
	budget := ledger.BudgetTokens
	if budget <= 0 {
		budget = reviewProbeRawOutputDefaultBudgetTokens
	}
	metadataReserve := ReviewProbeRawOutputMetadataReserve(budget)
	ledger.MetadataReserveTokens = metadataReserve
	ledger.BodyBudgetTokens = max(0, budget-metadataReserve)
	remainingBodyTokens := ledger.BodyBudgetTokens
	var b strings.Builder
	b.WriteString(reviewProbeRawOutputContextHeader)
	for _, entry := range input.Entries {
		rendered, bodyTokens, reason := renderReviewProbeRawOutputContextEntry(entry, remainingBodyTokens, len(input.Entries) == 1, redactor)
		ledgerRef := NewReviewProbeRawOutputLedgerRef(entry.Source, entry.Ref)
		ledgerRef.BodyTokens = bodyTokens
		switch {
		case reason != "":
			ledgerRef.Status = "budget_exhausted"
			ledgerRef.Reason = reason
			ledger.BudgetExhaustedRefs = append(ledger.BudgetExhaustedRefs, ledgerRef)
			ledger.FailClosedReason = reason
			ledger.CanAcceptSaturated = false
		case strings.TrimSpace(rendered) == "":
			ledgerRef.Status = "metadata_only"
			ledgerRef.Reason = ReviewProbeRawOutputReasonRequiredRefMetadataOnly
			ledger.MetadataOnlyRefs = append(ledger.MetadataOnlyRefs, ledgerRef)
			ledger.FailClosedReason = ReviewProbeRawOutputReasonRequiredRefMetadataOnly
			ledger.CanAcceptSaturated = false
		default:
			b.WriteString("\n")
			b.WriteString(rendered)
			remainingBodyTokens -= bodyTokens
			ledgerRef.Status = "rehydrated"
			ledger.RehydratedRefs = append(ledger.RehydratedRefs, ledgerRef)
		}
	}
	if len(ledger.RehydratedRefs) == 0 {
		ledger.CanAcceptSaturated = false
		if ledger.FailClosedReason == "" {
			ledger.FailClosedReason = ReviewProbeRawOutputReasonRehydrateUnavailable
		}
		return "", ledger
	}
	return b.String(), ledger
}

func renderReviewProbeRawOutputContextEntry(entry ReviewProbeRawOutputContextEntry, availableBodyTokens int, singleExplicitRef bool, redactor reviewmodelinput.Redactor) (string, int, string) {
	if availableBodyTokens <= 0 {
		return "", 0, ReviewProbeRawOutputReasonRequiredRefBodyTooSmall
	}
	maxBodyTokens := reviewProbeRawOutputPerCommandBodyMaxTokens(availableBodyTokens)
	if singleExplicitRef {
		maxBodyTokens = reviewProbeRawOutputSingleExplicitRefMaxTokens(availableBodyTokens)
	}
	bodyBudget := min(availableBodyTokens, maxBodyTokens)
	if bodyBudget < reviewProbeRawOutputRequiredRefBodyMinTokens && token.EstimateTokenCount(entry.Body) > bodyBudget {
		return "", bodyBudget, ReviewProbeRawOutputReasonRequiredRefBodyTooSmall
	}
	body := rawoutputs.RedactDisplaySecrets(redactor.RedactText(entry.Body))
	excerpt := reviewProbeRawOutputBodyExcerpt(body, bodyBudget)
	if strings.TrimSpace(excerpt) == "" {
		return "", 0, ReviewProbeRawOutputReasonRequiredRefMetadataOnly
	}
	bodyTokens := token.EstimateTokenCount(excerpt)
	if token.EstimateTokenCount(body) > bodyTokens && bodyTokens < reviewProbeRawOutputRequiredRefBodyMinTokens {
		return "", bodyTokens, ReviewProbeRawOutputReasonRequiredRefBodyTooSmall
	}
	source := entry.Source
	var b strings.Builder
	fmt.Fprintf(&b, "- ref: %s\n", entry.Ref.RefID)
	fmt.Fprintf(&b, "  surface: %s\n", entry.Ref.Surface)
	fmt.Fprintf(&b, "  probe_id: %s\n", source.ProbeID)
	if source.CommandIndex != nil {
		fmt.Fprintf(&b, "  command_index: %d\n", *source.CommandIndex)
	}
	fmt.Fprintf(&b, "  command_preview: %s\n", rawoutputs.SanitizeDisplayPreview(redactor.RedactText(source.CommandPreview), ReviewProbeRawOutputCommandPreviewRunes))
	fmt.Fprintf(&b, "  status: %s\n", source.Status)
	fmt.Fprintf(&b, "  exit_code: %d\n", source.ExitCode)
	fmt.Fprintf(&b, "  byte_size: %d\n", entry.Ref.ByteSize)
	fmt.Fprintf(&b, "  content_hash: %s\n", entry.Ref.ContentHash)
	fmt.Fprintf(&b, "  absorbed_by: %s\n", strings.Join(source.AbsorbedBy, ", "))
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

func reviewProbeRawOutputPerCommandBodyMaxTokens(available int) int {
	return max(1, available*reviewProbeRawOutputPerCommandBodyMaxPercent/100)
}

func reviewProbeRawOutputSingleExplicitRefMaxTokens(available int) int {
	return max(1, available*reviewProbeRawOutputSingleExplicitRefMaxPercent/100)
}

func cloneReviewProbeCommandIndex(index *int) *int {
	if index == nil {
		return nil
	}
	cloned := *index
	return &cloned
}

func normalizeReviewProbeRawOutputRedactor(redactor reviewmodelinput.Redactor) reviewmodelinput.Redactor {
	if redactor == nil {
		return reviewProbeRawOutputNoopRedactor{}
	}
	return redactor
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
