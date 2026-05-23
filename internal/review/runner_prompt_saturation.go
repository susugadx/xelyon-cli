package review

import "strings"

func buildReviewSaturationCheckPrompt(req ReviewRequest, evidenceMarkdown string, plan ReviewProbePlan, probeSummaries []ReviewProbeSummary, results []ReviewProbeResult, redactor reviewRunnerPromptRedactor, finalizedReport ReviewReport) string {
	var b strings.Builder
	b.WriteString("# Review Final Report Saturation Check\n\n")
	b.WriteString("Return exactly one JSON object for schema ")
	b.WriteString(ReviewSaturationCheckSchemaVersionV1)
	b.WriteString(". Do not include markdown or explanatory text outside the JSON.\n\n")
	b.WriteString("This is not a new review. Check whether the finalized report fully processed the Pass1 scope and the supplied evidence/probe context. Do not request tools, perform additional exploration, or add speculative findings.\n\n")

	appendReviewRunnerPromptStrictReviewerStance(&b, reviewRunnerPromptPostProbeInsufficientEvidenceGuidance)
	appendReviewRunnerPromptTextSection(&b, "Saturation Check JSON Contract", reviewSaturationCheckPromptContract())
	appendReviewRunnerPromptTextSection(&b, "Custom Instructions", req.CustomInstructions)
	appendReviewRunnerPromptMarkdownSection(&b, "Evidence Markdown", evidenceMarkdown)
	appendReviewRunnerPromptJSONSection(&b, "Decoded Probe Plan", plan)
	appendReviewRunnerPromptJSONSection(&b, "Probe Summaries For Report Schema", redactReviewProbeSummariesForPrompt(probeSummaries, redactor))
	appendReviewRunnerPromptJSONSection(&b, "Probe Result Context", buildReviewProbeResultPromptContexts(results, redactor))
	appendReviewRunnerPromptRedactedJSONSection(&b, "Finalized Review Report", finalizedReport, redactor)
	if finalizedReport.ComputedSummary != nil {
		appendReviewRunnerPromptJSONSection(&b, "Computed Summary", finalizedReport.ComputedSummary)
	}
	return b.String()
}

func buildReviewSaturationCheckRepairPrompt(req ReviewRequest, evidenceMarkdown string, plan ReviewProbePlan, probeSummaries []ReviewProbeSummary, results []ReviewProbeResult, redactor reviewRunnerPromptRedactor, finalizedReport ReviewReport, invalidOutput string, decodeOrValidationErr error) string {
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
	appendReviewRunnerPromptTextSection(&b, "Custom Instructions", req.CustomInstructions)
	appendReviewRunnerPromptMarkdownSection(&b, "Evidence Markdown", evidenceMarkdown)
	appendReviewRunnerPromptJSONSection(&b, "Decoded Probe Plan", plan)
	appendReviewRunnerPromptJSONSection(&b, "Probe Summaries For Report Schema", redactReviewProbeSummariesForPrompt(probeSummaries, redactor))
	appendReviewRunnerPromptJSONSection(&b, "Probe Result Context", buildReviewProbeResultPromptContexts(results, redactor))
	appendReviewRunnerPromptRedactedJSONSection(&b, "Finalized Review Report", finalizedReport, redactor)
	if finalizedReport.ComputedSummary != nil {
		appendReviewRunnerPromptJSONSection(&b, "Computed Summary", finalizedReport.ComputedSummary)
	}
	appendReviewRunnerPromptTextSection(&b, "Invalid Model Output", redactor.redactText(invalidOutput))
	appendReviewRunnerPromptTextSection(&b, "Decode Or Validation Error", redactor.redactText(reviewRunnerPromptErrorText(decodeOrValidationErr)))
	return b.String()
}

func buildReviewReportRevisionPrompt(req ReviewRequest, evidenceMarkdown string, plan ReviewProbePlan, probeSummaries []ReviewProbeSummary, results []ReviewProbeResult, redactor reviewRunnerPromptRedactor, finalizedReport ReviewReport, saturationCheck ReviewSaturationCheck) string {
	var b strings.Builder
	b.WriteString("# Review Pass 2: Report Revision\n\n")
	b.WriteString("Return exactly one JSON object for schema ")
	b.WriteString(ReviewReportSchemaVersionV2)
	b.WriteString(". Do not include markdown or explanatory text outside the JSON.\n\n")
	b.WriteString("Revise the supplied finalized report only to address the saturation check. Use only the supplied evidence, decoded probe plan, probe summaries, and probe result context. Do not perform a new review, do not request tools, and do not add speculative findings. The original finalized report may include runner-computed computed_summary as context; do not output top-level computed_summary in the revised report.\n\n")

	appendReviewRunnerPromptStrictReviewerStance(&b, reviewRunnerPromptPostProbeInsufficientEvidenceGuidance)
	appendReviewRunnerPromptTextSection(&b, "Review Report JSON Contract", reviewReportPromptContract())
	appendReviewRunnerPromptTextSection(&b, "Saturation Revision Guardrails", reviewReportRevisionSaturationGuardrails())
	appendReviewReportRevisionPromptContext(&b, req, evidenceMarkdown, plan, probeSummaries, results, redactor, finalizedReport, saturationCheck)
	return b.String()
}

func buildReviewReportRevisionRepairPrompt(req ReviewRequest, evidenceMarkdown string, plan ReviewProbePlan, probeSummaries []ReviewProbeSummary, results []ReviewProbeResult, redactor reviewRunnerPromptRedactor, finalizedReport ReviewReport, saturationCheck ReviewSaturationCheck, invalidRevisionOutput string, decodeOrValidationErr error) string {
	var b strings.Builder
	b.WriteString("# Review Pass 2: Report Revision JSON Repair\n\n")
	b.WriteString("The previous report revision response failed strict JSON decode or validation. Return corrected ")
	b.WriteString(ReviewReportSchemaVersionV2)
	b.WriteString(" JSON only. Do not add markdown fences. Do not change schema_version from the contract value. Do not request or rely on tools. Preserve trusted probe summary IDs; do not invent probe IDs.\n\n")
	b.WriteString("Use the same evidence, decoded probe plan, probe summaries, probe result context, original finalized report, and saturation check. Do not perform a new review. Only repair the revision so it satisfies the report contract and saturation check. Do not output computed_summary.\n\n")

	appendReviewRunnerPromptStrictReviewerStance(&b, reviewRunnerPromptPostProbeInsufficientEvidenceGuidance)
	appendReviewRunnerPromptTextSection(&b, "Review Report JSON Contract", reviewReportPromptContract())
	appendReviewRunnerPromptTextSection(&b, "Saturation Revision Guardrails", reviewReportRevisionSaturationGuardrails())
	appendReviewReportRevisionPromptContext(&b, req, evidenceMarkdown, plan, probeSummaries, results, redactor, finalizedReport, saturationCheck)
	appendReviewRunnerPromptTextSection(&b, "Invalid Model Output", redactor.redactText(invalidRevisionOutput))
	appendReviewRunnerPromptTextSection(&b, "Decode Or Validation Error", redactor.redactText(reviewRunnerPromptErrorText(decodeOrValidationErr)))
	return b.String()
}

func reviewReportRevisionSaturationGuardrails() string {
	return `Revise only the issues identified by the supplied saturation_check. Do not broaden the review, do not add unrelated findings, and do not rewrite already-correct report sections.
Preserve review_report.v2 schema shape exactly. Do not output top-level "computed_summary".
Preserve the trusted probe_summaries entries supplied for the report schema with the same count, same order, and same probe_id values.
Do not convert failed, blocked, timed-out, or mutated_worktree probe outcomes into clean, checked, dismissed, or verified coverage. Represent them as a finding, residual_risk, unverified scope, or blocked status when the evidence requires it.
Do not use shallow or empty related context/search evidence as proof of no impact. If truncation or evidence gaps prevent verification, reflect that as residual, unverified, or blocked within review_report.v2.`
}

func appendReviewReportRevisionPromptContext(b *strings.Builder, req ReviewRequest, evidenceMarkdown string, plan ReviewProbePlan, probeSummaries []ReviewProbeSummary, results []ReviewProbeResult, redactor reviewRunnerPromptRedactor, finalizedReport ReviewReport, saturationCheck ReviewSaturationCheck) {
	appendReviewRunnerPromptTextSection(b, "Custom Instructions", req.CustomInstructions)
	appendReviewRunnerPromptMarkdownSection(b, "Evidence Markdown", evidenceMarkdown)
	appendReviewRunnerPromptJSONSection(b, "Decoded Probe Plan", plan)
	appendReviewRunnerPromptJSONSection(b, "Probe Summaries For Report Schema", redactReviewProbeSummariesForPrompt(probeSummaries, redactor))
	appendReviewRunnerPromptJSONSection(b, "Probe Result Context", buildReviewProbeResultPromptContexts(results, redactor))
	appendReviewRunnerPromptRedactedJSONSection(b, "Original Finalized Review Report", finalizedReport, redactor)
	appendReviewRunnerPromptJSONSection(b, "Saturation Check", saturationCheck)
}

func appendReviewRunnerPromptRedactedJSONSection(b *strings.Builder, title string, value any, redactor reviewRunnerPromptRedactor) {
	appendReviewRunnerPromptSection(b, title)
	data, err := marshalReviewJSONIndent(value)
	if err != nil {
		appendReviewRunnerPromptFence(b, "json", "null")
		return
	}
	appendReviewRunnerPromptFence(b, "json", redactor.redactText(string(data)))
}
