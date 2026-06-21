package report

import (
	"fmt"
)

func validateEvidenceReferences(report ReviewReport, probeSummariesByID map[string]ReviewProbeSummary) error {
	if err := validateSurfaceCoverageEvidenceRefs("checked_surfaces", report.CheckedSurfaces, probeSummariesByID); err != nil {
		return err
	}
	if err := validateSurfaceCoverageEvidenceRefs("unverified_surfaces", report.UnverifiedSurfaces, probeSummariesByID); err != nil {
		return err
	}
	if err := validateResidualRiskEvidenceRefs("residual_risks", report.ResidualRisks, probeSummariesByID); err != nil {
		return err
	}
	if err := validateReviewReportScopeCoverageEvidenceRefs("scope_coverage", report.ScopeCoverage, probeSummariesByID); err != nil {
		return err
	}

	for i, group := range report.RootCauseGroups {
		groupField := fmt.Sprintf("root_cause_groups[%d]", i)
		if err := validateFindingEvidenceRefs(groupField+".findings", group.Findings, probeSummariesByID); err != nil {
			return err
		}
		if err := validateSurfaceCoverageEvidenceRefs(groupField+".checked_surfaces", group.CheckedSurfaces, probeSummariesByID); err != nil {
			return err
		}
		if err := validateSurfaceCoverageEvidenceRefs(groupField+".unverified_surfaces", group.UnverifiedSurfaces, probeSummariesByID); err != nil {
			return err
		}
		if err := validateResidualRiskEvidenceRefs(groupField+".residual_risks", group.ResidualRisks, probeSummariesByID); err != nil {
			return err
		}
	}

	return nil
}

func validateFindingEvidenceRefs(field string, findings []ReviewFinding, probeSummariesByID map[string]ReviewProbeSummary) error {
	for i, finding := range findings {
		findingField := fmt.Sprintf("%s[%d]", field, i)
		if err := validateEvidenceRefs(findingField+".evidence_refs", finding.EvidenceRefs, probeSummariesByID); err != nil {
			return err
		}
		if err := validateSurfaceCoverageEvidenceRefs(findingField+".checked_surfaces", finding.CheckedSurfaces, probeSummariesByID); err != nil {
			return err
		}
		if err := validateSurfaceCoverageEvidenceRefs(findingField+".unverified_surfaces", finding.UnverifiedSurfaces, probeSummariesByID); err != nil {
			return err
		}
		if err := validateResidualRiskEvidenceRefs(findingField+".residual_risks", finding.ResidualRisks, probeSummariesByID); err != nil {
			return err
		}
	}
	return nil
}

func validateSurfaceCoverageEvidenceRefs(field string, surfaces []ReviewSurfaceCoverage, probeSummariesByID map[string]ReviewProbeSummary) error {
	for i, surface := range surfaces {
		if err := validateEvidenceRefs(fmt.Sprintf("%s[%d].evidence_refs", field, i), surface.EvidenceRefs, probeSummariesByID); err != nil {
			return err
		}
	}
	return nil
}

func validateResidualRiskEvidenceRefs(field string, risks []ReviewResidualRisk, probeSummariesByID map[string]ReviewProbeSummary) error {
	for i, risk := range risks {
		if err := validateEvidenceRefs(fmt.Sprintf("%s[%d].evidence_refs", field, i), risk.EvidenceRefs, probeSummariesByID); err != nil {
			return err
		}
	}
	return nil
}

// ValidateEvidenceRefs は evidence ref 群の schema shape と probe summary 参照を検証する。
func ValidateEvidenceRefs(field string, refs []ReviewEvidenceRef, probeSummariesByID map[string]ReviewProbeSummary) error {
	return validateEvidenceRefs(field, refs, probeSummariesByID)
}

func validateEvidenceRefs(field string, refs []ReviewEvidenceRef, probeSummariesByID map[string]ReviewProbeSummary) error {
	for i, ref := range refs {
		if err := validateEvidenceRef(fmt.Sprintf("%s[%d]", field, i), ref, probeSummariesByID); err != nil {
			return err
		}
	}
	return nil
}

// ValidateEvidenceRef は evidence ref 1 件の schema shape と probe summary 参照を検証する。
func ValidateEvidenceRef(field string, ref ReviewEvidenceRef, probeSummariesByID map[string]ReviewProbeSummary) error {
	return validateEvidenceRef(field, ref, probeSummariesByID)
}

func validateEvidenceRefsShape(field string, refs []ReviewEvidenceRef) error {
	for i, ref := range refs {
		if _, err := validateEvidenceRefShape(fmt.Sprintf("%s[%d]", field, i), ref); err != nil {
			return err
		}
	}
	return nil
}

func validateEvidenceRef(field string, ref ReviewEvidenceRef, probeSummariesByID map[string]ReviewProbeSummary) error {
	probeID, err := validateEvidenceRefShape(field, ref)
	if err != nil {
		return err
	}

	var summary ReviewProbeSummary
	if probeID != "" {
		var exists bool
		summary, exists = probeSummariesByID[probeID]
		if !exists {
			return fmt.Errorf("%s.probe_id references unknown probe_id %q", field, probeID)
		}
	}

	if ref.CommandIndex != nil && probeID != "" && *ref.CommandIndex >= len(summary.Commands) {
		return fmt.Errorf("%s.command_index out of range: got %d, commands=%d", field, *ref.CommandIndex, len(summary.Commands))
	}

	return nil
}

func validateEvidenceRefShape(field string, ref ReviewEvidenceRef) (string, error) {
	if !isKnownReviewEvidenceKind(ref.Kind) {
		return "", fmt.Errorf("%s.kind must be known enum value: got %q", field, ref.Kind)
	}
	if ref.Kind == ReviewEvidenceKindExternalDoc {
		return "", validateExternalDocEvidenceRefShape(field, ref)
	}
	if hasExternalDocEvidenceRefFields(ref) {
		return "", fmt.Errorf("%s external_doc fields are allowed only when kind=%q", field, ReviewEvidenceKindExternalDoc)
	}

	probeID, err := validateOptionalProbeID(field+".probe_id", ref.ProbeID)
	if err != nil {
		return "", err
	}
	if ref.Kind == ReviewEvidenceKindProbeCommand {
		if probeID == "" {
			return "", fmt.Errorf("%s.probe_id is required when kind=%q", field, ReviewEvidenceKindProbeCommand)
		}
		if ref.CommandIndex == nil {
			return "", fmt.Errorf("%s.command_index is required when kind=%q", field, ReviewEvidenceKindProbeCommand)
		}
	}

	if ref.CommandIndex != nil && probeID == "" {
		return "", fmt.Errorf("%s.command_index requires probe_id", field)
	}

	if ref.CommandIndex != nil {
		if *ref.CommandIndex < 0 {
			return "", fmt.Errorf("%s.command_index must be >= 0: got %d", field, *ref.CommandIndex)
		}
	}

	if ref.Line < 0 {
		return "", fmt.Errorf("%s.line must be >= 0: got %d", field, ref.Line)
	}
	evidencePath, err := validateOptionalEvidencePath(field+".path", ref.Path)
	if err != nil {
		return "", err
	}
	if ref.Line > 0 && evidencePath == "" {
		return "", fmt.Errorf("%s.path is required when line > 0", field)
	}
	if evidencePath != "" {
		if err := validateEvidencePath(field+".path", evidencePath); err != nil {
			return "", err
		}
	}
	if reviewEvidenceKindRequiresPath(ref.Kind) && evidencePath == "" {
		return "", fmt.Errorf("%s.path is required when kind=%q", field, ref.Kind)
	}

	return probeID, nil
}

func reviewEvidenceKindRequiresPath(kind string) bool {
	switch kind {
	case ReviewEvidenceKindFile, ReviewEvidenceKindDiff, ReviewEvidenceKindRuleFile:
		return true
	default:
		return false
	}
}
