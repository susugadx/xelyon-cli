package report

import (
	"fmt"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/review/domain"
)

func validateReportBasicFields(report ReviewReport) error {
	if report.SchemaVersion != ReviewReportSchemaVersionV2 {
		return fmt.Errorf("schema_version must be %q: got %q", ReviewReportSchemaVersionV2, report.SchemaVersion)
	}
	if report.TargetKind != domain.TargetCurrentChanges {
		return fmt.Errorf("target_kind must be %q: got %q", domain.TargetCurrentChanges, report.TargetKind)
	}
	if report.GeneratedAt.IsZero() {
		return fmt.Errorf("generated_at must be non-zero")
	}
	if !isKnownReviewVerdict(report.Verdict) {
		return fmt.Errorf("verdict must be one of %q, %q, %q: got %q", ReviewVerdictClean, ReviewVerdictHasFindings, ReviewVerdictBlocked, report.Verdict)
	}
	if !isKnownReviewVerificationStatus(report.OverallVerificationStatus) {
		return fmt.Errorf("overall_verification_status must be known enum value: got %q", report.OverallVerificationStatus)
	}
	return nil
}

func validateProbeSummaries(summaries []ReviewProbeSummary) (map[string]ReviewProbeSummary, error) {
	probeSummariesByID := make(map[string]ReviewProbeSummary, len(summaries))
	for i, summary := range summaries {
		probeIDField := fmt.Sprintf("probe_summaries[%d].probe_id", i)
		probeID, err := validateRequiredProbeID(probeIDField, summary.ProbeID)
		if err != nil {
			return nil, err
		}
		if _, exists := probeSummariesByID[probeID]; exists {
			return nil, fmt.Errorf("probe_summaries[%d].probe_id duplicates %q", i, probeID)
		}
		if !isKnownReviewProbeMode(summary.Mode) {
			return nil, fmt.Errorf("probe_summaries[%d].mode must be known enum value: got %q", i, summary.Mode)
		}
		if !isKnownReviewProbeStatus(summary.Status) {
			return nil, fmt.Errorf("probe_summaries[%d].status must be known enum value: got %q", i, summary.Status)
		}
		for j, command := range summary.Commands {
			if strings.TrimSpace(command.Command) == "" {
				return nil, fmt.Errorf("probe_summaries[%d].commands[%d].command must be non-empty", i, j)
			}
			if !isKnownReviewProbeStatus(command.Status) {
				return nil, fmt.Errorf("probe_summaries[%d].commands[%d].status must be known enum value: got %q", i, j, command.Status)
			}
		}
		probeSummariesByID[probeID] = summary
	}
	return probeSummariesByID, nil
}

func validateRootCauseGroups(groups []ReviewRootCauseGroup) error {
	groupIDs := make(map[string]struct{}, len(groups))
	findingIDs := make(map[string]struct{})
	for i, group := range groups {
		groupID, err := validateRequiredReportID(fmt.Sprintf("root_cause_groups[%d].id", i), group.ID)
		if err != nil {
			return err
		}
		if _, exists := groupIDs[groupID]; exists {
			return fmt.Errorf("root_cause_groups[%d].id duplicates %q", i, groupID)
		}
		groupIDs[groupID] = struct{}{}
		if strings.TrimSpace(group.Title) == "" {
			return fmt.Errorf("root_cause_groups[%d].title must be non-empty", i)
		}
		if !isKnownReviewGroupSeverity(group.Severity) {
			return fmt.Errorf("root_cause_groups[%d].severity must be known enum value: got %q", i, group.Severity)
		}
		if !isKnownReviewVerificationStatus(group.VerificationStatus) {
			return fmt.Errorf("root_cause_groups[%d].verification_status must be known enum value: got %q", i, group.VerificationStatus)
		}
		for j, finding := range group.Findings {
			findingID, err := validateOptionalReportID(fmt.Sprintf("root_cause_groups[%d].findings[%d].id", i, j), finding.ID)
			if err != nil {
				return err
			}
			if findingID == "" {
				continue
			}
			if _, exists := findingIDs[findingID]; exists {
				return fmt.Errorf("root_cause_groups[%d].findings[%d].id duplicates %q", i, j, findingID)
			}
			findingIDs[findingID] = struct{}{}
		}
	}
	return nil
}

func validateReportRequiredContent(report ReviewReport) error {
	if err := validateSurfaceCoverages("checked_surfaces", report.CheckedSurfaces); err != nil {
		return err
	}
	if err := validateSurfaceCoverages("unverified_surfaces", report.UnverifiedSurfaces); err != nil {
		return err
	}
	if err := validateResidualRisks("residual_risks", report.ResidualRisks); err != nil {
		return err
	}
	if err := validateReviewReportScopeCoverageShape("scope_coverage", report.ScopeCoverage); err != nil {
		return err
	}

	for i, group := range report.RootCauseGroups {
		groupField := fmt.Sprintf("root_cause_groups[%d]", i)
		if err := validateFindings(groupField+".findings", group.Findings); err != nil {
			return err
		}
		if err := validateSurfaceCoverages(groupField+".checked_surfaces", group.CheckedSurfaces); err != nil {
			return err
		}
		if err := validateSurfaceCoverages(groupField+".unverified_surfaces", group.UnverifiedSurfaces); err != nil {
			return err
		}
		if err := validateResidualRisks(groupField+".residual_risks", group.ResidualRisks); err != nil {
			return err
		}
	}

	return nil
}

func validateFindings(field string, findings []ReviewFinding) error {
	for i, finding := range findings {
		findingField := fmt.Sprintf("%s[%d]", field, i)
		if strings.TrimSpace(finding.Title) == "" {
			return fmt.Errorf("%s.title must be non-empty", findingField)
		}
		if err := validateSurfaceCoverages(findingField+".checked_surfaces", finding.CheckedSurfaces); err != nil {
			return err
		}
		if err := validateSurfaceCoverages(findingField+".unverified_surfaces", finding.UnverifiedSurfaces); err != nil {
			return err
		}
		if err := validateResidualRisks(findingField+".residual_risks", finding.ResidualRisks); err != nil {
			return err
		}
	}
	return nil
}

func validateSurfaceCoverages(field string, surfaces []ReviewSurfaceCoverage) error {
	for i, surface := range surfaces {
		if _, err := validateRequiredReportID(fmt.Sprintf("%s[%d].surface_id", field, i), surface.SurfaceID); err != nil {
			return err
		}
	}
	return nil
}

func validateResidualRisks(field string, risks []ReviewResidualRisk) error {
	for i, risk := range risks {
		if strings.TrimSpace(risk.Summary) == "" {
			return fmt.Errorf("%s[%d].summary must be non-empty", field, i)
		}
	}
	return nil
}
