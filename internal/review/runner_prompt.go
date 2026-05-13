package review

import "strings"

const (
	reviewRunnerJSONRepairOutputInstructions = "Return corrected JSON only. Do not add markdown fences. Do not change schema_version from the contract value. Do not request or rely on tools."
	reviewRunnerJSONRepairScopeInstruction   = "Repair only the JSON shape and values needed to satisfy the contract."
)

func buildReviewProbePlanPrompt(req ReviewRequest, evidenceMarkdown string) string {
	var b strings.Builder
	b.WriteString("# Review Pass 1: Probe Plan\n\n")
	b.WriteString("Return exactly one JSON object for schema ")
	b.WriteString(ReviewProbePlanSchemaVersionV2)
	b.WriteString(". Do not include markdown or explanatory text outside the JSON.\n\n")
	b.WriteString("First enumerate material impact surfaces from the provided evidence, then candidate risks, then only bounded probes that confirm or falsify those risks or unverified material surfaces. Use plan order as execution order. If no candidate risks remain, provide no_candidate_risk_reason for every impact surface ID. If no probe is useful, return an empty probes array and a no_probe_reason that names the checked surface and risk IDs.\n\n")

	appendReviewRunnerPromptTextSection(&b, "Probe Plan JSON Contract", reviewProbePlanPromptContract())
	appendReviewRunnerPromptTextSection(&b, "Custom Instructions", req.CustomInstructions)
	appendReviewRunnerPromptMarkdownSection(&b, "Evidence Markdown", evidenceMarkdown)
	return b.String()
}

func buildReviewProbePlanRepairPrompt(req ReviewRequest, evidenceMarkdown, invalidOutput string, decodeOrValidationErr error) string {
	var b strings.Builder
	b.WriteString("# Review Pass 1: Probe Plan JSON Repair\n\n")
	b.WriteString("The previous probe plan response failed strict JSON decode or validation. ")
	b.WriteString(reviewRunnerJSONRepairOutputInstructions)
	b.WriteString("\n\n")
	b.WriteString("Keep the same target and task context. ")
	b.WriteString(reviewRunnerJSONRepairScopeInstruction)
	b.WriteString("\n\n")

	appendReviewRunnerPromptTextSection(&b, "Probe Plan JSON Contract", reviewProbePlanPromptContract())
	appendReviewRunnerPromptTextSection(&b, "Custom Instructions", req.CustomInstructions)
	appendReviewRunnerPromptMarkdownSection(&b, "Evidence Markdown", evidenceMarkdown)
	appendReviewRunnerPromptTextSection(&b, "Invalid Model Output", invalidOutput)
	appendReviewRunnerPromptTextSection(&b, "Decode Or Validation Error", reviewRunnerPromptErrorText(decodeOrValidationErr))
	return b.String()
}

func buildReviewReportPrompt(req ReviewRequest, evidenceMarkdown string, plan ReviewProbePlan, probeSummaries []ReviewProbeSummary, results []ReviewProbeResult, redactor reviewRunnerPromptRedactor) string {
	var b strings.Builder
	b.WriteString("# Review Pass 2: Report\n\n")
	b.WriteString("Return exactly one JSON object for schema ")
	b.WriteString(ReviewReportSchemaVersionV2)
	b.WriteString(". Do not include markdown or explanatory text outside the JSON.\n\n")
	b.WriteString("Use the evidence, decoded probe plan, probe summaries, and probe result context to produce the final report. Preserve probe_summaries using the supplied summaries. The final report must classify every decoded probe plan impact surface and candidate risk in scope_coverage exactly once. Reference only repo-relative paths or displayed evidence paths.\n\n")

	appendReviewRunnerPromptTextSection(&b, "Review Report JSON Contract", reviewReportPromptContract())
	appendReviewRunnerPromptTextSection(&b, "Custom Instructions", req.CustomInstructions)
	appendReviewRunnerPromptMarkdownSection(&b, "Evidence Markdown", evidenceMarkdown)
	appendReviewRunnerPromptJSONSection(&b, "Decoded Probe Plan", plan)
	appendReviewRunnerPromptJSONSection(&b, "Probe Summaries For Report Schema", redactReviewProbeSummariesForPrompt(probeSummaries, redactor))
	appendReviewRunnerPromptJSONSection(&b, "Probe Result Context", buildReviewProbeResultPromptContexts(results, redactor))
	return b.String()
}

func buildReviewReportRepairPrompt(req ReviewRequest, evidenceMarkdown string, plan ReviewProbePlan, probeSummaries []ReviewProbeSummary, results []ReviewProbeResult, redactor reviewRunnerPromptRedactor, invalidOutput string, decodeOrValidationErr error) string {
	var b strings.Builder
	b.WriteString("# Review Pass 2: Report JSON Repair\n\n")
	b.WriteString("The previous report response failed strict JSON decode or validation. ")
	b.WriteString(reviewRunnerJSONRepairOutputInstructions)
	b.WriteString(" Preserve trusted probe summary IDs; do not invent probe IDs.\n\n")
	b.WriteString("Use the same evidence, decoded probe plan, probe summaries, and probe result context. ")
	b.WriteString(reviewRunnerJSONRepairScopeInstruction)
	b.WriteString("\n\n")

	appendReviewRunnerPromptTextSection(&b, "Review Report JSON Contract", reviewReportPromptContract())
	appendReviewRunnerPromptTextSection(&b, "Custom Instructions", req.CustomInstructions)
	appendReviewRunnerPromptMarkdownSection(&b, "Evidence Markdown", evidenceMarkdown)
	appendReviewRunnerPromptJSONSection(&b, "Decoded Probe Plan", plan)
	appendReviewRunnerPromptJSONSection(&b, "Probe Summaries For Report Schema", redactReviewProbeSummariesForPrompt(probeSummaries, redactor))
	appendReviewRunnerPromptJSONSection(&b, "Probe Result Context", buildReviewProbeResultPromptContexts(results, redactor))
	appendReviewRunnerPromptTextSection(&b, "Invalid Model Output", redactor.redactText(invalidOutput))
	appendReviewRunnerPromptTextSection(&b, "Decode Or Validation Error", redactor.redactText(reviewRunnerPromptErrorText(decodeOrValidationErr)))
	return b.String()
}

func reviewRunnerPromptErrorText(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func appendReviewRunnerPromptTextSection(b *strings.Builder, title, content string) {
	appendReviewRunnerPromptSection(b, title)
	appendReviewRunnerPromptFence(b, "text", content)
}

func appendReviewRunnerPromptMarkdownSection(b *strings.Builder, title, content string) {
	appendReviewRunnerPromptSection(b, title)
	appendReviewRunnerPromptFence(b, "markdown", content)
}

func appendReviewRunnerPromptJSONSection(b *strings.Builder, title string, value any) {
	appendReviewRunnerPromptSection(b, title)
	data, err := marshalReviewJSONIndent(value)
	if err != nil {
		appendReviewRunnerPromptFence(b, "json", "null")
		return
	}
	appendReviewRunnerPromptFence(b, "json", string(data))
}

func appendReviewRunnerPromptSection(b *strings.Builder, title string) {
	b.WriteString("## ")
	b.WriteString(title)
	b.WriteString("\n")
}

func appendReviewRunnerPromptFence(b *strings.Builder, language, content string) {
	fence := reviewRunnerPromptFence(content)
	b.WriteString(fence)
	b.WriteString(language)
	b.WriteByte('\n')
	b.WriteString(content)
	if !strings.HasSuffix(content, "\n") {
		b.WriteByte('\n')
	}
	b.WriteString(fence)
	b.WriteString("\n\n")
}

func reviewRunnerPromptFence(content string) string {
	return strings.Repeat("`", max(3, longestReviewRunnerPromptBacktickRun(content)+1))
}

func longestReviewRunnerPromptBacktickRun(content string) int {
	longest := 0
	current := 0
	for _, r := range content {
		if r == '`' {
			current++
			longest = max(longest, current)
			continue
		}
		current = 0
	}
	return longest
}
