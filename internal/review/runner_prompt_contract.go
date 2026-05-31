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
    "no_candidate_risk_reason": "",
    "probes": [
      {
        "id": "probe-1",
        "surface_ids": ["surface-1"],
        "risk_ids": ["risk-1"],
        "purpose": "confirm or falsify risk-1 on surface-1 with a bounded check, at most %d bytes",
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
- "target_kind" must be %q. "impact_surfaces" must contain at least one entry. "candidate_risks" may be empty only when no material candidate risk remains after evaluating every impact surface. "probes" must contain at most %d entries.
- Scope analysis order: first enumerate material "impact_surfaces"; then derive "candidate_risks" for material risks from those surfaces; then create probes only for evidence-backed risks or unverified material surfaces.
- Consider changed files, callers, tests, related search hits, related tests/context files, CLI, TUI, config, validator, prompt contract, JSON schema, sandbox, timeout, path validation, error handling, persistence, and compatibility as material surfaces when the evidence makes them relevant.
- Generic impact candidates in Evidence Markdown are review leads, not proof of impact. Do not report findings solely because a candidate exists, but do not ignore them when deciding "impact_surfaces".
- If generic impact candidates cannot be verified from current evidence, classify the relevant surface/risk as unverified or residual, or plan a bounded probe. Absence of generic impact candidates is not proof of no impact.
- Impact surface IDs and risk IDs must be unique, non-empty canonical IDs using only ASCII letters, digits, hyphen, or underscore. Risk "surface_ids" must reference existing impact surface IDs.
- Impact surface "summary" and "reason" must be non-empty. Candidate risk "summary" and "verification_strategy" must be non-empty.
- Impact surface category must be one of %s. Impact surface status must be one of %s.
- Candidate risk severity must be one of %s. Candidate risk status must be one of %s.
- Each impact surface and candidate risk requires either non-empty "evidence_summary" or at least one "evidence_refs" entry.
- Scope evidence refs are pre-probe only: "kind" must be one of %s, and "probe", "probe_command", "probe_id", and "command_index" are forbidden in the probe plan.
- "external_doc" refs require "doc_id", "snippet_id", "url", "fetched_at", and snippet "content_hash" copied from fetched external_docs snippets. Raw web search results are discovery-only and must not be cited as evidence refs.
- Every changed file path, rename old path, deleted/renamed file path, and inventory category path shown in Evidence Markdown must appear literally in at least one impact surface "evidence_summary" or impact surface "evidence_refs[].path".
- When the production, config, tests, docs, or generated inventory category is non-empty, impact surfaces must mention that category name or one path from that category.
- When generic impact candidates are present, impact surfaces must cover each candidate role group by naming the role, one candidate path, or one candidate token. Literal coverage of every candidate path is not required.
- Untracked files must be explicitly covered by an impact surface path/summary, the word "untracked", or "no_candidate_risk_reason"/"no_probe_reason" naming the untracked path or untracked state.
- If diff, changed/related context, or related search evidence is truncated, do not mark all impact surfaces checked; plan at least one bounded probe or keep material scope unverified through a probed surface.
- If generic impact candidates are truncated, do not return an all-checked no-probe plan.
- Do not use empty related context/search evidence as proof for an all-checked no-probe plan. A no-probe all-checked plan requires related context files or related search hits.
- If "candidate_risks" is empty, "no_candidate_risk_reason" must be non-empty, must mention every impact surface ID, and must explain why no material candidate risk remains for each named surface. Do not use generic "No risk" wording. If "candidate_risks" is non-empty, omit "no_candidate_risk_reason" or set it to "".
- If "probes" is empty, "no_probe_reason" must be non-empty, every impact surface status must be %q, every candidate risk status must be %q, and "no_probe_reason" must name every checked surface ID and checked risk ID. If "probes" is non-empty, omit "no_probe_reason" or set it to "".
- Probe IDs must be unique, non-empty canonical IDs using only ASCII letters, digits, hyphen, or underscore.
- Each probe must include "surface_ids" or "risk_ids" with at least one referenced ID. Probe "surface_ids" must reference existing impact surface IDs. Probe "risk_ids" must reference existing candidate risk IDs.
- Every impact surface with status "needs_probe" or "unverified" must be referenced directly by at least one probe "surface_ids" entry. Every candidate risk with status "needs_probe" or "unverified" must be referenced directly by at least one probe "risk_ids" entry. Checked surfaces and "checked_by_evidence" risks may remain unreferenced by probes.
- Each probe purpose must explain how the referenced surface or risk IDs will be confirmed or falsified.
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
- Do not plan broad speculative test suites. The validator requires each probe to be linked to the candidate risks or unverified material surfaces it will confirm or falsify.
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
    "residual_risks": [],
    "scope_coverage": {
      "reviewed_impact_surfaces": [
        {
          "surface_id": "surface-1",
          "status": %q,
          "summary": "how impact surface surface-1 was reviewed",
          "evidence_refs": [],
          "finding_ids": []
        }
      ],
      "reviewed_candidate_risks": [
        {
          "risk_id": "risk-1",
          "status": %q,
          "summary": "why risk-1 was dismissed, became a finding, remains residual, or is unverified",
          "evidence_refs": [],
          "finding_ids": []
        }
      ],
      "new_findings_from_report_pass": []
    }
  }
- There is no top-level "findings" or "has_findings" field. Findings must be nested under "root_cause_groups[].findings".
- Do not output top-level "computed_summary"; runner computes it after validation from the validated report and trusted probe summaries.
- "schema_version" must be %q and "target_kind" must be %q.
- "verdict" must be one of %q, %q, %q.
- "overall_verification_status" and group "verification_status" must be one of %q, %q, %q, %q, %q.
- Root cause groups use: {"id","title","summary","severity","verification_status","fix_strategy","do_not_fix_by","verification_plan","findings","checked_surfaces","unverified_surfaces","residual_risks"}. "id" and "title" are required. Group IDs must be unique and contain no whitespace. Severity must be one of %q, %q, %q, %q, %q.
- Findings use: {"id","title","summary","evidence_refs","checked_surfaces","unverified_surfaces","residual_risks"}. "title" is required. In runner reports with root cause groups, finding IDs are required, must be unique, and must contain no whitespace so scope_coverage can reference them.
- Evidence refs use: {"kind","summary","probe_id","command_index","path","line","snippet","doc_id","snippet_id","url","fetched_at","content_hash"}. "kind" must be one of %q, %q, %q, %q, %q, %q, %q. "probe_command" refs require both "probe_id" and zero-based "command_index". "file", "diff", and "rule_file" refs require "path". "line" must be non-negative; line > 0 requires "path". Paths must be canonical repo-relative evidence paths.
- "external_doc" refs require "doc_id", "snippet_id", "url", "fetched_at", and snippet "content_hash" copied from a fetched external_docs snippet in Evidence Markdown. Do not cite raw web search results; only fetched external_doc snippets are citation-capable.
- Surface coverage entries use: {"surface_id","summary","evidence_refs"}. "surface_id" is required and must contain no whitespace.
- Residual risks use: {"id","summary","suggested_mitigation","evidence_refs"}. "summary" is required.
- Scope coverage is required in runner reports. It must classify every ID from the Decoded Probe Plan exactly once: "reviewed_impact_surfaces" must contain each "impact_surfaces[].id" as "surface_id", and "reviewed_candidate_risks" must contain each "candidate_risks[].id" as "risk_id". Unknown or duplicate IDs are invalid.
- Impact surface scope status must be one of %s. Candidate risk scope status must be one of %s.
- Candidate risks must be classified as "finding", "dismissed", "residual_risk", or "unverified". Use "finding_ids" to connect scope coverage entries to root cause findings when a Pass1 risk becomes a finding. Finding IDs must reference "root_cause_groups[].findings[].id".
- "finding_ids" in reviewed impact surfaces and reviewed candidate risks must be empty unless that scope entry status is "finding".
- Every root cause finding ID must be connected from "scope_coverage" via an impact surface "finding_ids", candidate risk "finding_ids", or "new_findings_from_report_pass[].finding_ids".
- A "clean" verdict is allowed only when every impact surface status is %q and every candidate risk status is %q.
- A "clean" verdict is invalid when any supplied trusted probe summary status is %q, %q, %q, or %q, or when "mutated_worktree" is true.
- If a Pass1 impact surface status was "needs_probe" or "unverified", reporting that surface as %q requires an "evidence_refs" entry with kind "probe" or "probe_command" for a linked probe_id whose trusted probe summary status is %q.
- If a Pass1 candidate risk status was "needs_probe" or "unverified", reporting that risk as %q requires an "evidence_refs" entry with kind "probe" or "probe_command" for a linked probe_id whose trusted probe summary status is %q.
- If any linked trusted probe failed, was blocked, timed out, or mutated the worktree, do not classify that linked impact surface as %q or that linked candidate risk as %q; use "finding", "residual_risk", or "unverified" according to the evidence.
- Reviewed impact surface status %q must include non-empty "finding_ids"; each referenced finding must exist under "root_cause_groups" and include evidence_refs.
- Reviewed candidate risk status %q must include non-empty "finding_ids"; each referenced finding must exist under "root_cause_groups" and include evidence_refs.
- A "blocked" verdict must have unverified scope coverage or an existing blocked reason.
- Findings discovered during Pass2 that were not Pass1 candidate risks are allowed only as root cause findings connected through "scope_coverage.new_findings_from_report_pass[].finding_ids".
- Probe summaries must preserve the supplied "Probe Summaries For Report Schema" entries with the same count, same order, and same "probe_id" values. Do not invent probe IDs. Probe summary modes must be one of %q, %q, %q. Probe and command statuses must be one of %q, %q, %q, %q, %q.

Verdict contract:
- %q: "overall_verification_status" must be %q or %q, and "root_cause_groups" must be empty.
- %q: "overall_verification_status" must be %q or %q, at least one root cause group is required, and each group "verification_status" must be %q or %q. Each root cause group must include at least one "findings" item, non-empty "fix_strategy", and at least one "verification_plan" item. Each finding must include at least one "evidence_refs" item.
- %q: "overall_verification_status" must be %q, %q, or %q, and the report must include a blocked reason in "summary", "unverified_surfaces", "residual_risks", or a blocked/timed-out/mutated probe summary.
`,
		ReviewReportSchemaVersionV2,
		TargetCurrentChanges,
		ReviewVerificationVerified,
		ReviewVerdictClean,
		ReviewReportImpactSurfaceChecked,
		ReviewReportCandidateRiskDismissed,
		ReviewReportSchemaVersionV2,
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
		ReviewEvidenceKindExternalDoc,
		quoteAndJoinSortedReviewPromptValues(reviewReportImpactSurfaceStatusPromptValues()),
		quoteAndJoinSortedReviewPromptValues(reviewReportCandidateRiskStatusPromptValues()),
		ReviewReportImpactSurfaceChecked,
		ReviewReportCandidateRiskDismissed,
		ReviewProbeFailed,
		ReviewProbeBlocked,
		ReviewProbeTimedOut,
		ReviewProbeMutatedWorktree,
		ReviewReportImpactSurfaceChecked,
		ReviewProbePassed,
		ReviewReportCandidateRiskDismissed,
		ReviewProbePassed,
		ReviewReportImpactSurfaceChecked,
		ReviewReportCandidateRiskDismissed,
		ReviewReportImpactSurfaceFinding,
		ReviewReportCandidateRiskFinding,
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

func reviewReportImpactSurfaceStatusPromptValues() []string {
	values := make([]string, 0, len(reviewReportImpactSurfaceStatuses))
	for _, status := range reviewReportImpactSurfaceStatuses {
		values = append(values, string(status))
	}
	return values
}

func reviewReportCandidateRiskStatusPromptValues() []string {
	values := make([]string, 0, len(reviewReportCandidateRiskStatuses))
	for _, status := range reviewReportCandidateRiskStatuses {
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
