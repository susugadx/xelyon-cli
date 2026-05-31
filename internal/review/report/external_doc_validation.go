package report

import (
	"fmt"
	"net/url"
	"strings"
	"time"
)

func validateExternalDocEvidenceRefShape(field string, ref ReviewEvidenceRef) error {
	if ref.ProbeID != "" {
		return fmt.Errorf("%s.probe_id is not allowed when kind=%q", field, ReviewEvidenceKindExternalDoc)
	}
	if ref.CommandIndex != nil {
		return fmt.Errorf("%s.command_index is not allowed when kind=%q", field, ReviewEvidenceKindExternalDoc)
	}
	if ref.Path != "" {
		return fmt.Errorf("%s.path is not allowed when kind=%q", field, ReviewEvidenceKindExternalDoc)
	}
	if ref.Line != 0 {
		return fmt.Errorf("%s.line is not allowed when kind=%q", field, ReviewEvidenceKindExternalDoc)
	}
	if ref.Snippet != "" {
		return fmt.Errorf("%s.snippet is not allowed when kind=%q", field, ReviewEvidenceKindExternalDoc)
	}
	if strings.TrimSpace(ref.DocID) == "" {
		return fmt.Errorf("%s.doc_id is required when kind=%q", field, ReviewEvidenceKindExternalDoc)
	}
	if strings.TrimSpace(ref.SnippetID) == "" {
		return fmt.Errorf("%s.snippet_id is required when kind=%q", field, ReviewEvidenceKindExternalDoc)
	}
	if ref.DocID != strings.TrimSpace(ref.DocID) || containsAnyWhitespace(ref.DocID) {
		return fmt.Errorf("%s.doc_id must be canonical without whitespace: got %q", field, ref.DocID)
	}
	if ref.SnippetID != strings.TrimSpace(ref.SnippetID) || containsAnyWhitespace(ref.SnippetID) {
		return fmt.Errorf("%s.snippet_id must be canonical without whitespace: got %q", field, ref.SnippetID)
	}
	if err := validateExternalDocEvidenceRefURL(field+".url", ref.URL); err != nil {
		return err
	}
	if _, err := time.Parse(time.RFC3339Nano, ref.FetchedAt); err != nil {
		return fmt.Errorf("%s.fetched_at must be RFC3339: %w", field, err)
	}
	if !isReviewSHA256Hash(ref.ContentHash) {
		return fmt.Errorf("%s.content_hash must be sha256:<64 hex chars>", field)
	}
	return nil
}

func validateExternalDocEvidenceRefURL(field, candidate string) error {
	if strings.TrimSpace(candidate) == "" {
		return fmt.Errorf("%s is required when kind=%q", field, ReviewEvidenceKindExternalDoc)
	}
	if candidate != strings.TrimSpace(candidate) {
		return fmt.Errorf("%s must not have leading/trailing whitespace: got %q", field, candidate)
	}
	parsed, err := url.Parse(candidate)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" {
		return fmt.Errorf("%s must be an HTTPS URL", field)
	}
	return nil
}

func hasExternalDocEvidenceRefFields(ref ReviewEvidenceRef) bool {
	return ref.DocID != "" || ref.SnippetID != "" || ref.URL != "" || ref.FetchedAt != "" || ref.ContentHash != ""
}

func isReviewSHA256Hash(candidate string) bool {
	if !strings.HasPrefix(candidate, "sha256:") || len(candidate) != len("sha256:")+64 {
		return false
	}
	for _, r := range candidate[len("sha256:"):] {
		switch {
		case r >= '0' && r <= '9':
		case r >= 'a' && r <= 'f':
		default:
			return false
		}
	}
	return true
}

// WalkReviewReportEvidenceRefs は report 内の evidence_refs を schema field path 付きで巡回する。
func WalkReviewReportEvidenceRefs(report ReviewReport, visit func(string, []ReviewEvidenceRef) error) error {
	if err := walkReviewSurfaceCoverageEvidenceRefs("checked_surfaces", report.CheckedSurfaces, visit); err != nil {
		return err
	}
	if err := walkReviewSurfaceCoverageEvidenceRefs("unverified_surfaces", report.UnverifiedSurfaces, visit); err != nil {
		return err
	}
	if err := walkReviewResidualRiskEvidenceRefs("residual_risks", report.ResidualRisks, visit); err != nil {
		return err
	}
	if report.ScopeCoverage != nil {
		for i, surface := range report.ScopeCoverage.ReviewedImpactSurfaces {
			if err := visit(fmt.Sprintf("scope_coverage.reviewed_impact_surfaces[%d].evidence_refs", i), surface.EvidenceRefs); err != nil {
				return err
			}
		}
		for i, risk := range report.ScopeCoverage.ReviewedCandidateRisks {
			if err := visit(fmt.Sprintf("scope_coverage.reviewed_candidate_risks[%d].evidence_refs", i), risk.EvidenceRefs); err != nil {
				return err
			}
		}
		for i, finding := range report.ScopeCoverage.NewFindingsFromReportPass {
			if err := visit(fmt.Sprintf("scope_coverage.new_findings_from_report_pass[%d].evidence_refs", i), finding.EvidenceRefs); err != nil {
				return err
			}
		}
	}
	for i, group := range report.RootCauseGroups {
		groupField := fmt.Sprintf("root_cause_groups[%d]", i)
		if err := walkReviewFindingEvidenceRefs(groupField+".findings", group.Findings, visit); err != nil {
			return err
		}
		if err := walkReviewSurfaceCoverageEvidenceRefs(groupField+".checked_surfaces", group.CheckedSurfaces, visit); err != nil {
			return err
		}
		if err := walkReviewSurfaceCoverageEvidenceRefs(groupField+".unverified_surfaces", group.UnverifiedSurfaces, visit); err != nil {
			return err
		}
		if err := walkReviewResidualRiskEvidenceRefs(groupField+".residual_risks", group.ResidualRisks, visit); err != nil {
			return err
		}
	}
	return nil
}

// WalkReviewSaturationEvidenceRefs は saturation check 内の evidence_refs を schema field path 付きで巡回する。
func WalkReviewSaturationEvidenceRefs(check ReviewSaturationCheck, visit func(string, []ReviewEvidenceRef) error) error {
	for i, candidate := range check.AdditionalFindingCandidates {
		if err := visit(fmt.Sprintf("additional_finding_candidates[%d].evidence_refs", i), candidate.EvidenceRefs); err != nil {
			return err
		}
	}
	return nil
}

func walkReviewFindingEvidenceRefs(field string, findings []ReviewFinding, visit func(string, []ReviewEvidenceRef) error) error {
	for i, finding := range findings {
		findingField := fmt.Sprintf("%s[%d]", field, i)
		if err := visit(findingField+".evidence_refs", finding.EvidenceRefs); err != nil {
			return err
		}
		if err := walkReviewSurfaceCoverageEvidenceRefs(findingField+".checked_surfaces", finding.CheckedSurfaces, visit); err != nil {
			return err
		}
		if err := walkReviewSurfaceCoverageEvidenceRefs(findingField+".unverified_surfaces", finding.UnverifiedSurfaces, visit); err != nil {
			return err
		}
		if err := walkReviewResidualRiskEvidenceRefs(findingField+".residual_risks", finding.ResidualRisks, visit); err != nil {
			return err
		}
	}
	return nil
}

func walkReviewSurfaceCoverageEvidenceRefs(field string, surfaces []ReviewSurfaceCoverage, visit func(string, []ReviewEvidenceRef) error) error {
	for i, surface := range surfaces {
		if err := visit(fmt.Sprintf("%s[%d].evidence_refs", field, i), surface.EvidenceRefs); err != nil {
			return err
		}
	}
	return nil
}

func walkReviewResidualRiskEvidenceRefs(field string, risks []ReviewResidualRisk, visit func(string, []ReviewEvidenceRef) error) error {
	for i, risk := range risks {
		if err := visit(fmt.Sprintf("%s[%d].evidence_refs", field, i), risk.EvidenceRefs); err != nil {
			return err
		}
	}
	return nil
}
