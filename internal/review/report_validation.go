package review

import (
	"fmt"
	"strings"
)

// ValidateReviewReport は schema v2 の review report 契約を検証する。
func ValidateReviewReport(report ReviewReport) error {
	if err := validateReportBasicFields(report); err != nil {
		return err
	}

	probeSummariesByID, err := validateProbeSummaries(report.ProbeSummaries)
	if err != nil {
		return err
	}

	if err := validateRootCauseGroups(report.RootCauseGroups); err != nil {
		return err
	}

	if err := validateReportRequiredContent(report); err != nil {
		return err
	}

	if err := validateEvidenceReferences(report, probeSummariesByID); err != nil {
		return err
	}

	if err := validateVerdictContract(report); err != nil {
		return err
	}

	if err := validateReviewReportScopeCoverageSemanticContract(report); err != nil {
		return err
	}

	return nil
}

func validateReportBasicFields(report ReviewReport) error {
	if report.SchemaVersion != ReviewReportSchemaVersionV2 {
		return fmt.Errorf("schema_version must be %q: got %q", ReviewReportSchemaVersionV2, report.SchemaVersion)
	}
	if report.TargetKind != TargetCurrentChanges {
		return fmt.Errorf("target_kind must be %q: got %q", TargetCurrentChanges, report.TargetKind)
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

func validateEvidenceRefs(field string, refs []ReviewEvidenceRef, probeSummariesByID map[string]ReviewProbeSummary) error {
	for i, ref := range refs {
		if err := validateEvidenceRef(fmt.Sprintf("%s[%d]", field, i), ref, probeSummariesByID); err != nil {
			return err
		}
	}
	return nil
}

func validateEvidenceRef(field string, ref ReviewEvidenceRef, probeSummariesByID map[string]ReviewProbeSummary) error {
	if !isKnownReviewEvidenceKind(ref.Kind) {
		return fmt.Errorf("%s.kind must be known enum value: got %q", field, ref.Kind)
	}

	probeID, err := validateOptionalProbeID(field+".probe_id", ref.ProbeID)
	if err != nil {
		return err
	}
	if ref.Kind == ReviewEvidenceKindProbeCommand {
		if probeID == "" {
			return fmt.Errorf("%s.probe_id is required when kind=%q", field, ReviewEvidenceKindProbeCommand)
		}
		if ref.CommandIndex == nil {
			return fmt.Errorf("%s.command_index is required when kind=%q", field, ReviewEvidenceKindProbeCommand)
		}
	}

	if ref.CommandIndex != nil && probeID == "" {
		return fmt.Errorf("%s.command_index requires probe_id", field)
	}

	var summary ReviewProbeSummary
	if probeID != "" {
		var exists bool
		summary, exists = probeSummariesByID[probeID]
		if !exists {
			return fmt.Errorf("%s.probe_id references unknown probe_id %q", field, probeID)
		}
	}

	if ref.CommandIndex != nil {
		if *ref.CommandIndex < 0 {
			return fmt.Errorf("%s.command_index must be >= 0: got %d", field, *ref.CommandIndex)
		}
		if probeID != "" && *ref.CommandIndex >= len(summary.Commands) {
			return fmt.Errorf("%s.command_index out of range: got %d, commands=%d", field, *ref.CommandIndex, len(summary.Commands))
		}
	}

	if ref.Line < 0 {
		return fmt.Errorf("%s.line must be >= 0: got %d", field, ref.Line)
	}
	evidencePath, err := validateOptionalEvidencePath(field+".path", ref.Path)
	if err != nil {
		return err
	}
	if ref.Line > 0 && evidencePath == "" {
		return fmt.Errorf("%s.path is required when line > 0", field)
	}
	if evidencePath != "" {
		if err := validateEvidencePath(field+".path", evidencePath); err != nil {
			return err
		}
	}
	if reviewEvidenceKindRequiresPath(ref.Kind) && evidencePath == "" {
		return fmt.Errorf("%s.path is required when kind=%q", field, ref.Kind)
	}

	return nil
}

func reviewEvidenceKindRequiresPath(kind string) bool {
	switch kind {
	case ReviewEvidenceKindFile, ReviewEvidenceKindDiff, ReviewEvidenceKindRuleFile:
		return true
	default:
		return false
	}
}

func validateEvidencePath(field, candidate string) error {
	return validateReviewCanonicalRelativePath(field, candidate, reviewRelativePathValidationPolicy{
		pathKind:         "repo-relative path",
		rejectWhitespace: false,
	})
}

func validateRequiredProbeID(field, candidate string) (string, error) {
	if strings.TrimSpace(candidate) == "" {
		return "", fmt.Errorf("%s must be non-empty", field)
	}
	return validateOptionalProbeID(field, candidate)
}

func validateRequiredReportID(field, candidate string) (string, error) {
	if candidate == "" {
		return "", fmt.Errorf("%s must be non-empty", field)
	}
	return validateOptionalReportID(field, candidate)
}

func validateOptionalReportID(field, candidate string) (string, error) {
	trimmed := strings.TrimSpace(candidate)
	if trimmed == "" {
		if candidate != "" {
			return "", fmt.Errorf("%s must be canonical report ID without whitespace: got %q", field, candidate)
		}
		return "", nil
	}
	if trimmed != candidate {
		return "", fmt.Errorf("%s must be canonical report ID without leading/trailing whitespace: got %q", field, candidate)
	}
	if containsAnyWhitespace(candidate) {
		return "", fmt.Errorf("%s must not include whitespace: got %q", field, candidate)
	}
	return candidate, nil
}

func validateOptionalProbeID(field, candidate string) (string, error) {
	trimmed := strings.TrimSpace(candidate)
	if trimmed == "" {
		if candidate != "" {
			return "", fmt.Errorf("%s must be canonical probe_id without leading/trailing whitespace: got %q", field, candidate)
		}
		return "", nil
	}
	if trimmed != candidate {
		return "", fmt.Errorf("%s must be canonical probe_id without leading/trailing whitespace: got %q", field, candidate)
	}
	return candidate, nil
}

func validateOptionalEvidencePath(field, candidate string) (string, error) {
	trimmed := strings.TrimSpace(candidate)
	if trimmed == "" {
		if candidate != "" {
			return "", fmt.Errorf("%s must be canonical repo-relative path without leading/trailing whitespace: got %q", field, candidate)
		}
		return "", nil
	}
	if trimmed != candidate {
		return "", fmt.Errorf("%s must be canonical repo-relative path without leading/trailing whitespace: got %q", field, candidate)
	}
	return candidate, nil
}

func validateVerdictContract(report ReviewReport) error {
	switch report.Verdict {
	case ReviewVerdictClean:
		return validateCleanVerdictContract(report)
	case ReviewVerdictHasFindings:
		return validateHasFindingsVerdictContract(report)
	case ReviewVerdictBlocked:
		return validateBlockedVerdictContract(report)
	default:
		return fmt.Errorf("verdict must be one of %q, %q, %q: got %q", ReviewVerdictClean, ReviewVerdictHasFindings, ReviewVerdictBlocked, report.Verdict)
	}
}

func validateCleanVerdictContract(report ReviewReport) error {
	switch report.OverallVerificationStatus {
	case ReviewVerificationVerified, ReviewVerificationPartiallyVerified:
	default:
		return fmt.Errorf("verdict %q requires overall_verification_status to be %q or %q: got %q",
			ReviewVerdictClean,
			ReviewVerificationVerified,
			ReviewVerificationPartiallyVerified,
			report.OverallVerificationStatus,
		)
	}
	if len(report.RootCauseGroups) > 0 {
		return fmt.Errorf("verdict %q requires root_cause_groups to be empty", ReviewVerdictClean)
	}
	return nil
}

func validateHasFindingsVerdictContract(report ReviewReport) error {
	switch report.OverallVerificationStatus {
	case ReviewVerificationVerified, ReviewVerificationPartiallyVerified:
	default:
		return fmt.Errorf("verdict %q requires overall_verification_status to be %q or %q: got %q",
			ReviewVerdictHasFindings,
			ReviewVerificationVerified,
			ReviewVerificationPartiallyVerified,
			report.OverallVerificationStatus,
		)
	}
	if err := validateHasFindingsRootCauseGroupsVerdictContract(report.RootCauseGroups); err != nil {
		return err
	}
	return nil
}

func validateHasFindingsRootCauseGroupsVerdictContract(groups []ReviewRootCauseGroup) error {
	if len(groups) == 0 {
		return fmt.Errorf("verdict %q requires at least one root_cause_group", ReviewVerdictHasFindings)
	}
	for i, group := range groups {
		switch group.VerificationStatus {
		case ReviewVerificationVerified, ReviewVerificationPartiallyVerified:
		default:
			return fmt.Errorf("verdict %q requires root_cause_groups[%d].verification_status to be %q or %q: got %q",
				ReviewVerdictHasFindings,
				i,
				ReviewVerificationVerified,
				ReviewVerificationPartiallyVerified,
				group.VerificationStatus,
			)
		}
	}
	for i, group := range groups {
		groupField := fmt.Sprintf("root_cause_groups[%d]", i)
		if len(group.Findings) == 0 {
			return fmt.Errorf("%s.findings must contain at least one finding", groupField)
		}
		for j, finding := range group.Findings {
			if len(finding.EvidenceRefs) == 0 {
				return fmt.Errorf("%s.findings[%d].evidence_refs must contain at least one evidence ref", groupField, j)
			}
		}
		if strings.TrimSpace(group.FixStrategy) == "" {
			return fmt.Errorf("%s.fix_strategy must be non-empty", groupField)
		}
		if len(group.VerificationPlan) == 0 {
			return fmt.Errorf("%s.verification_plan must contain at least one item", groupField)
		}
	}
	return nil
}

func validateBlockedVerdictContract(report ReviewReport) error {
	switch report.OverallVerificationStatus {
	case ReviewVerificationUnverified, ReviewVerificationPartiallyVerified, ReviewVerificationBlockedOrInconclusive:
	default:
		return fmt.Errorf("verdict %q requires overall_verification_status to be %q, %q, or %q: got %q",
			ReviewVerdictBlocked,
			ReviewVerificationUnverified,
			ReviewVerificationPartiallyVerified,
			ReviewVerificationBlockedOrInconclusive,
			report.OverallVerificationStatus,
		)
	}
	if !hasBlockedReason(report) {
		return fmt.Errorf("verdict %q requires blocked reason in summary, unverified_surfaces, residual_risks, or blocked probe_summaries status", ReviewVerdictBlocked)
	}
	return nil
}

func isKnownReviewVerdict(verdict ReviewVerdict) bool {
	switch verdict {
	case ReviewVerdictClean, ReviewVerdictHasFindings, ReviewVerdictBlocked:
		return true
	default:
		return false
	}
}

func isKnownReviewVerificationStatus(status ReviewVerificationStatus) bool {
	switch status {
	case ReviewVerificationVerified,
		ReviewVerificationPartiallyVerified,
		ReviewVerificationUnverified,
		ReviewVerificationNotApplicable,
		ReviewVerificationBlockedOrInconclusive:
		return true
	default:
		return false
	}
}

func isKnownReviewGroupSeverity(severity ReviewGroupSeverity) bool {
	for _, known := range reviewGroupSeverities {
		if severity == known {
			return true
		}
	}
	return false
}

func isKnownReviewEvidenceKind(kind string) bool {
	for _, known := range reviewEvidenceKinds {
		if kind == known {
			return true
		}
	}
	return false
}

func hasBlockedReason(report ReviewReport) bool {
	return hasLegacyBlockedReason(report) || hasScopeCoverageUnverified(report.ScopeCoverage)
}

func hasLegacyBlockedReason(report ReviewReport) bool {
	if strings.TrimSpace(report.Summary) != "" {
		return true
	}
	if hasSurfaceCoverageReason(report.UnverifiedSurfaces) {
		return true
	}
	if hasResidualRiskReason(report.ResidualRisks) {
		return true
	}
	for _, summary := range report.ProbeSummaries {
		switch summary.Status {
		case ReviewProbeBlocked, ReviewProbeTimedOut, ReviewProbeMutatedWorktree:
			return true
		}
	}
	return false
}

func hasSurfaceCoverageReason(surfaces []ReviewSurfaceCoverage) bool {
	for _, surface := range surfaces {
		if strings.TrimSpace(surface.SurfaceID) != "" || strings.TrimSpace(surface.Summary) != "" || len(surface.EvidenceRefs) > 0 {
			return true
		}
	}
	return false
}

func hasResidualRiskReason(risks []ReviewResidualRisk) bool {
	for _, risk := range risks {
		if strings.TrimSpace(risk.Summary) != "" {
			return true
		}
	}
	return false
}
