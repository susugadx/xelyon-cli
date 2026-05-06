package review

import (
	"fmt"
	"path"
	"strings"
	"unicode"
)

// ValidateReviewReport は schema v1 の review report 契約を検証する。
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

	if err := validateEvidenceReferences(report, probeSummariesByID); err != nil {
		return err
	}

	if err := validateVerdictContract(report); err != nil {
		return err
	}

	return nil
}

func validateReportBasicFields(report ReviewReport) error {
	if report.SchemaVersion != ReviewReportSchemaVersionV1 {
		return fmt.Errorf("schema_version must be %q: got %q", ReviewReportSchemaVersionV1, report.SchemaVersion)
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

	return nil
}

func validateEvidencePath(field, candidate string) error {
	if isReviewAbsolutePathLike(candidate) {
		return fmt.Errorf("%s must be repo-relative path: got absolute path %q", field, candidate)
	}
	if strings.Contains(candidate, `\`) {
		return fmt.Errorf("%s must use '/' separators: got %q", field, candidate)
	}
	for _, segment := range strings.Split(candidate, "/") {
		if segment == "." || segment == ".." {
			return fmt.Errorf("%s must not contain %q segment: got %q", field, segment, candidate)
		}
	}

	cleaned := path.Clean(candidate)
	if cleaned != candidate {
		return fmt.Errorf("%s must be canonical repo-relative path: got %q (canonical: %q)", field, candidate, cleaned)
	}
	return nil
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
	if containsAnyWhitespace(candidate) {
		return "", fmt.Errorf("%s must not include whitespace: got %q", field, candidate)
	}
	return candidate, nil
}

func containsAnyWhitespace(candidate string) bool {
	for _, r := range candidate {
		if unicode.IsSpace(r) {
			return true
		}
	}
	return false
}

func validateVerdictContract(report ReviewReport) error {
	switch report.Verdict {
	case ReviewVerdictClean:
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
	case ReviewVerdictHasFindings:
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
		if len(report.RootCauseGroups) == 0 {
			return fmt.Errorf("verdict %q requires at least one root_cause_group", ReviewVerdictHasFindings)
		}
		for i, group := range report.RootCauseGroups {
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
	case ReviewVerdictBlocked:
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
	default:
		return fmt.Errorf("verdict must be one of %q, %q, %q: got %q", ReviewVerdictClean, ReviewVerdictHasFindings, ReviewVerdictBlocked, report.Verdict)
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
	switch severity {
	case ReviewGroupSeverityCritical,
		ReviewGroupSeverityHigh,
		ReviewGroupSeverityMedium,
		ReviewGroupSeverityLow,
		ReviewGroupSeverityInfo:
		return true
	default:
		return false
	}
}

func isKnownReviewProbeStatus(status ReviewProbeStatus) bool {
	switch status {
	case ReviewProbePassed, ReviewProbeFailed, ReviewProbeBlocked, ReviewProbeTimedOut, ReviewProbeMutatedWorktree:
		return true
	default:
		return false
	}
}

func isKnownReviewEvidenceKind(kind string) bool {
	switch kind {
	case ReviewEvidenceKindProbeCommand,
		ReviewEvidenceKindProbe,
		ReviewEvidenceKindFile,
		ReviewEvidenceKindDiff,
		ReviewEvidenceKindGitStatus,
		ReviewEvidenceKindRuleFile:
		return true
	default:
		return false
	}
}

func hasBlockedReason(report ReviewReport) bool {
	if strings.TrimSpace(report.Summary) != "" {
		return true
	}
	if len(report.UnverifiedSurfaces) > 0 {
		return true
	}
	if len(report.ResidualRisks) > 0 {
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
