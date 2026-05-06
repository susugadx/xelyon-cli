package review

import "strings"

func buildReviewProbePlanPrompt(req ReviewRequest, evidenceMarkdown string) string {
	var b strings.Builder
	b.WriteString("# Review Pass 1: Probe Plan\n\n")
	b.WriteString("Return exactly one JSON object for schema ")
	b.WriteString(ReviewProbePlanSchemaVersionV1)
	b.WriteString(". Do not include markdown or explanatory text outside the JSON.\n\n")
	b.WriteString("Plan only bounded verification probes that materially reduce uncertainty for the provided evidence. Use plan order as execution order. If no probe is useful, return an empty probes array and a non-empty no_probe_reason.\n\n")
	b.WriteString("Required top-level fields: schema_version, target_kind, probes. target_kind must be current_changes.\n\n")

	appendReviewRunnerPromptTextSection(&b, "Custom Instructions", req.CustomInstructions)
	appendReviewRunnerPromptMarkdownSection(&b, "Evidence Markdown", evidenceMarkdown)
	return b.String()
}

func buildReviewReportPrompt(req ReviewRequest, evidenceMarkdown string, plan ReviewProbePlan, probeSummaries []ReviewProbeSummary, results []ReviewProbeResult, redactor reviewRunnerPromptRedactor) string {
	var b strings.Builder
	b.WriteString("# Review Pass 2: Report\n\n")
	b.WriteString("Return exactly one JSON object for schema ")
	b.WriteString(ReviewReportSchemaVersionV1)
	b.WriteString(". Do not include markdown or explanatory text outside the JSON.\n\n")
	b.WriteString("Use the evidence, decoded probe plan, probe summaries, and probe result context to produce the final report. Preserve probe_summaries using the supplied summaries. Reference only repo-relative paths or displayed evidence paths.\n\n")
	b.WriteString("Required top-level fields: schema_version, target_kind, generated_at, overall_verification_status, verdict.\n\n")

	appendReviewRunnerPromptTextSection(&b, "Custom Instructions", req.CustomInstructions)
	appendReviewRunnerPromptMarkdownSection(&b, "Evidence Markdown", evidenceMarkdown)
	appendReviewRunnerPromptJSONSection(&b, "Decoded Probe Plan", plan)
	appendReviewRunnerPromptJSONSection(&b, "Probe Summaries For Report Schema", redactReviewProbeSummariesForPrompt(probeSummaries, redactor))
	appendReviewRunnerPromptJSONSection(&b, "Probe Result Context", buildReviewProbeResultPromptContexts(results, redactor))
	return b.String()
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
