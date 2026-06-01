package modelinput

import (
	"fmt"

	reviewreport "github.com/susugadx/xelyon-cli/internal/review/report"
)

func reviewSaturationCheckPromptContract() string {
	return fmt.Sprintf(`Strict JSON contract:
- The decoder rejects unknown fields. Use only the fields listed here.
- Top-level object:
  {
    "schema_version": %q,
    "status": %q,
    "checked_summary": "short summary of what final report coverage was checked",
    "missing_surface_ids": ["surface-1"],
    "missing_risk_ids": ["risk-1"],
    "additional_finding_candidates": [
      {
        "summary": "existing-evidence finding candidate omitted from the finalized report",
        "evidence_refs": [
          {"kind": %q, "path": "internal/review/runner.go", "line": 1, "summary": "optional evidence summary"}
        ],
        "reason": "why this candidate is grounded in the supplied evidence and Pass1/report scope"
      }
    ],
    "revision_instructions": "revise the report to classify surface-1 and risk-1 or include the evidence-backed candidate"
  }
- "schema_version" must be %q.
- "status" must be one of %q, %q, %q.
- "checked_summary" must be non-empty. When status is %q, treat "checked_summary" as the block reason.
- This is a saturation check, not a new review. Check only whether the Finalized Review Report processed the Decoded Probe Plan's Pass1 "impact_surfaces" and "candidate_risks" plus the supplied evidence/probe context.
- Do not request tools, perform additional exploration, infer from missing evidence, or add speculative findings.
- Do not treat "scope_coverage" as saturated just because it repeats every Pass1 ID. Verify that each ID's status, summary, evidence_refs, finding links, residual risk, blocked state, or unverified state honestly reflects the supplied evidence and probe outcomes.
- Failed, blocked, timed-out, or "mutated_worktree" probe outcomes must be reflected in the report verdict, scope coverage statuses, residual risks, blocked/unverified status, or finding evidence. Do not let them become clean, checked, dismissed, or verified by wording alone.
- Shallow, empty, or absent related context/search evidence is not proof of no impact.
- Disabled, failed, truncated, or inconclusive review web search evidence is not external spec coverage.
- Fetched external_doc snippets are citation-capable evidence, but are not automatically official documentation or confirmed external specs. Source credibility values are "official_candidate", "third_party", and "unknown"; only "official_candidate" may be treated as an official-source candidate, not proof by itself. Do not infer official or authoritative status from search query wording, source title, URL label, snippet wording, or "official documentation" text alone. Do not treat them as confirmed official documentation when source_credibility is "unknown" or "third_party"; source_credibility and the cited snippet content must both support that claim.
- If source credibility is unclear, fetch failed, evidence is truncated, or review web search evidence is inconclusive, classify the relevant scope as unverified, residual, or blocked instead of confirmed. If fetched external_doc snippets contradict the implementation or report, classify the issue as a finding, residual risk, unverified scope, or blocked status according to the supplied evidence.
- When evidence, diff, related context, or probe output is truncated, do not bias toward saturated/clean/verified. Consider whether the report should be blocked, residual, unverified, or needs_revision.
- Trace review pressure signals from Evidence Markdown and the Decoded Probe Plan through "impact_surfaces", "candidate_risks", and the Finalized Review Report. Pressure signals include changed production/config/test/docs/generated files, generic impact candidates, prompt/JSON contract changes, validation changes, persistence/state changes, broad inventory categories, mutation outcomes, and truncation.
- If supplied evidence or probe output supports an additional finding candidate within the report scope and the report omitted it without classification, return %q with an "additional_finding_candidates" entry.
- "missing_surface_ids" may contain only IDs from Decoded Probe Plan "impact_surfaces[].id"; values must be unique canonical IDs using only ASCII letters, digits, hyphen, or underscore.
- "missing_risk_ids" may contain only IDs from Decoded Probe Plan "candidate_risks[].id"; values must be unique canonical IDs using only ASCII letters, digits, hyphen, or underscore.
- "additional_finding_candidates" is allowed only for candidates grounded in existing Evidence Markdown, Probe Summaries For Report Schema, Probe Result Context, or Finalized Review Report coverage. Do not use it for new exploration or speculation.
- Each additional finding candidate requires non-empty "summary", non-empty "reason", and at least one valid "evidence_refs" entry. Evidence refs use the same fields and constraints as review_report.v2 evidence refs. Probe-backed refs may reference only supplied probe summaries.
- Status %q requires "missing_surface_ids", "missing_risk_ids", "additional_finding_candidates", and "revision_instructions" to be empty.
- Status %q requires non-empty "revision_instructions" and at least one missing surface ID, missing risk ID, or additional finding candidate. Use it when the Finalized Review Report escapes, omits, or downplays pressure signals, evidence gaps, probe outcomes, or evidence/probe-backed finding candidates.
- Status %q means the supplied context is insufficient to determine saturation. Do not return a clean/saturated check when blocked.
- Output only this saturation check JSON object. Do not output review_report.v2, do not include markdown fences, and do not include top-level "computed_summary".
`,
		reviewreport.ReviewSaturationCheckSchemaVersionV1,
		reviewreport.ReviewSaturationStatusNeedsRevision,
		reviewreport.ReviewEvidenceKindFile,
		reviewreport.ReviewSaturationCheckSchemaVersionV1,
		reviewreport.ReviewSaturationStatusSaturated,
		reviewreport.ReviewSaturationStatusNeedsRevision,
		reviewreport.ReviewSaturationStatusBlocked,
		reviewreport.ReviewSaturationStatusBlocked,
		reviewreport.ReviewSaturationStatusNeedsRevision,
		reviewreport.ReviewSaturationStatusSaturated,
		reviewreport.ReviewSaturationStatusNeedsRevision,
		reviewreport.ReviewSaturationStatusBlocked,
	)
}
