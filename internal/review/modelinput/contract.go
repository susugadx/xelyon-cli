package modelinput

import (
	"fmt"
	"sort"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/review/domain"
	reviewprobe "github.com/susugadx/xelyon-cli/internal/review/probe"
	reviewreport "github.com/susugadx/xelyon-cli/internal/review/report"
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
- "external_doc" refs require "doc_id", "snippet_id", "url", "fetched_at", and snippet "content_hash" copied from fetched external_docs snippets. Raw web search results are discovery-only and must not be cited as evidence refs. Fetched external_doc snippets are citation-capable evidence, but are not automatically official documentation or confirmed external specs. Source credibility values are "official_candidate", "third_party", and "unknown"; only "official_candidate" may be treated as an official-source candidate, not proof by itself. Do not infer official or authoritative status from search query wording, source title, URL label, snippet wording, or "official documentation" text alone. Do not treat them as confirmed official documentation when source_credibility is "unknown" or "third_party"; source_credibility and the cited snippet content must both support that claim, and external_support.level/official_confirmation must allow confirmed official status.
%s
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
		reviewprobe.ReviewProbePlanSchemaVersionV2,
		domain.TargetCurrentChanges,
		reviewprobe.ReviewProbeImpactSurfaceChangedFile,
		reviewreport.ReviewEvidenceKindDiff,
		reviewprobe.ReviewProbeImpactSurfaceNeedsProbe,
		reviewreport.ReviewGroupSeverityMedium,
		reviewreport.ReviewEvidenceKindDiff,
		reviewprobe.ReviewProbeCandidateRiskNeedsProbe,
		reviewprobe.MaxReviewProbePlanPurposeBytes,
		domain.ReviewProbeHostReadOnly,
		domain.TargetCurrentChanges,
		reviewprobe.MaxReviewProbePlanProbes,
		quoteAndJoinSortedReviewPromptValues(reviewProbeImpactSurfaceCategoryPromptValues()),
		quoteAndJoinSortedReviewPromptValues(reviewProbeImpactSurfaceStatusPromptValues()),
		quoteAndJoinSortedReviewPromptValues(reviewGroupSeverityPromptValues()),
		quoteAndJoinSortedReviewPromptValues(reviewProbeCandidateRiskStatusPromptValues()),
		quoteAndJoinSortedReviewPromptValues(reviewProbePlanPreProbeEvidenceKindPromptValues()),
		reviewExternalSupportPromptGuardrails(),
		reviewprobe.ReviewProbeImpactSurfaceChecked,
		reviewprobe.ReviewProbeCandidateRiskCheckedByEvidence,
		domain.ReviewProbeHostReadOnly,
		domain.ReviewProbeScratchOnly,
		domain.ReviewProbeRepoSandbox,
		reviewprobe.MaxReviewProbePlanCommands,
		domain.ReviewProbeHostReadOnly,
		reviewprobe.MaxReviewProbePlanFiles,
		reviewprobe.MaxReviewProbePlanFileContentBytes,
		reviewprobe.MaxReviewProbePlanTotalFileContentBytes,
		reviewprobe.MaxReviewProbePlanTimeoutSeconds,
		reviewprobe.MaxReviewProbePlanMaxOutputBytes,
		domain.ReviewProbeHostReadOnly,
		sortedQuotedHostReadOnlyCommandNames(),
		domain.ReviewProbeScratchOnly,
		sortedQuotedScratchOnlyCommandNames(),
		domain.ReviewProbeRepoSandbox,
		sortedQuotedRepoSandboxCommandNames(),
	)
}

func sortedQuotedHostReadOnlyCommandNames() string {
	return quoteAndJoinSortedReviewPromptValues(reviewprobe.HostReadOnlyCommandNames())
}

func sortedQuotedScratchOnlyCommandNames() string {
	return quoteAndJoinSortedReviewPromptValues(reviewprobe.ScratchOnlyCommandNames())
}

func sortedQuotedRepoSandboxCommandNames() string {
	return quoteAndJoinSortedReviewPromptValues(reviewprobe.RepoSandboxCommandNames())
}

func reviewProbeImpactSurfaceCategoryPromptValues() []string {
	categories := reviewprobe.KnownReviewProbeImpactSurfaceCategories()
	values := make([]string, 0, len(categories))
	for _, category := range categories {
		values = append(values, string(category))
	}
	return values
}

func reviewProbeImpactSurfaceStatusPromptValues() []string {
	statuses := reviewprobe.KnownReviewProbeImpactSurfaceStatuses()
	values := make([]string, 0, len(statuses))
	for _, status := range statuses {
		values = append(values, string(status))
	}
	return values
}

func reviewProbeCandidateRiskStatusPromptValues() []string {
	statuses := reviewprobe.KnownReviewProbeCandidateRiskStatuses()
	values := make([]string, 0, len(statuses))
	for _, status := range statuses {
		values = append(values, string(status))
	}
	return values
}

func reviewReportImpactSurfaceStatusPromptValues() []string {
	statuses := reviewreport.KnownReviewReportImpactSurfaceStatuses()
	values := make([]string, 0, len(statuses))
	for _, status := range statuses {
		values = append(values, string(status))
	}
	return values
}

func reviewReportCandidateRiskStatusPromptValues() []string {
	statuses := reviewreport.KnownReviewReportCandidateRiskStatuses()
	values := make([]string, 0, len(statuses))
	for _, status := range statuses {
		values = append(values, string(status))
	}
	return values
}

func reviewGroupSeverityPromptValues() []string {
	severities := reviewreport.KnownReviewGroupSeverities()
	values := make([]string, 0, len(severities))
	for _, severity := range severities {
		values = append(values, string(severity))
	}
	return values
}

func reviewProbePlanPreProbeEvidenceKindPromptValues() []string {
	evidenceKinds := reviewreport.KnownReviewEvidenceKinds()
	values := make([]string, 0, len(evidenceKinds))
	for _, kind := range evidenceKinds {
		if reviewprobe.IsReviewProbePlanPreProbeEvidenceKind(kind) {
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
