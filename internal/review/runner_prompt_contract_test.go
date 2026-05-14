package review

import (
	"errors"
	"strings"
	"testing"
)

func TestBuildReviewProbePlanPromptIncludesStrictSchemaContract(t *testing.T) {
	prompt := buildReviewProbePlanPrompt(NewCurrentChangesRequest(""), "diff evidence")

	wants := []string{
		"## Probe Plan JSON Contract",
		"The decoder rejects unknown fields",
		`"schema_version": "review_probe_plan.v2"`,
		`"target_kind": "current_changes"`,
		`"impact_surfaces"`,
		`"candidate_risks"`,
		`"no_candidate_risk_reason": ""`,
		`"surface_ids": ["surface-1"]`,
		`"risk_ids": ["risk-1"]`,
		`"category": "changed_file"`,
		`"status": "needs_probe"`,
		`"severity": "medium"`,
		`"checked_by_evidence"`,
		`canonical IDs using only ASCII letters, digits, hyphen, or underscore`,
		`Impact surface "summary" and "reason" must be non-empty`,
		`Candidate risk "summary" and "verification_strategy" must be non-empty`,
		`Each probe must include "surface_ids" or "risk_ids"`,
		`Every impact surface with status "needs_probe" or "unverified" must be referenced directly`,
		`Each probe purpose must explain how the referenced surface or risk IDs will be confirmed or falsified`,
		`no_candidate_risk_reason" must be non-empty, must mention every impact surface ID`,
		`Do not use generic "No risk" wording`,
		`no_probe_reason" must name every checked surface ID and checked risk ID`,
		`Scope evidence refs are pre-probe only`,
		`The validator requires each probe to be linked`,
		`"mode": "host_readonly"`,
		`"host_readonly"`,
		`"scratch_only"`,
		`"repo_sandbox"`,
		`"commands"`,
		`"command" is an executable name only`,
		`"files" is for generated files in isolated modes only`,
		`It must be empty when mode is "host_readonly"`,
		`Do not output shell pipelines`,
		`"go"`,
		`"npm"`,
		`"cargo"`,
	}
	for _, want := range wants {
		if !strings.Contains(prompt, want) {
			t.Fatalf("probe plan prompt missing %q:\n%s", want, prompt)
		}
	}
}

func TestBuildReviewProbePlanRepairPromptIncludesRepairContract(t *testing.T) {
	prompt := buildReviewProbePlanRepairPrompt(
		NewCurrentChangesRequest("focus repair"),
		"diff evidence",
		`{"schema_version":"wrong"}`,
		errors.New("schema_version must be review_probe_plan.v2"),
	)

	wants := []string{
		"Review Pass 1: Probe Plan JSON Repair",
		"Return corrected JSON only.",
		"Do not add markdown fences.",
		"Do not change schema_version from the contract value.",
		"Do not request or rely on tools.",
		"## Probe Plan JSON Contract",
		`"schema_version": "review_probe_plan.v2"`,
		`"impact_surfaces"`,
		`"candidate_risks"`,
		`"no_candidate_risk_reason": ""`,
		`"surface_ids": ["surface-1"]`,
		`"risk_ids": ["risk-1"]`,
		`canonical IDs using only ASCII letters, digits, hyphen, or underscore`,
		`Impact surface "summary" and "reason" must be non-empty`,
		`Candidate risk "summary" and "verification_strategy" must be non-empty`,
		`Every impact surface with status "needs_probe" or "unverified" must be referenced directly`,
		`Each probe purpose must explain how the referenced surface or risk IDs will be confirmed or falsified`,
		`no_candidate_risk_reason" must be non-empty, must mention every impact surface ID`,
		`Do not use generic "No risk" wording`,
		"focus repair",
		"diff evidence",
		"## Invalid Model Output",
		`{"schema_version":"wrong"}`,
		"## Decode Or Validation Error",
		"schema_version must be review_probe_plan.v2",
	}
	for _, want := range wants {
		if !strings.Contains(prompt, want) {
			t.Fatalf("probe plan repair prompt missing %q:\n%s", want, prompt)
		}
	}
}

func TestBuildReviewReportPromptIncludesStrictSchemaContract(t *testing.T) {
	prompt := buildReviewReportPrompt(
		NewCurrentChangesRequest(""),
		"diff evidence",
		newNoProbeReviewProbePlanForTest(),
		nil,
		nil,
		reviewRunnerPromptRedactor{},
	)

	wants := []string{
		"## Review Report JSON Contract",
		"The decoder rejects unknown fields",
		`"schema_version": "review_report.v2"`,
		`"target_kind": "current_changes"`,
		`"scope_coverage"`,
		`"reviewed_impact_surfaces"`,
		`"reviewed_candidate_risks"`,
		`"new_findings_from_report_pass"`,
		`"new_findings_from_report_pass": []`,
		`must classify every ID from the Decoded Probe Plan exactly once`,
		`Candidate risks must be classified as "finding", "dismissed", "residual_risk", or "unverified"`,
		`"finding_ids" in reviewed impact surfaces and reviewed candidate risks must be empty unless that scope entry status is "finding"`,
		`A "clean" verdict is allowed only when every impact surface status is "checked" and every candidate risk status is "dismissed"`,
		`Findings discovered during Pass2 that were not Pass1 candidate risks`,
		`There is no top-level "findings" or "has_findings" field`,
		`"root_cause_groups[].findings"`,
		`"verdict" must be one of "clean", "has_findings", "blocked"`,
		`"overall_verification_status" and group "verification_status" must be one of`,
		`Severity must be one of "critical", "high", "medium", "low", "info"`,
		`finding IDs are required, must be unique`,
		`Every root cause finding ID must be connected from "scope_coverage"`,
		`"kind" must be one of "probe_command", "probe", "file", "diff", "git_status", "rule_file"`,
		`"file", "diff", and "rule_file" refs require "path"`,
		`Do not output top-level "computed_summary"; runner computes it after validation`,
		`Probe summaries must preserve the supplied "Probe Summaries For Report Schema" entries`,
		`Verdict contract`,
		`"has_findings": "overall_verification_status" must be "verified" or "partially_verified"`,
		`Each root cause group must include at least one "findings" item, non-empty "fix_strategy", and at least one "verification_plan" item`,
		`Each finding must include at least one "evidence_refs" item`,
	}
	for _, want := range wants {
		if !strings.Contains(prompt, want) {
			t.Fatalf("report prompt missing %q:\n%s", want, prompt)
		}
	}
	assertReviewReportPromptContainsScopeFindingLinkageContract(t, prompt)
	forbids := []string{
		`"probe_id": "probe-1"`,
		`"finding_ids": ["finding-2"]`,
	}
	for _, forbid := range forbids {
		if strings.Contains(prompt, forbid) {
			t.Fatalf("report prompt contains dangling sample reference %q:\n%s", forbid, prompt)
		}
	}
}

func TestBuildReviewReportPromptDoesNotIncludeComputedSummaryInTopLevelSample(t *testing.T) {
	prompt := buildReviewReportPrompt(
		NewCurrentChangesRequest(""),
		"diff evidence",
		newNoProbeReviewProbePlanForTest(),
		nil,
		nil,
		reviewRunnerPromptRedactor{},
	)

	if !strings.Contains(prompt, `Do not output top-level "computed_summary"; runner computes it after validation`) {
		t.Fatalf("report prompt missing computed_summary prohibition:\n%s", prompt)
	}
	sample := extractReviewReportPromptTopLevelSampleForTest(t, prompt)
	if strings.Contains(sample, `"computed_summary"`) {
		t.Fatalf("report prompt top-level sample contains computed_summary:\n%s", sample)
	}
}

func TestBuildReviewReportRepairPromptIncludesRepairContract(t *testing.T) {
	prompt := buildReviewReportRepairPrompt(
		NewCurrentChangesRequest("focus repair"),
		"diff evidence",
		newNoProbeReviewProbePlanForTest(),
		[]ReviewProbeSummary{
			{
				ProbeID: "probe-1",
				Mode:    ReviewProbeHostReadOnly,
				Status:  ReviewProbePassed,
			},
		},
		nil,
		reviewRunnerPromptRedactor{},
		`{"schema_version":"wrong"}`,
		errors.New("generated_at must be non-zero"),
	)

	wants := []string{
		"Review Pass 2: Report JSON Repair",
		"Return corrected JSON only.",
		"Do not add markdown fences.",
		"Do not change schema_version from the contract value.",
		"Do not request or rely on tools.",
		"Preserve trusted probe summary IDs; do not invent probe IDs.",
		"## Review Report JSON Contract",
		`"schema_version": "review_report.v2"`,
		`"scope_coverage"`,
		`must classify every ID from the Decoded Probe Plan exactly once`,
		"focus repair",
		"diff evidence",
		"## Decoded Probe Plan",
		"## Probe Summaries For Report Schema",
		`"probe_id": "probe-1"`,
		"## Invalid Model Output",
		`{"schema_version":"wrong"}`,
		"## Decode Or Validation Error",
		"generated_at must be non-zero",
	}
	for _, want := range wants {
		if !strings.Contains(prompt, want) {
			t.Fatalf("report repair prompt missing %q:\n%s", want, prompt)
		}
	}
	assertReviewReportPromptContainsScopeFindingLinkageContract(t, prompt)
}

func extractReviewReportPromptTopLevelSampleForTest(t *testing.T, prompt string) string {
	t.Helper()

	start := strings.Index(prompt, "- Top-level object:")
	if start < 0 {
		t.Fatalf("report prompt missing top-level object sample:\n%s", prompt)
	}
	endMarker := "\n- There is no top-level"
	end := strings.Index(prompt[start:], endMarker)
	if end < 0 {
		t.Fatalf("report prompt missing top-level object sample end marker:\n%s", prompt)
	}
	return prompt[start : start+end]
}

func assertReviewReportPromptContainsScopeFindingLinkageContract(t *testing.T, prompt string) {
	t.Helper()

	wants := []string{
		`impact surface status "finding" must include non-empty "finding_ids"`,
		`candidate risk status "finding" must include non-empty "finding_ids"`,
		`each referenced finding must exist under "root_cause_groups" and include evidence_refs`,
	}
	for _, want := range wants {
		if !strings.Contains(prompt, want) {
			t.Fatalf("report prompt missing scope finding linkage contract %q:\n%s", want, prompt)
		}
	}
}
