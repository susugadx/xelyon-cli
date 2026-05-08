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
- "target_kind" must be %q. "probes" must contain at most %d entries.
- If "probes" is empty, "no_probe_reason" must be non-empty. If "probes" is non-empty, omit "no_probe_reason" or set it to "".
- Probe IDs must be unique, non-empty, canonical IDs without whitespace.
- "mode" must be one of %q, %q, %q.
- Each probe must contain 1 to %d commands. Each command "command" is an executable name only: no whitespace, no null byte, no slash, no backslash.
- "args" is a JSON array of already-split arguments. Do not output shell pipelines, redirects, env assignments, command strings, or quoted command lines.
- "work_dir" is optional. When present, it must be "." or a canonical relative path.
- "files" is for generated files in isolated modes only. It must be empty when mode is %q. Paths must be canonical relative paths, unique per probe, with at most %d files, %d bytes per file, and %d total bytes per probe.
- "timeout_seconds" and "max_output_bytes" are optional non-negative integers. Their maximums are %d seconds and %d bytes.

Mode command contract:
- %q runs against the original repository in a read-only hardened environment. It allows commands: %s. Prefer focused read-only inspection and normal test commands only.
- %q runs only against generated scratch files. It allows commands: %s. Python commands must name a single script path; "go" is limited to "go run" of one .go file.
- %q runs against an isolated copy of the repository plus generated files. It allows commands: %s. Path-like args must stay inside the sandbox/repo copy.
`,
		ReviewProbePlanSchemaVersionV1,
		TargetCurrentChanges,
		MaxReviewProbePlanPurposeBytes,
		ReviewProbeHostReadOnly,
		TargetCurrentChanges,
		MaxReviewProbePlanProbes,
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
- Evidence refs use: {"kind","summary","probe_id","command_index","path","line","snippet"}. "kind" must be one of %q, %q, %q, %q, %q, %q. "probe_command" refs require both "probe_id" and zero-based "command_index". "line" must be non-negative; line > 0 requires "path". Paths must be canonical repo-relative evidence paths.
- Surface coverage entries use: {"surface_id","summary","evidence_refs"}. "surface_id" is required and must contain no whitespace.
- Residual risks use: {"id","summary","suggested_mitigation","evidence_refs"}. "summary" is required.
- Probe summaries must preserve the supplied "Probe Summaries For Report Schema" entries. Do not invent probe IDs. Probe summary modes must be one of %q, %q, %q. Probe and command statuses must be one of %q, %q, %q, %q, %q.

Verdict contract:
- %q: "overall_verification_status" must be %q or %q, and "root_cause_groups" must be empty.
- %q: "overall_verification_status" must be %q or %q, at least one root cause group is required, and each group "verification_status" must be %q or %q.
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

func quoteAndJoinSortedReviewPromptValues(values []string) string {
	sort.Strings(values)
	quoted := make([]string, 0, len(values))
	for _, value := range values {
		quoted = append(quoted, fmt.Sprintf("%q", value))
	}
	return strings.Join(quoted, ", ")
}
