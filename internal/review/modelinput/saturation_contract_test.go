package modelinput

import (
	"errors"
	"strings"
	"testing"

	reviewreport "github.com/susugadx/xelyon-cli/internal/review/report"
)

func TestBuildReviewSaturationCheckPromptIncludesStrictSchemaContract(t *testing.T) {
	report := withComputedSummaryForRunnerTest(newRunnerCleanReportForTest(nil), nil)
	prompt := BuildSaturationCheckPrompt(SaturationCheckPromptInput{
		CustomInstructions: "focus saturation",
		EvidenceMarkdown:   "diff evidence",
		Plan:               newNoProbeReviewProbePlanForTest(),
		FinalizedReport:    report,
	})

	wants := []string{
		"Review Final Report Saturation Check",
		"## Saturation Check JSON Contract",
		"The decoder rejects unknown fields",
		`"schema_version": "review_saturation_check.v1"`,
		`"status": "needs_revision"`,
		`"missing_surface_ids": ["surface-1"]`,
		`"missing_risk_ids": ["risk-1"]`,
		`"additional_finding_candidates"`,
		`"kind": "file"`,
		`"revision_instructions": "revise the report to classify surface-1 and risk-1 or include the evidence-backed candidate"`,
		`"status" must be one of "saturated", "needs_revision", "blocked"`,
		"This is a saturation check, not a new review",
		"Do not request tools, perform additional exploration",
		"Do not treat \"scope_coverage\" as saturated just because it repeats every Pass1 ID",
		"Failed, blocked, timed-out, or \"mutated_worktree\" probe outcomes must be reflected",
		"Shallow, empty, or absent related context/search evidence is not proof of no impact",
		"Fetched external_doc snippets are citation-capable evidence, but are not automatically official documentation or confirmed external specs",
		`Source credibility values are "official_candidate", "third_party", and "unknown"`,
		`only "official_candidate" may be treated as an official-source candidate`,
		`Do not infer official or authoritative status from search query wording, source title, URL label, snippet wording, or "official documentation" text alone`,
		`source_credibility is "unknown" or "third_party"`,
		"source_credibility and the cited snippet content must both support that claim",
		`external_support.level/official_confirmation must allow confirmed official status`,
		`Treat external_support.level as the maximum external evidence confidence`,
		`Levels "none", "weak", and "partial" are not confirmed external spec coverage`,
		`external_support.official_confirmation=false means official confirmation is absent`,
		"If source credibility is unclear, fetch failed, evidence is truncated, or review web search evidence is inconclusive",
		"do not bias toward saturated/clean/verified",
		"Trace review pressure signals from Evidence Markdown and the Decoded Probe Plan",
		"Missing verification alone is a coverage gap, not an additional finding candidate",
		"supports an additional finding candidate within the report scope",
		`may contain only IDs from Decoded Probe Plan "impact_surfaces[].id"`,
		`may contain only IDs from Decoded Probe Plan "candidate_risks[].id"`,
		"Do not use it for new exploration or speculation",
		`Status "saturated" requires "missing_surface_ids", "missing_risk_ids", "additional_finding_candidates", and "revision_instructions" to be empty`,
		`Status "needs_revision" requires non-empty "revision_instructions"`,
		"Use it when the Finalized Review Report escapes, omits, or downplays pressure signals",
		"Output only this saturation check JSON object",
		`do not include top-level "computed_summary"`,
		"## Finalized Review Report",
		"## Computed Summary",
		"focus saturation",
		"diff evidence",
	}
	for _, want := range wants {
		if !strings.Contains(prompt, want) {
			t.Fatalf("saturation check prompt missing %q:\n%s", want, prompt)
		}
	}
	assertReviewRunnerPromptContainsExternalSupportGuardrails(t, prompt)
	sample := extractReviewSaturationPromptTopLevelSampleForTest(t, prompt)
	if strings.Contains(sample, `"probe_id": "probe-1"`) {
		t.Fatalf("saturation prompt top-level sample contains dangling probe reference:\n%s", sample)
	}
	if _, err := reviewreport.DecodeReviewSaturationCheckJSON([]byte(sample), newNoProbePlanScopeForTest(), newPlanAwareCleanReportForValidationTest()); err != nil {
		t.Fatalf("saturation prompt top-level sample does not validate: %v\n%s", err, sample)
	}
}

func TestBuildReviewSaturationCheckRepairPromptIncludesRepairContract(t *testing.T) {
	report := withComputedSummaryForRunnerTest(newRunnerCleanReportForTest(nil), nil)
	prompt := BuildSaturationCheckRepairPrompt(SaturationCheckRepairPromptInput{
		CustomInstructions:    "focus saturation repair",
		EvidenceMarkdown:      "diff evidence",
		Plan:                  newNoProbeReviewProbePlanForTest(),
		FinalizedReport:       report,
		InvalidOutput:         `{"schema_version":"wrong"}`,
		DecodeOrValidationErr: errors.New("status must be known"),
	})

	wants := []string{
		"Review Final Report Saturation Check JSON Repair",
		"Return corrected JSON only.",
		"Do not add markdown fences.",
		"Do not change schema_version from the contract value.",
		"Do not request or rely on tools.",
		"Keep the same finalized report and Pass1 scope.",
		"## Saturation Check JSON Contract",
		`"schema_version": "review_saturation_check.v1"`,
		"Failed, blocked, timed-out, or \"mutated_worktree\" probe outcomes must be reflected",
		"Fetched external_doc snippets are citation-capable evidence, but are not automatically official documentation or confirmed external specs",
		`Source credibility values are "official_candidate", "third_party", and "unknown"`,
		`only "official_candidate" may be treated as an official-source candidate`,
		`source_credibility is "unknown" or "third_party"`,
		`Treat external_support.level as the maximum external evidence confidence`,
		"Use it when the Finalized Review Report escapes, omits, or downplays pressure signals",
		"Missing verification alone is a coverage gap, not an additional finding candidate",
		"focus saturation repair",
		"diff evidence",
		"## Finalized Review Report",
		"## Invalid Model Output",
		`{"schema_version":"wrong"}`,
		"## Decode Or Validation Error",
		"status must be known",
	}
	for _, want := range wants {
		if !strings.Contains(prompt, want) {
			t.Fatalf("saturation check repair prompt missing %q:\n%s", want, prompt)
		}
	}
	assertReviewRunnerPromptContainsExternalSupportGuardrails(t, prompt)
}

func TestBuildReviewSaturationAndRevisionPromptsIncludeStrictReviewerStance(t *testing.T) {
	report := withComputedSummaryForRunnerTest(newRunnerCleanReportForTest(nil), nil)
	plan := newNoProbeReviewProbePlanForTest()
	check := needsRevisionCheckForPromptContractTest()
	prompts := map[string]string{
		"saturation check": BuildSaturationCheckPrompt(SaturationCheckPromptInput{
			CustomInstructions: "focus saturation",
			EvidenceMarkdown:   "diff evidence",
			Plan:               plan,
			FinalizedReport:    report,
		}),
		"saturation check repair": BuildSaturationCheckRepairPrompt(SaturationCheckRepairPromptInput{
			CustomInstructions:    "focus saturation repair",
			EvidenceMarkdown:      "diff evidence",
			Plan:                  plan,
			FinalizedReport:       report,
			InvalidOutput:         `{"schema_version":"wrong"}`,
			DecodeOrValidationErr: errors.New("status must be known"),
		}),
		"report revision": BuildReportRevisionPrompt(ReportRevisionPromptInput{
			CustomInstructions: "focus revision",
			EvidenceMarkdown:   "diff evidence",
			Plan:               plan,
			FinalizedReport:    report,
			SaturationCheck:    check,
		}),
		"report revision repair": BuildReportRevisionRepairPrompt(ReportRevisionRepairPromptInput{
			CustomInstructions:    "focus revision repair",
			EvidenceMarkdown:      "diff evidence",
			Plan:                  plan,
			FinalizedReport:       report,
			SaturationCheck:       check,
			InvalidRevisionOutput: `{"schema_version":"review_report.v2","computed_summary":{}}`,
			DecodeOrValidationErr: errors.New("target_kind must be current_changes"),
		}),
	}

	for name, prompt := range prompts {
		t.Run(name, func(t *testing.T) {
			assertReviewRunnerPromptContainsStrictReviewerStance(t, prompt)
			assertReviewRunnerPromptContainsPostProbeInsufficientEvidenceGuidance(t, prompt)
		})
	}
}

func TestBuildReviewReportRevisionPromptIncludesSaturationContract(t *testing.T) {
	report := withComputedSummaryForRunnerTest(newRunnerCleanReportForTest(nil), nil)
	prompt := BuildReportRevisionPrompt(ReportRevisionPromptInput{
		CustomInstructions: "focus revision",
		EvidenceMarkdown:   "diff evidence",
		Plan:               newNoProbeReviewProbePlanForTest(),
		FinalizedReport:    report,
		SaturationCheck:    needsRevisionCheckForPromptContractTest(),
	})

	wants := []string{
		"Review Pass 2: Report Revision",
		`Return exactly one JSON object for schema review_report.v2`,
		"Revise the supplied finalized report only to address the saturation check",
		"Do not perform a new review",
		"do not request tools",
		"do not add speculative findings",
		"do not output top-level computed_summary",
		"## Review Report JSON Contract",
		`Do not output top-level "computed_summary"; runner computes it after validation`,
		"## Saturation Revision Guardrails",
		"Revise only the issues identified by the supplied saturation_check",
		"Preserve review_report.v2 schema shape exactly",
		"Preserve the trusted probe_summaries entries supplied for the report schema with the same count, same order, and same probe_id values",
		"Do not convert failed, blocked, timed-out, or mutated_worktree probe outcomes into clean, checked, dismissed, or verified coverage",
		"Deterministic coverage audit feedback in saturation_check is not a mandate to create a finding",
		"Missing verification alone is a coverage gap, not an additional finding candidate",
		"Raw web search results are discovery-only",
		"Fetched external_doc snippets are citation-capable evidence, but are not automatically official documentation or confirmed external specs",
		`Source credibility values are "official_candidate", "third_party", and "unknown"`,
		`only "official_candidate" may be treated as an official-source candidate`,
		`Do not infer official or authoritative status from search query wording, source title, URL label, snippet wording, or "official documentation" text alone`,
		`source_credibility is "unknown" or "third_party"`,
		"source_credibility and the cited snippet content must both support that claim",
		`external_support.level/official_confirmation must allow confirmed official status`,
		`Treat external_support.level as the maximum external evidence confidence`,
		`Levels "none", "weak", and "partial" are not confirmed external spec coverage`,
		`external_support.official_confirmation=false means official confirmation is absent`,
		"## Original Finalized Review Report",
		"## Saturation Check",
		`"status": "needs_revision"`,
		`"missing_risk_ids": [
    "risk-1"
  ]`,
		"focus revision",
		"diff evidence",
	}
	for _, want := range wants {
		if !strings.Contains(prompt, want) {
			t.Fatalf("report revision prompt missing %q:\n%s", want, prompt)
		}
	}
	assertReviewRunnerPromptContainsExternalSupportGuardrails(t, prompt)
}

func TestBuildReviewReportRevisionRepairPromptIncludesRepairContract(t *testing.T) {
	report := withComputedSummaryForRunnerTest(newRunnerCleanReportForTest(nil), nil)
	prompt := BuildReportRevisionRepairPrompt(ReportRevisionRepairPromptInput{
		CustomInstructions:    "focus revision repair",
		EvidenceMarkdown:      "diff evidence",
		Plan:                  newNoProbeReviewProbePlanForTest(),
		FinalizedReport:       report,
		SaturationCheck:       needsRevisionCheckForPromptContractTest(),
		InvalidRevisionOutput: `{"schema_version":"review_report.v2","computed_summary":{}}`,
		DecodeOrValidationErr: errors.New("target_kind must be current_changes"),
	})

	wants := []string{
		"Review Pass 2: Report Revision JSON Repair",
		"Return corrected review_report.v2 JSON only.",
		"Do not add markdown fences.",
		"Do not change schema_version from the contract value.",
		"Do not request or rely on tools.",
		"Preserve trusted probe summary IDs; do not invent probe IDs.",
		"Do not output computed_summary.",
		"Do not perform a new review.",
		"Only repair the revision so it satisfies the report contract and saturation check.",
		"## Review Report JSON Contract",
		`Do not output top-level "computed_summary"; runner computes it after validation`,
		"## Saturation Revision Guardrails",
		"Revise only the issues identified by the supplied saturation_check",
		"Preserve review_report.v2 schema shape exactly",
		"Preserve the trusted probe_summaries entries supplied for the report schema with the same count, same order, and same probe_id values",
		"Do not convert failed, blocked, timed-out, or mutated_worktree probe outcomes into clean, checked, dismissed, or verified coverage",
		"Missing verification alone is a coverage gap, not an additional finding candidate",
		"## Original Finalized Review Report",
		"## Saturation Check",
		`"status": "needs_revision"`,
		`"missing_risk_ids": [
    "risk-1"
  ]`,
		"## Invalid Model Output",
		`{"schema_version":"review_report.v2","computed_summary":{}}`,
		"## Decode Or Validation Error",
		"target_kind must be current_changes",
		"focus revision repair",
		"diff evidence",
	}
	for _, want := range wants {
		if !strings.Contains(prompt, want) {
			t.Fatalf("report revision repair prompt missing %q:\n%s", want, prompt)
		}
	}
	assertReviewRunnerPromptContainsExternalSupportGuardrails(t, prompt)
}

func needsRevisionCheckForPromptContractTest() reviewreport.ReviewSaturationCheck {
	return reviewreport.ReviewSaturationCheck{
		SchemaVersion:        reviewreport.ReviewSaturationCheckSchemaVersionV1,
		Status:               reviewreport.ReviewSaturationStatusNeedsRevision,
		CheckedSummary:       "risk-1 was not reflected.",
		MissingRiskIDs:       []string{"risk-1"},
		RevisionInstructions: "Classify risk-1 in scope_coverage.",
	}
}

func extractReviewSaturationPromptTopLevelSampleForTest(t *testing.T, prompt string) string {
	t.Helper()

	start := strings.Index(prompt, "- Top-level object:")
	if start < 0 {
		t.Fatalf("saturation prompt missing top-level object sample:\n%s", prompt)
	}
	relativeObjectStart := strings.Index(prompt[start:], "{")
	if relativeObjectStart < 0 {
		t.Fatalf("saturation prompt missing JSON object sample:\n%s", prompt)
	}
	start += relativeObjectStart
	endMarker := "\n- \"schema_version\" must be"
	end := strings.Index(prompt[start:], endMarker)
	if end < 0 {
		t.Fatalf("saturation prompt missing top-level object sample end marker:\n%s", prompt)
	}
	return prompt[start : start+end]
}
