package review

import "fmt"

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
- "missing_surface_ids" may contain only IDs from Decoded Probe Plan "impact_surfaces[].id"; values must be unique canonical IDs using only ASCII letters, digits, hyphen, or underscore.
- "missing_risk_ids" may contain only IDs from Decoded Probe Plan "candidate_risks[].id"; values must be unique canonical IDs using only ASCII letters, digits, hyphen, or underscore.
- "additional_finding_candidates" is allowed only for candidates grounded in existing Evidence Markdown, Probe Summaries For Report Schema, Probe Result Context, or Finalized Review Report coverage. Do not use it for new exploration or speculation.
- Each additional finding candidate requires non-empty "summary", non-empty "reason", and at least one valid "evidence_refs" entry. Evidence refs use the same fields and constraints as review_report.v2 evidence refs. Probe-backed refs may reference only supplied probe summaries.
- Status %q requires "missing_surface_ids", "missing_risk_ids", "additional_finding_candidates", and "revision_instructions" to be empty.
- Status %q requires non-empty "revision_instructions" and at least one missing surface ID, missing risk ID, or additional finding candidate.
- Status %q means the supplied context is insufficient to determine saturation. Do not return a clean/saturated check when blocked.
- Output only this saturation check JSON object. Do not output review_report.v2, do not include markdown fences, and do not include top-level "computed_summary".
`,
		ReviewSaturationCheckSchemaVersionV1,
		ReviewSaturationStatusNeedsRevision,
		ReviewEvidenceKindFile,
		ReviewSaturationCheckSchemaVersionV1,
		ReviewSaturationStatusSaturated,
		ReviewSaturationStatusNeedsRevision,
		ReviewSaturationStatusBlocked,
		ReviewSaturationStatusBlocked,
		ReviewSaturationStatusSaturated,
		ReviewSaturationStatusNeedsRevision,
		ReviewSaturationStatusBlocked,
	)
}
