package review

import (
	"errors"
	"strings"
	"testing"
)

func TestBuildReviewSaturationCheckPromptIncludesStrictSchemaContract(t *testing.T) {
	report := withComputedSummaryForRunnerTest(newRunnerCleanReportForTest(nil), nil)
	prompt := buildReviewSaturationCheckPrompt(
		NewCurrentChangesRequest("focus saturation"),
		"diff evidence",
		newNoProbeReviewProbePlanForTest(),
		nil,
		nil,
		reviewRunnerPromptRedactor{},
		report,
	)

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
		`may contain only IDs from Decoded Probe Plan "impact_surfaces[].id"`,
		`may contain only IDs from Decoded Probe Plan "candidate_risks[].id"`,
		"Do not use it for new exploration or speculation",
		`Status "saturated" requires "missing_surface_ids", "missing_risk_ids", "additional_finding_candidates", and "revision_instructions" to be empty`,
		`Status "needs_revision" requires non-empty "revision_instructions"`,
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
	sample := extractReviewSaturationPromptTopLevelSampleForTest(t, prompt)
	if strings.Contains(sample, `"probe_id": "probe-1"`) {
		t.Fatalf("saturation prompt top-level sample contains dangling probe reference:\n%s", sample)
	}
	if _, err := DecodeReviewSaturationCheckJSON([]byte(sample), newNoProbeReviewProbePlanForTest(), newPlanAwareCleanReportForValidationTest()); err != nil {
		t.Fatalf("saturation prompt top-level sample does not validate: %v\n%s", err, sample)
	}
}

func TestBuildReviewSaturationCheckRepairPromptIncludesRepairContract(t *testing.T) {
	report := withComputedSummaryForRunnerTest(newRunnerCleanReportForTest(nil), nil)
	prompt := buildReviewSaturationCheckRepairPrompt(
		NewCurrentChangesRequest("focus saturation repair"),
		"diff evidence",
		newNoProbeReviewProbePlanForTest(),
		nil,
		nil,
		reviewRunnerPromptRedactor{},
		report,
		`{"schema_version":"wrong"}`,
		errors.New("status must be known"),
	)

	wants := []string{
		"Review Final Report Saturation Check JSON Repair",
		"Return corrected JSON only.",
		"Do not add markdown fences.",
		"Do not change schema_version from the contract value.",
		"Do not request or rely on tools.",
		"Keep the same finalized report and Pass1 scope.",
		"## Saturation Check JSON Contract",
		`"schema_version": "review_saturation_check.v1"`,
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
}

func TestBuildReviewReportRevisionPromptIncludesSaturationContract(t *testing.T) {
	report := withComputedSummaryForRunnerTest(newRunnerCleanReportForTest(nil), nil)
	check := ReviewSaturationCheck{
		SchemaVersion:        ReviewSaturationCheckSchemaVersionV1,
		Status:               ReviewSaturationStatusNeedsRevision,
		CheckedSummary:       "risk-1 was not reflected.",
		MissingRiskIDs:       []string{"risk-1"},
		RevisionInstructions: "Classify risk-1 in scope_coverage.",
	}
	prompt := buildReviewReportRevisionPrompt(
		NewCurrentChangesRequest("focus revision"),
		"diff evidence",
		newNoProbeReviewProbePlanForTest(),
		nil,
		nil,
		reviewRunnerPromptRedactor{},
		report,
		check,
	)

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
