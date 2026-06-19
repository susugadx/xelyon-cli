package modelinput

import (
	"fmt"

	"github.com/susugadx/xelyon-cli/internal/review/domain"
	reviewreport "github.com/susugadx/xelyon-cli/internal/review/report"
)

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
- "external_doc" refs require "doc_id", "snippet_id", "url", "fetched_at", and snippet "content_hash" copied from a fetched external_docs snippet in Evidence Markdown. Do not cite raw web search results; fetched external_doc snippets are citation-capable evidence, but are not automatically official documentation or confirmed external specs. Source credibility values are "official_candidate", "third_party", and "unknown"; only "official_candidate" may be treated as an official-source candidate, not proof by itself. Do not infer official or authoritative status from search query wording, source title, URL label, snippet wording, or "official documentation" text alone. Do not treat them as confirmed official documentation when source_credibility is "unknown" or "third_party"; source_credibility and the cited snippet content must both support that claim, and external_support.level/official_confirmation must allow confirmed official status.
%s
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
		reviewreport.ReviewReportSchemaVersionV2,
		domain.TargetCurrentChanges,
		reviewreport.ReviewVerificationVerified,
		reviewreport.ReviewVerdictClean,
		reviewreport.ReviewReportImpactSurfaceChecked,
		reviewreport.ReviewReportCandidateRiskDismissed,
		reviewreport.ReviewReportSchemaVersionV2,
		domain.TargetCurrentChanges,
		reviewreport.ReviewVerdictClean,
		reviewreport.ReviewVerdictHasFindings,
		reviewreport.ReviewVerdictBlocked,
		reviewreport.ReviewVerificationVerified,
		reviewreport.ReviewVerificationPartiallyVerified,
		reviewreport.ReviewVerificationUnverified,
		reviewreport.ReviewVerificationNotApplicable,
		reviewreport.ReviewVerificationBlockedOrInconclusive,
		reviewreport.ReviewGroupSeverityCritical,
		reviewreport.ReviewGroupSeverityHigh,
		reviewreport.ReviewGroupSeverityMedium,
		reviewreport.ReviewGroupSeverityLow,
		reviewreport.ReviewGroupSeverityInfo,
		reviewreport.ReviewEvidenceKindProbeCommand,
		reviewreport.ReviewEvidenceKindProbe,
		reviewreport.ReviewEvidenceKindFile,
		reviewreport.ReviewEvidenceKindDiff,
		reviewreport.ReviewEvidenceKindGitStatus,
		reviewreport.ReviewEvidenceKindRuleFile,
		reviewreport.ReviewEvidenceKindExternalDoc,
		reviewExternalSupportPromptGuardrails(),
		quoteAndJoinSortedReviewPromptValues(reviewReportImpactSurfaceStatusPromptValues()),
		quoteAndJoinSortedReviewPromptValues(reviewReportCandidateRiskStatusPromptValues()),
		reviewreport.ReviewReportImpactSurfaceChecked,
		reviewreport.ReviewReportCandidateRiskDismissed,
		domain.ReviewProbeFailed,
		domain.ReviewProbeBlocked,
		domain.ReviewProbeTimedOut,
		domain.ReviewProbeMutatedWorktree,
		reviewreport.ReviewReportImpactSurfaceChecked,
		domain.ReviewProbePassed,
		reviewreport.ReviewReportCandidateRiskDismissed,
		domain.ReviewProbePassed,
		reviewreport.ReviewReportImpactSurfaceChecked,
		reviewreport.ReviewReportCandidateRiskDismissed,
		reviewreport.ReviewReportImpactSurfaceFinding,
		reviewreport.ReviewReportCandidateRiskFinding,
		domain.ReviewProbeHostReadOnly,
		domain.ReviewProbeScratchOnly,
		domain.ReviewProbeRepoSandbox,
		domain.ReviewProbePassed,
		domain.ReviewProbeFailed,
		domain.ReviewProbeBlocked,
		domain.ReviewProbeTimedOut,
		domain.ReviewProbeMutatedWorktree,
		reviewreport.ReviewVerdictClean,
		reviewreport.ReviewVerificationVerified,
		reviewreport.ReviewVerificationPartiallyVerified,
		reviewreport.ReviewVerdictHasFindings,
		reviewreport.ReviewVerificationVerified,
		reviewreport.ReviewVerificationPartiallyVerified,
		reviewreport.ReviewVerificationVerified,
		reviewreport.ReviewVerificationPartiallyVerified,
		reviewreport.ReviewVerdictBlocked,
		reviewreport.ReviewVerificationUnverified,
		reviewreport.ReviewVerificationPartiallyVerified,
		reviewreport.ReviewVerificationBlockedOrInconclusive,
	)
}
