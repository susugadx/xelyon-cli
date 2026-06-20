package modelinput

import (
	"strings"

	reviewprobe "github.com/susugadx/xelyon-cli/internal/review/probe"
	reviewprobeplan "github.com/susugadx/xelyon-cli/internal/review/probeplan"
	reviewreport "github.com/susugadx/xelyon-cli/internal/review/report"
)

// SaturationCheckPromptInput は final report saturation check prompt の入力 DTO。
type SaturationCheckPromptInput struct {
	CustomInstructions          string
	ReviewStateSummary          string
	EvidenceMarkdown            string
	Plan                        reviewprobeplan.ReviewProbePlan
	ProbeSummaries              []reviewreport.ReviewProbeSummary
	ProbeResults                []reviewprobe.ReviewProbeResult
	Redactor                    Redactor
	ProbeResultOptions          ProbeResultPromptContextOptions
	ReviewProbeRawOutputContext string
	ReviewProbeRawOutputLedger  any
	FinalizedReport             reviewreport.ReviewReport
}

// SaturationCheckRepairPromptInput は saturation check repair prompt の入力 DTO。
type SaturationCheckRepairPromptInput struct {
	CustomInstructions          string
	ReviewStateSummary          string
	EvidenceMarkdown            string
	Plan                        reviewprobeplan.ReviewProbePlan
	ProbeSummaries              []reviewreport.ReviewProbeSummary
	ProbeResults                []reviewprobe.ReviewProbeResult
	Redactor                    Redactor
	ProbeResultOptions          ProbeResultPromptContextOptions
	ReviewProbeRawOutputContext string
	ReviewProbeRawOutputLedger  any
	FinalizedReport             reviewreport.ReviewReport
	InvalidOutput               string
	DecodeOrValidationErr       error
}

// ReportRevisionPromptInput は saturation check 後の report revision prompt 入力 DTO。
type ReportRevisionPromptInput struct {
	CustomInstructions          string
	ReviewStateSummary          string
	EvidenceMarkdown            string
	Plan                        reviewprobeplan.ReviewProbePlan
	ProbeSummaries              []reviewreport.ReviewProbeSummary
	ProbeResults                []reviewprobe.ReviewProbeResult
	Redactor                    Redactor
	ProbeResultOptions          ProbeResultPromptContextOptions
	ReviewProbeRawOutputContext string
	ReviewProbeRawOutputLedger  any
	FinalizedReport             reviewreport.ReviewReport
	SaturationCheck             reviewreport.ReviewSaturationCheck
}

// ReportRevisionRepairPromptInput は report revision repair prompt の入力 DTO。
type ReportRevisionRepairPromptInput struct {
	CustomInstructions          string
	ReviewStateSummary          string
	EvidenceMarkdown            string
	Plan                        reviewprobeplan.ReviewProbePlan
	ProbeSummaries              []reviewreport.ReviewProbeSummary
	ProbeResults                []reviewprobe.ReviewProbeResult
	Redactor                    Redactor
	ProbeResultOptions          ProbeResultPromptContextOptions
	ReviewProbeRawOutputContext string
	ReviewProbeRawOutputLedger  any
	FinalizedReport             reviewreport.ReviewReport
	SaturationCheck             reviewreport.ReviewSaturationCheck
	InvalidRevisionOutput       string
	DecodeOrValidationErr       error
}

// BuildSaturationCheckPrompt は final report saturation check 用の deterministic prompt を組み立てる。
func BuildSaturationCheckPrompt(input SaturationCheckPromptInput) string {
	redactor := normalizeRedactor(input.Redactor)

	var b strings.Builder
	b.WriteString("# Review Final Report Saturation Check\n\n")
	b.WriteString("Return exactly one JSON object for schema ")
	b.WriteString(reviewreport.ReviewSaturationCheckSchemaVersionV1)
	b.WriteString(". Do not include markdown or explanatory text outside the JSON.\n\n")
	b.WriteString("This is not a new review. Check whether the finalized report fully processed the Pass1 scope and the supplied evidence/probe context. Do not request tools, perform additional exploration, or add speculative findings.\n\n")

	appendReviewRunnerPromptStrictReviewerStance(&b, reviewRunnerPromptPostProbeInsufficientEvidenceGuidance)
	appendReviewRunnerPromptTextSection(&b, "Saturation Check JSON Contract", reviewSaturationCheckPromptContract())
	appendReviewRunnerPromptTextSection(&b, "Custom Instructions", input.CustomInstructions)
	appendReviewRunnerPromptOptionalTextSection(&b, "Review State Summary", input.ReviewStateSummary)
	appendReviewRunnerPromptMarkdownSection(&b, "Evidence Markdown", input.EvidenceMarkdown)
	appendReviewRunnerPromptJSONSection(&b, "Decoded Probe Plan", input.Plan)
	appendReviewRunnerPromptJSONSection(&b, "Probe Summaries For Report Schema", redactReviewProbeSummariesForPrompt(input.ProbeSummaries, redactor))
	appendReviewRunnerPromptJSONSection(&b, "Probe Result Context", BuildProbeResultPromptContextsWithOptions(input.ProbeResults, redactor, input.ProbeResultOptions))
	appendReviewRunnerPromptOptionalTextSection(&b, "Review Probe Raw Output Context", input.ReviewProbeRawOutputContext)
	appendReviewRunnerPromptOptionalJSONSection(&b, "Review Probe Raw Output Rehydrate Ledger", input.ReviewProbeRawOutputLedger)
	appendReviewRunnerPromptRedactedJSONSection(&b, "Finalized Review Report", input.FinalizedReport, redactor)
	if input.FinalizedReport.ComputedSummary != nil {
		appendReviewRunnerPromptJSONSection(&b, "Computed Summary", input.FinalizedReport.ComputedSummary)
	}
	return b.String()
}

// BuildSaturationCheckRepairPrompt は saturation check の strict decode/validation repair prompt を組み立てる。
func BuildSaturationCheckRepairPrompt(input SaturationCheckRepairPromptInput) string {
	redactor := normalizeRedactor(input.Redactor)

	var b strings.Builder
	b.WriteString("# Review Final Report Saturation Check JSON Repair\n\n")
	b.WriteString("The previous saturation check response failed strict JSON decode or validation. ")
	b.WriteString(reviewRunnerJSONRepairOutputInstructions)
	b.WriteString("\n\n")
	b.WriteString("Keep the same finalized report and Pass1 scope. ")
	b.WriteString(reviewRunnerJSONRepairScopeInstruction)
	b.WriteString("\n\n")

	appendReviewRunnerPromptStrictReviewerStance(&b, reviewRunnerPromptPostProbeInsufficientEvidenceGuidance)
	appendReviewRunnerPromptTextSection(&b, "Saturation Check JSON Contract", reviewSaturationCheckPromptContract())
	appendReviewRunnerPromptTextSection(&b, "Custom Instructions", input.CustomInstructions)
	appendReviewRunnerPromptOptionalTextSection(&b, "Review State Summary", input.ReviewStateSummary)
	appendReviewRunnerPromptMarkdownSection(&b, "Evidence Markdown", input.EvidenceMarkdown)
	appendReviewRunnerPromptJSONSection(&b, "Decoded Probe Plan", input.Plan)
	appendReviewRunnerPromptJSONSection(&b, "Probe Summaries For Report Schema", redactReviewProbeSummariesForPrompt(input.ProbeSummaries, redactor))
	appendReviewRunnerPromptJSONSection(&b, "Probe Result Context", BuildProbeResultPromptContextsWithOptions(input.ProbeResults, redactor, input.ProbeResultOptions))
	appendReviewRunnerPromptOptionalTextSection(&b, "Review Probe Raw Output Context", input.ReviewProbeRawOutputContext)
	appendReviewRunnerPromptOptionalJSONSection(&b, "Review Probe Raw Output Rehydrate Ledger", input.ReviewProbeRawOutputLedger)
	appendReviewRunnerPromptRedactedJSONSection(&b, "Finalized Review Report", input.FinalizedReport, redactor)
	if input.FinalizedReport.ComputedSummary != nil {
		appendReviewRunnerPromptJSONSection(&b, "Computed Summary", input.FinalizedReport.ComputedSummary)
	}
	appendReviewRunnerPromptTextSection(&b, "Invalid Model Output", redactor.RedactText(input.InvalidOutput))
	appendReviewRunnerPromptTextSection(&b, "Decode Or Validation Error", redactor.RedactText(reviewRunnerPromptErrorText(input.DecodeOrValidationErr)))
	return b.String()
}

// BuildReportRevisionPrompt は saturation check に基づく report revision prompt を組み立てる。
func BuildReportRevisionPrompt(input ReportRevisionPromptInput) string {
	var b strings.Builder
	b.WriteString("# Review Pass 2: Report Revision\n\n")
	b.WriteString("Return exactly one JSON object for schema ")
	b.WriteString(reviewreport.ReviewReportSchemaVersionV2)
	b.WriteString(". Do not include markdown or explanatory text outside the JSON.\n\n")
	b.WriteString("Revise the supplied finalized report only to address the saturation check. Use only the supplied evidence, decoded probe plan, probe summaries, and probe result context. Do not perform a new review, do not request tools, and do not add speculative findings. The original finalized report may include runner-computed computed_summary as context; do not output top-level computed_summary in the revised report.\n\n")

	appendReviewRunnerPromptStrictReviewerStance(&b, reviewRunnerPromptPostProbeInsufficientEvidenceGuidance)
	appendReviewRunnerPromptTextSection(&b, "Review Report JSON Contract", reviewReportPromptContract())
	appendReviewRunnerPromptTextSection(&b, "Saturation Revision Guardrails", reviewReportRevisionSaturationGuardrails())
	appendReviewReportRevisionPromptContext(&b, input)
	return b.String()
}

// BuildReportRevisionRepairPrompt は report revision の strict decode/validation repair prompt を組み立てる。
func BuildReportRevisionRepairPrompt(input ReportRevisionRepairPromptInput) string {
	redactor := normalizeRedactor(input.Redactor)

	var b strings.Builder
	b.WriteString("# Review Pass 2: Report Revision JSON Repair\n\n")
	b.WriteString("The previous report revision response failed strict JSON decode or validation. Return corrected ")
	b.WriteString(reviewreport.ReviewReportSchemaVersionV2)
	b.WriteString(" JSON only. Do not add markdown fences. Do not change schema_version from the contract value. Do not request or rely on tools. Preserve trusted probe summary IDs; do not invent probe IDs.\n\n")
	b.WriteString("Use the same evidence, decoded probe plan, probe summaries, probe result context, original finalized report, and saturation check. Do not perform a new review. Only repair the revision so it satisfies the report contract and saturation check. Do not output computed_summary.\n\n")

	appendReviewRunnerPromptStrictReviewerStance(&b, reviewRunnerPromptPostProbeInsufficientEvidenceGuidance)
	appendReviewRunnerPromptTextSection(&b, "Review Report JSON Contract", reviewReportPromptContract())
	appendReviewRunnerPromptTextSection(&b, "Saturation Revision Guardrails", reviewReportRevisionSaturationGuardrails())
	appendReviewReportRevisionPromptContext(&b, ReportRevisionPromptInput{
		CustomInstructions:          input.CustomInstructions,
		EvidenceMarkdown:            input.EvidenceMarkdown,
		Plan:                        input.Plan,
		ProbeSummaries:              input.ProbeSummaries,
		ProbeResults:                input.ProbeResults,
		Redactor:                    redactor,
		ProbeResultOptions:          input.ProbeResultOptions,
		ReviewProbeRawOutputContext: input.ReviewProbeRawOutputContext,
		ReviewProbeRawOutputLedger:  input.ReviewProbeRawOutputLedger,
		FinalizedReport:             input.FinalizedReport,
		SaturationCheck:             input.SaturationCheck,
		ReviewStateSummary:          input.ReviewStateSummary,
	})
	appendReviewRunnerPromptTextSection(&b, "Invalid Model Output", redactor.RedactText(input.InvalidRevisionOutput))
	appendReviewRunnerPromptTextSection(&b, "Decode Or Validation Error", redactor.RedactText(reviewRunnerPromptErrorText(input.DecodeOrValidationErr)))
	return b.String()
}

func reviewReportRevisionSaturationGuardrails() string {
	return `Revise only the issues identified by the supplied saturation_check. Do not broaden the review, do not add unrelated findings, and do not rewrite already-correct report sections.
Preserve review_report.v2 schema shape exactly. Do not output top-level "computed_summary".
Preserve the trusted probe_summaries entries supplied for the report schema with the same count, same order, and same probe_id values.
Do not convert failed, blocked, timed-out, or mutated_worktree probe outcomes into clean, checked, dismissed, or verified coverage. Represent them as a finding, residual_risk, unverified scope, or blocked status when the evidence requires it.
Deterministic coverage audit feedback in saturation_check is not a mandate to create a finding. Use it to re-check the supplied evidence and then reflect the result as a finding, no-finding rationale, scope_coverage evidence/summary, residual risk, unverified status, or blocked status.
Do not use shallow or empty related context/search evidence as proof of no impact. Raw web search results are discovery-only. Fetched external_doc snippets are citation-capable evidence, but are not automatically official documentation or confirmed external specs. Source credibility values are "official_candidate", "third_party", and "unknown"; only "official_candidate" may be treated as an official-source candidate, not proof by itself. Do not infer official or authoritative status from search query wording, source title, URL label, snippet wording, or "official documentation" text alone. Do not treat them as confirmed official documentation when source_credibility is "unknown" or "third_party"; source_credibility and the cited snippet content must both support that claim, and external_support.level/official_confirmation must allow confirmed official status. Disabled, failed, truncated, or inconclusive review web search evidence is not external spec coverage. If credibility, truncation, or evidence gaps prevent verification, reflect that as residual, unverified, or blocked within review_report.v2.
` + reviewExternalSupportPromptGuardrails()
}

func appendReviewReportRevisionPromptContext(b *strings.Builder, input ReportRevisionPromptInput) {
	redactor := normalizeRedactor(input.Redactor)
	appendReviewRunnerPromptTextSection(b, "Custom Instructions", input.CustomInstructions)
	appendReviewRunnerPromptOptionalTextSection(b, "Review State Summary", input.ReviewStateSummary)
	appendReviewRunnerPromptMarkdownSection(b, "Evidence Markdown", input.EvidenceMarkdown)
	appendReviewRunnerPromptJSONSection(b, "Decoded Probe Plan", input.Plan)
	appendReviewRunnerPromptJSONSection(b, "Probe Summaries For Report Schema", redactReviewProbeSummariesForPrompt(input.ProbeSummaries, redactor))
	appendReviewRunnerPromptJSONSection(b, "Probe Result Context", BuildProbeResultPromptContextsWithOptions(input.ProbeResults, redactor, input.ProbeResultOptions))
	appendReviewRunnerPromptOptionalTextSection(b, "Review Probe Raw Output Context", input.ReviewProbeRawOutputContext)
	appendReviewRunnerPromptOptionalJSONSection(b, "Review Probe Raw Output Rehydrate Ledger", input.ReviewProbeRawOutputLedger)
	appendReviewRunnerPromptRedactedJSONSection(b, "Original Finalized Review Report", input.FinalizedReport, redactor)
	appendReviewRunnerPromptJSONSection(b, "Saturation Check", input.SaturationCheck)
}

func appendReviewRunnerPromptRedactedJSONSection(b *strings.Builder, title string, value any, redactor Redactor) {
	redactor = normalizeRedactor(redactor)
	appendReviewRunnerPromptSection(b, title)
	data, err := marshalReviewJSONIndent(value)
	if err != nil {
		appendReviewRunnerPromptFence(b, "json", "null")
		return
	}
	appendReviewRunnerPromptFence(b, "json", redactor.RedactText(string(data)))
}
