package promptreduction

import "strings"

// ReviewPromptReductionMode は review model prompt 専用の provider-facing 削減 mode。
type ReviewPromptReductionMode string

const (
	// ReviewPromptReductionModeOff は review prompt 削減を無効化する。
	ReviewPromptReductionModeOff ReviewPromptReductionMode = "off"
	// ReviewPromptReductionModeDryRun は review prompt を変えない。削減見込みの owner は呼び出し側に置く。
	ReviewPromptReductionModeDryRun ReviewPromptReductionMode = "dry_run"
	// ReviewPromptReductionModeApply は review prompt 内の安全な probe output compact を適用する。
	ReviewPromptReductionModeApply ReviewPromptReductionMode = "apply"
)

// NormalizeReviewPromptReductionMode は unknown mode を off に正規化する。
func NormalizeReviewPromptReductionMode(mode ReviewPromptReductionMode) ReviewPromptReductionMode {
	switch mode {
	case ReviewPromptReductionModeApply:
		return ReviewPromptReductionModeApply
	case ReviewPromptReductionModeDryRun:
		return ReviewPromptReductionModeDryRun
	default:
		return ReviewPromptReductionModeOff
	}
}

func (m ReviewPromptReductionMode) compactCommandOutputs() bool {
	return NormalizeReviewPromptReductionMode(m) == ReviewPromptReductionModeApply
}

// ReviewRawOutputArtifactsMode は review prompt 用 raw output artifact-backed absorption の mode。
type ReviewRawOutputArtifactsMode string

const (
	// ReviewRawOutputArtifactsModeOff は review raw output artifact-backed absorption を無効化する。
	ReviewRawOutputArtifactsModeOff ReviewRawOutputArtifactsMode = "off"
	// ReviewRawOutputArtifactsModeDryRun は artifact/ref/ledger 候補だけを記録し prompt は raw のままにする。
	ReviewRawOutputArtifactsModeDryRun ReviewRawOutputArtifactsMode = "dry_run"
	// ReviewRawOutputArtifactsModeApply は rehydrate ledger が成立した場合だけ prompt absorption を適用する。
	ReviewRawOutputArtifactsModeApply ReviewRawOutputArtifactsMode = "apply"
)

// NormalizeReviewRawOutputArtifactsMode は unknown mode を off に正規化する。
func NormalizeReviewRawOutputArtifactsMode(mode ReviewRawOutputArtifactsMode) ReviewRawOutputArtifactsMode {
	switch mode {
	case ReviewRawOutputArtifactsModeApply:
		return ReviewRawOutputArtifactsModeApply
	case ReviewRawOutputArtifactsModeDryRun:
		return ReviewRawOutputArtifactsModeDryRun
	default:
		return ReviewRawOutputArtifactsModeOff
	}
}

// ReviewPromptReductionFamily は review prompt 内で削減候補を分類する family。
type ReviewPromptReductionFamily string

const (
	// ReviewPromptReductionFamilyProbePlan は probe plan 中間成果物。
	ReviewPromptReductionFamilyProbePlan ReviewPromptReductionFamily = "probe_plan"
	// ReviewPromptReductionFamilyProbeResult は probe result 中間成果物。
	ReviewPromptReductionFamilyProbeResult ReviewPromptReductionFamily = "probe_result"
	// ReviewPromptReductionFamilyRelatedContext は related context 中間成果物。
	ReviewPromptReductionFamilyRelatedContext ReviewPromptReductionFamily = "related_context"
	// ReviewPromptReductionFamilyExternalDoc は external_doc / web evidence 中間成果物。
	ReviewPromptReductionFamilyExternalDoc ReviewPromptReductionFamily = "external_doc"
	// ReviewPromptReductionFamilyReportDraft は report draft 中間成果物。
	ReviewPromptReductionFamilyReportDraft ReviewPromptReductionFamily = "report_draft"
	// ReviewPromptReductionFamilyScopeCoverage は scope coverage 中間成果物。
	ReviewPromptReductionFamilyScopeCoverage ReviewPromptReductionFamily = "scope_coverage"
	// ReviewPromptReductionFamilyGitOutput は git / command output 中間成果物。
	ReviewPromptReductionFamilyGitOutput ReviewPromptReductionFamily = "git_output"
)

// ReviewPromptReductionItemStatus は compact 可否を決める item 状態。
type ReviewPromptReductionItemStatus string

const (
	// ReviewPromptReductionItemCandidate は削減候補だが prompt には未適用の item。
	ReviewPromptReductionItemCandidate ReviewPromptReductionItemStatus = "candidate"
	// ReviewPromptReductionItemCurrent は現在 prompt で必要な item。compact 禁止。
	ReviewPromptReductionItemCurrent ReviewPromptReductionItemStatus = "current"
	// ReviewPromptReductionItemAbsorbed は後続 state に吸収済みの item。
	ReviewPromptReductionItemAbsorbed ReviewPromptReductionItemStatus = "absorbed"
	// ReviewPromptReductionItemRehydrated は raw artifact から provider prompt へ rehydrate 済みの item。
	ReviewPromptReductionItemRehydrated ReviewPromptReductionItemStatus = "rehydrated"
	// ReviewPromptReductionItemUnresolved は未解決 risk と関係する item。compact 禁止。
	ReviewPromptReductionItemUnresolved ReviewPromptReductionItemStatus = "unresolved"
	// ReviewPromptReductionItemFindingEvidence は finding evidence。compact 禁止。
	ReviewPromptReductionItemFindingEvidence ReviewPromptReductionItemStatus = "finding_evidence"
	// ReviewPromptReductionItemUnknown は判定不能。compact 禁止。
	ReviewPromptReductionItemUnknown ReviewPromptReductionItemStatus = "unknown"
)

// ReviewPromptAbsorptionRef は compact item の吸収先を表す。
type ReviewPromptAbsorptionRef struct {
	Kind string `json:"kind"`
	ID   string `json:"id,omitempty"`
}

// ReviewPromptReductionItem は review prompt payload 内の削減候補状態。
type ReviewPromptReductionItem struct {
	ID                string                          `json:"id"`
	Family            ReviewPromptReductionFamily     `json:"family"`
	Phase             ReviewModelPhase                `json:"phase"`
	Status            ReviewPromptReductionItemStatus `json:"status"`
	AbsorbedBy        []ReviewPromptAbsorptionRef     `json:"absorbed_by,omitempty"`
	EvidenceRefs      []ReviewEvidenceRef             `json:"evidence_refs,omitempty"`
	RawArtifactRef    string                          `json:"raw_artifact_ref,omitempty"`
	PromptArtifactRef string                          `json:"prompt_artifact_ref,omitempty"`
	Summary           string                          `json:"summary,omitempty"`
	OriginalBytes     int                             `json:"original_bytes,omitempty"`
	ReplacementBytes  int                             `json:"replacement_bytes,omitempty"`
}

// ReviewProbeRawOutputLedgerRef は review probe raw output ref の rehydrate ledger 行。
type ReviewProbeRawOutputLedgerRef struct {
	RefID        string `json:"ref_id"`
	ProbeID      string `json:"probe_id,omitempty"`
	CommandIndex *int   `json:"command_index,omitempty"`
	ContentHash  string `json:"content_hash,omitempty"`
	ByteSize     int    `json:"byte_size,omitempty"`
	ApproxTokens int    `json:"approx_tokens,omitempty"`
	BodyTokens   int    `json:"body_tokens,omitempty"`
	Status       string `json:"status,omitempty"`
	Reason       string `json:"reason,omitempty"`
}

// ReviewProbeRawOutputLedger は saturation / revision prompt の raw output rehydrate 成否を表す。
type ReviewProbeRawOutputLedger struct {
	ReviewRunID           string                          `json:"review_run_id,omitempty"`
	Phase                 ReviewModelPhase                `json:"phase,omitempty"`
	PromptKind            string                          `json:"prompt_kind,omitempty"`
	BudgetTokens          int                             `json:"budget_tokens,omitempty"`
	MetadataReserveTokens int                             `json:"metadata_reserve_tokens,omitempty"`
	BodyBudgetTokens      int                             `json:"body_budget_tokens,omitempty"`
	RequiredRefs          []ReviewProbeRawOutputLedgerRef `json:"required_refs,omitempty"`
	OptionalRefs          []ReviewProbeRawOutputLedgerRef `json:"optional_refs,omitempty"`
	RehydratedRefs        []ReviewProbeRawOutputLedgerRef `json:"rehydrated_refs,omitempty"`
	MetadataOnlyRefs      []ReviewProbeRawOutputLedgerRef `json:"metadata_only_refs,omitempty"`
	MissingRefs           []ReviewProbeRawOutputLedgerRef `json:"missing_refs,omitempty"`
	BudgetExhaustedRefs   []ReviewProbeRawOutputLedgerRef `json:"budget_exhausted_refs,omitempty"`
	FailClosedReason      string                          `json:"fail_closed_reason,omitempty"`
	CanAcceptSaturated    bool                            `json:"can_accept_saturated"`
}

func (l ReviewProbeRawOutputLedger) empty() bool {
	return len(l.RequiredRefs) == 0 &&
		len(l.OptionalRefs) == 0 &&
		len(l.RehydratedRefs) == 0 &&
		len(l.MetadataOnlyRefs) == 0 &&
		len(l.MissingRefs) == 0 &&
		len(l.BudgetExhaustedRefs) == 0 &&
		strings.TrimSpace(l.FailClosedReason) == ""
}

// ReviewStateSummary は compact 後も残す current review state の deterministic index。
type ReviewStateSummary struct {
	Target                   string
	ChangedFiles             []string
	ImpactSurfaces           []string
	CandidateRisks           []string
	UnresolvedRisks          []string
	ConfirmedFindings        []string
	FindingEvidenceRefs      []string
	DismissedRisks           []string
	ScopeCoverage            []string
	ExternalEvidence         []string
	LatestReportStatus       string
	SaturationStatus         string
	NextProbeFocus           []string
	AbsorbedIntermediateRefs []string
}

// PromptText は provider-facing prompt へ入れる deterministic summary を返す。
func (s ReviewStateSummary) PromptText() string {
	var b strings.Builder
	appendReviewStateSummaryLine(&b, "target", s.Target)
	appendReviewStateSummaryList(&b, "changed_files", s.ChangedFiles)
	appendReviewStateSummaryList(&b, "impact_surfaces", s.ImpactSurfaces)
	appendReviewStateSummaryList(&b, "candidate_risks", s.CandidateRisks)
	appendReviewStateSummaryList(&b, "unresolved_risks", s.UnresolvedRisks)
	appendReviewStateSummaryList(&b, "confirmed_findings", s.ConfirmedFindings)
	appendReviewStateSummaryList(&b, "finding_evidence_refs", s.FindingEvidenceRefs)
	appendReviewStateSummaryList(&b, "dismissed_risks", s.DismissedRisks)
	appendReviewStateSummaryList(&b, "scope_coverage", s.ScopeCoverage)
	appendReviewStateSummaryList(&b, "external_evidence", s.ExternalEvidence)
	appendReviewStateSummaryLine(&b, "latest_report_status", s.LatestReportStatus)
	appendReviewStateSummaryLine(&b, "saturation_status", s.SaturationStatus)
	appendReviewStateSummaryList(&b, "next_probe_focus", s.NextProbeFocus)
	appendReviewStateSummaryList(&b, "absorbed_intermediate", s.AbsorbedIntermediateRefs)
	return strings.TrimSpace(b.String())
}

// ReviewPromptReductionState は ReviewRunner.Run 1 回分の prompt 削減状態。
type ReviewPromptReductionState struct {
	Mode    ReviewPromptReductionMode
	Summary ReviewStateSummary
	Items   []ReviewPromptReductionItem
	Report  ReviewPromptReductionReport
}

func appendReviewStateSummaryLine(b *strings.Builder, key, value string) {
	value = strings.TrimSpace(value)
	if value == "" {
		return
	}
	b.WriteString(key)
	b.WriteString(": ")
	b.WriteString(value)
	b.WriteByte('\n')
}

func appendReviewStateSummaryList(b *strings.Builder, key string, values []string) {
	if len(values) == 0 {
		return
	}
	b.WriteString(key)
	b.WriteString(":\n")
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		b.WriteString("- ")
		b.WriteString(value)
		b.WriteByte('\n')
	}
}

// ReviewPromptAbsorptionRefsFromOwners は owner string を reduction item の吸収先 ref に変換する。
func ReviewPromptAbsorptionRefsFromOwners(owners []string) []ReviewPromptAbsorptionRef {
	if len(owners) == 0 {
		return nil
	}
	refs := make([]ReviewPromptAbsorptionRef, 0, len(owners))
	for _, owner := range owners {
		owner = strings.TrimSpace(owner)
		if owner == "" {
			continue
		}
		switch {
		case strings.HasPrefix(owner, "scope_coverage.surface."):
			refs = append(refs, ReviewPromptAbsorptionRef{
				Kind: "scope_coverage.surface",
				ID:   strings.TrimPrefix(owner, "scope_coverage.surface."),
			})
		case strings.HasPrefix(owner, "scope_coverage.risk."):
			refs = append(refs, ReviewPromptAbsorptionRef{
				Kind: "scope_coverage.risk",
				ID:   strings.TrimPrefix(owner, "scope_coverage.risk."),
			})
		default:
			refs = append(refs, ReviewPromptAbsorptionRef{Kind: owner})
		}
	}
	return refs
}
