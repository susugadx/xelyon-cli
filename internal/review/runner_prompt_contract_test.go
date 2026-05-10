package review

import (
	"strings"
	"testing"
)

func TestBuildReviewProbePlanPromptIncludesStrictSchemaContract(t *testing.T) {
	prompt := buildReviewProbePlanPrompt(NewCurrentChangesRequest(""), "diff evidence")

	wants := []string{
		"## Probe Plan JSON Contract",
		"The decoder rejects unknown fields",
		`"schema_version": "review_probe_plan.v1"`,
		`"target_kind": "current_changes"`,
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

func TestBuildReviewReportPromptIncludesStrictSchemaContract(t *testing.T) {
	prompt := buildReviewReportPrompt(
		NewCurrentChangesRequest(""),
		"diff evidence",
		ReviewProbePlan{
			SchemaVersion: ReviewProbePlanSchemaVersionV1,
			TargetKind:    TargetCurrentChanges,
			NoProbeReason: "not needed",
			Probes:        []ReviewPlannedProbe{},
		},
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
