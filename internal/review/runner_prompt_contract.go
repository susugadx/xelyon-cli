package review

import (
	"fmt"
	"sort"
	"strings"
)

func reviewProbePlanPromptContract() string {
	return fmt.Sprintf(`Strict JSON contract:
- The decoder rejects unknown fields. Use only the fields listed here.
- Top-level object:
  {
    "schema_version": %q,
    "target_kind": %q,
    "summary": "optional short string",
    "impact_surfaces": [
      {
        "id": "surface-1",
        "summary": "material surface summary",
        "category": %q,
        "evidence_summary": "pre-probe evidence summary",
        "evidence_refs": [
          {"kind": %q, "path": "internal/review/probe_plan.go", "line": 1, "summary": "optional evidence summary"}
        ],
        "status": %q,
        "reason": "why this surface is checked, needs a probe, or remains unverified"
      }
    ],
    "candidate_risks": [
      {
        "id": "risk-1",
        "summary": "candidate risk summary",
        "severity": %q,
        "surface_ids": ["surface-1"],
        "evidence_summary": "pre-probe evidence summary",
        "evidence_refs": [
          {"kind": %q, "path": "internal/review/probe_plan.go"}
        ],
        "verification_strategy": "how a bounded probe would confirm or falsify the risk",
        "status": %q
      }
    ],
    "probes": [
      {
        "id": "probe-1",
        "purpose": "non-empty purpose, at most %d bytes",
        "mode": %q,
        "commands": [
          {"command": "go", "args": ["test", "./internal/review"], "work_dir": "."}
        ],
        "files": [
          {"path": "relative/generated_file.go", "content": "file contents"}
        ],
        "timeout_seconds": 60,
        "max_output_bytes": 65536
      }
    ],
    "no_probe_reason": "required non-empty only when probes is empty"
  }
- "target_kind" must be %q. "impact_surfaces" must contain at least one entry. "candidate_risks" may be empty. "probes" must contain at most %d entries.
- Scope analysis order: first enumerate material "impact_surfaces"; then derive "candidate_risks" from those surfaces; then create probes only for evidence-backed risks or unverified material surfaces.
- Consider changed files, callers, tests, related search hits, related tests/context files, CLI, TUI, config, validator, prompt contract, JSON schema, sandbox, timeout, path validation, error handling, persistence, and compatibility as material surfaces when the evidence makes them relevant.
- Impact surface IDs and risk IDs must be unique, non-empty canonical IDs without whitespace. Risk "surface_ids" must reference existing impact surface IDs.
- Impact surface category must be one of %s. Impact surface status must be one of %s.
- Candidate risk severity must be one of %s. Candidate risk status must be one of %s.
- Each impact surface and candidate risk requires either non-empty "evidence_summary" or at least one "evidence_refs" entry.
- Scope evidence refs are pre-probe only: "kind" must be one of %s, and "probe", "probe_command", "probe_id", and "command_index" are forbidden in the probe plan.
- If "probes" is empty, "no_probe_reason" must be non-empty, every impact surface status must be %q, every candidate risk status must be %q, and "no_probe_reason" must name every checked surface ID and checked risk ID. If "probes" is non-empty, omit "no_probe_reason" or set it to "".
- Probe IDs must be unique, non-empty, canonical IDs without whitespace.
- "mode" must be one of %q, %q, %q.
- Each probe must contain 1 to %d commands. Each command "command" is an executable name only: no whitespace, no null byte, no slash, no backslash.
- "args" is a JSON array of already-split arguments. Do not output shell pipelines, redirects, env assignments, command strings, or quoted command lines.
- "work_dir" is optional. When present, it must be "." or a canonical relative path.
- "files" is for generated files in isolated modes only. It must be empty when mode is %q. Paths must be canonical relative paths, unique per probe, with at most %d files, %d bytes per file, and %d total bytes per probe.
- "timeout_seconds" and "max_output_bytes" are optional non-negative integers. Their maximums are %d seconds and %d bytes.

Mode command contract:
- %q runs against the original repository in a read-only hardened environment. It allows commands: %s. Use it for additional reads and lightweight confirmation that must not mutate the worktree.
- %q runs only against generated scratch files. It allows commands: %s. Use it for repo-clean reproductions or small experiments that do not require the real worktree. Python commands must name a single script path; "go" is limited to "go run" of one .go file.
- %q runs against an isolated copy of the repository plus generated files. It allows commands: %s. Use it for real tests or change-impact verification where writes/build artifacts are expected inside the sandbox. Path-like args must stay inside the sandbox/repo copy.
- Do not plan broad speculative test suites. Each probe purpose must name the candidate risk or unverified surface it will confirm or falsify.
`,
		ReviewProbePlanSchemaVersionV2,
		TargetCurrentChanges,
		ReviewProbeImpactSurfaceChangedFile,
		ReviewEvidenceKindDiff,
		ReviewProbeImpactSurfaceNeedsProbe,
		ReviewGroupSeverityMedium,
		ReviewEvidenceKindDiff,
		ReviewProbeCandidateRiskNeedsProbe,
		MaxReviewProbePlanPurposeBytes,
		ReviewProbeHostReadOnly,
		TargetCurrentChanges,
		MaxReviewProbePlanProbes,
		quoteAndJoinSortedReviewPromptValues(reviewProbeImpactSurfaceCategoryPromptValues()),
		quoteAndJoinSortedReviewPromptValues(reviewProbeImpactSurfaceStatusPromptValues()),
		quoteAndJoinSortedReviewPromptValues(reviewGroupSeverityPromptValues()),
		quoteAndJoinSortedReviewPromptValues(reviewProbeCandidateRiskStatusPromptValues()),
		quoteAndJoinSortedReviewPromptValues(reviewProbePlanPreProbeEvidenceKindPromptValues()),
		ReviewProbeImpactSurfaceChecked,
		ReviewProbeCandidateRiskCheckedByEvidence,
		ReviewProbeHostReadOnly,
		ReviewProbeScratchOnly,
		ReviewProbeRepoSandbox,
		MaxReviewProbePlanCommands,
		ReviewProbeHostReadOnly,
		MaxReviewProbePlanFiles,
		MaxReviewProbePlanFileContentBytes,
		MaxReviewProbePlanTotalFileContentBytes,
		MaxReviewProbePlanTimeoutSeconds,
		MaxReviewProbePlanMaxOutputBytes,
		ReviewProbeHostReadOnly,
		sortedQuotedHostReadOnlyCommandNames(),
		ReviewProbeScratchOnly,
		sortedQuotedScratchOnlyCommandNames(),
		ReviewProbeRepoSandbox,
		sortedQuotedRepoSandboxCommandNames(),
	)
}

func reviewReportPromptContract() string {
	return fmt.Sprintf(`Strict JSON contract:
- The decoder rejects unknown fields. Use only the fields listed here.
- Top-level object:
  {
    "schema_version": %q,
    "target_kind": %q,
    "custom_instructions": "optional copy of the request instructions",
    "generated_at": "RFC3339 timestamp",
    "overall_verification_status": %q,
    "verdict": %q,
    "summary": "short final summary",
    "root_cause_groups": [],
    "probe_summaries": [],
    "checked_surfaces": [],
    "unverified_surfaces": [],
    "residual_risks": []
  }
- There is no top-level "findings" or "has_findings" field. Findings must be nested under "root_cause_groups[].findings".
- "schema_version" must be %q and "target_kind" must be %q.
- "verdict" must be one of %q, %q, %q.
- "overall_verification_status" and group "verification_status" must be one of %q, %q, %q, %q, %q.
- Root cause groups use: {"id","title","summary","severity","verification_status","fix_strategy","do_not_fix_by","verification_plan","findings","checked_surfaces","unverified_surfaces","residual_risks"}. "id" and "title" are required. Group IDs must be unique and contain no whitespace. Severity must be one of %q, %q, %q, %q, %q.
- Findings use: {"id","title","summary","evidence_refs","checked_surfaces","unverified_surfaces","residual_risks"}. "title" is required. Finding IDs are optional but must be unique when provided and contain no whitespace.
- Evidence refs use: {"kind","summary","probe_id","command_index","path","line","snippet"}. "kind" must be one of %q, %q, %q, %q, %q, %q. "probe_command" refs require both "probe_id" and zero-based "command_index". "file", "diff", and "rule_file" refs require "path". "line" must be non-negative; line > 0 requires "path". Paths must be canonical repo-relative evidence paths.
- Surface coverage entries use: {"surface_id","summary","evidence_refs"}. "surface_id" is required and must contain no whitespace.
- Residual risks use: {"id","summary","suggested_mitigation","evidence_refs"}. "summary" is required.
- Probe summaries must preserve the supplied "Probe Summaries For Report Schema" entries. Do not invent probe IDs. Probe summary modes must be one of %q, %q, %q. Probe and command statuses must be one of %q, %q, %q, %q, %q.

Verdict contract:
- %q: "overall_verification_status" must be %q or %q, and "root_cause_groups" must be empty.
- %q: "overall_verification_status" must be %q or %q, at least one root cause group is required, and each group "verification_status" must be %q or %q. Each root cause group must include at least one "findings" item, non-empty "fix_strategy", and at least one "verification_plan" item. Each finding must include at least one "evidence_refs" item.
- %q: "overall_verification_status" must be %q, %q, or %q, and the report must include a blocked reason in "summary", "unverified_surfaces", "residual_risks", or a blocked/timed-out/mutated probe summary.
`,
		ReviewReportSchemaVersionV1,
		TargetCurrentChanges,
		ReviewVerificationVerified,
		ReviewVerdictClean,
		ReviewReportSchemaVersionV1,
		TargetCurrentChanges,
		ReviewVerdictClean,
		ReviewVerdictHasFindings,
		ReviewVerdictBlocked,
		ReviewVerificationVerified,
		ReviewVerificationPartiallyVerified,
		ReviewVerificationUnverified,
		ReviewVerificationNotApplicable,
		ReviewVerificationBlockedOrInconclusive,
		ReviewGroupSeverityCritical,
		ReviewGroupSeverityHigh,
		ReviewGroupSeverityMedium,
		ReviewGroupSeverityLow,
		ReviewGroupSeverityInfo,
		ReviewEvidenceKindProbeCommand,
		ReviewEvidenceKindProbe,
		ReviewEvidenceKindFile,
		ReviewEvidenceKindDiff,
		ReviewEvidenceKindGitStatus,
		ReviewEvidenceKindRuleFile,
		ReviewProbeHostReadOnly,
		ReviewProbeScratchOnly,
		ReviewProbeRepoSandbox,
		ReviewProbePassed,
		ReviewProbeFailed,
		ReviewProbeBlocked,
		ReviewProbeTimedOut,
		ReviewProbeMutatedWorktree,
		ReviewVerdictClean,
		ReviewVerificationVerified,
		ReviewVerificationPartiallyVerified,
		ReviewVerdictHasFindings,
		ReviewVerificationVerified,
		ReviewVerificationPartiallyVerified,
		ReviewVerificationVerified,
		ReviewVerificationPartiallyVerified,
		ReviewVerdictBlocked,
		ReviewVerificationUnverified,
		ReviewVerificationPartiallyVerified,
		ReviewVerificationBlockedOrInconclusive,
	)
}

func sortedQuotedHostReadOnlyCommandNames() string {
	names := make([]string, 0, len(hostReadOnlyCommandSpecs))
	for name := range hostReadOnlyCommandSpecs {
		names = append(names, name)
	}
	return quoteAndJoinSortedReviewPromptValues(names)
}

func sortedQuotedScratchOnlyCommandNames() string {
	names := make([]string, 0, len(scratchOnlyCommandSpecs))
	for name := range scratchOnlyCommandSpecs {
		names = append(names, name)
	}
	return quoteAndJoinSortedReviewPromptValues(names)
}

func sortedQuotedRepoSandboxCommandNames() string {
	names := make([]string, 0, len(repoSandboxCommandSpecs))
	for name := range repoSandboxCommandSpecs {
		names = append(names, name)
	}
	return quoteAndJoinSortedReviewPromptValues(names)
}

func reviewProbeImpactSurfaceCategoryPromptValues() []string {
	values := make([]string, 0, len(reviewProbeImpactSurfaceCategories))
	for _, category := range reviewProbeImpactSurfaceCategories {
		values = append(values, string(category))
	}
	return values
}

func reviewProbeImpactSurfaceStatusPromptValues() []string {
	values := make([]string, 0, len(reviewProbeImpactSurfaceStatuses))
	for _, status := range reviewProbeImpactSurfaceStatuses {
		values = append(values, string(status))
	}
	return values
}

func reviewProbeCandidateRiskStatusPromptValues() []string {
	values := make([]string, 0, len(reviewProbeCandidateRiskStatuses))
	for _, status := range reviewProbeCandidateRiskStatuses {
		values = append(values, string(status))
	}
	return values
}

func reviewGroupSeverityPromptValues() []string {
	values := make([]string, 0, len(reviewGroupSeverities))
	for _, severity := range reviewGroupSeverities {
		values = append(values, string(severity))
	}
	return values
}

func reviewProbePlanPreProbeEvidenceKindPromptValues() []string {
	values := make([]string, 0, len(reviewEvidenceKinds))
	for _, kind := range reviewEvidenceKinds {
		if isReviewProbePlanPreProbeEvidenceKind(kind) {
			values = append(values, kind)
		}
	}
	return values
}

func quoteAndJoinSortedReviewPromptValues(values []string) string {
	sort.Strings(values)
	quoted := make([]string, 0, len(values))
	for _, value := range values {
		quoted = append(quoted, fmt.Sprintf("%q", value))
	}
	return strings.Join(quoted, ", ")
}
