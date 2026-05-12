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
		`"category": "changed_file"`,
		`"status": "needs_probe"`,
		`"severity": "medium"`,
		`"checked_by_evidence"`,
		`no_probe_reason" must name every checked surface ID and checked risk ID`,
		`Scope evidence refs are pre-probe only`,
		`Do not plan broad speculative test suites`,
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
		`"schema_version": "review_report.v1"`,
		`"target_kind": "current_changes"`,
		`There is no top-level "findings" or "has_findings" field`,
		`"root_cause_groups[].findings"`,
		`"verdict" must be one of "clean", "has_findings", "blocked"`,
		`"overall_verification_status" and group "verification_status" must be one of`,
		`Severity must be one of "critical", "high", "medium", "low", "info"`,
		`"kind" must be one of "probe_command", "probe", "file", "diff", "git_status", "rule_file"`,
		`"file", "diff", and "rule_file" refs require "path"`,
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
		`"schema_version": "review_report.v1"`,
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
}
