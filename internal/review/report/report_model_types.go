package report

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/susugadx/xelyon-cli/internal/review/domain"
)

const (
	// ReviewReportModelSchemaVersionV2 は provider が返す中間 review report model schema である。
	// Audit token: review_report_model.v2.
	ReviewReportModelSchemaVersionV2 = "xelyon.review.report_model.v2"
)

// ReviewReportModelOutput は review provider output 用の中間 DTO である。
// final artifact は ToReviewReport で review_report.v2 へ変換する。
type ReviewReportModelOutput struct {
	SchemaVersion             string                              `json:"schema_version"`
	TargetKind                domain.TargetKind                   `json:"target_kind"`
	CustomInstructions        string                              `json:"custom_instructions,omitempty"`
	GeneratedAt               time.Time                           `json:"generated_at"`
	OverallVerificationStatus ReviewVerificationStatus            `json:"overall_verification_status"`
	Verdict                   ReviewVerdict                       `json:"verdict"`
	Summary                   string                              `json:"summary"`
	SuggestedFindings         []ReviewReportModelSuggestedFinding `json:"suggested_findings"`
	CoverageGaps              []ReviewReportModelCoverageGap      `json:"coverage_gaps"`
	ProbeSummaries            []ReviewProbeSummary                `json:"probe_summaries"`
	ScopeCoverage             *ReviewReportScopeCoverage          `json:"scope_coverage"`
}

// ReviewReportModelSuggestedFinding は provider が提案する finding DTO である。
type ReviewReportModelSuggestedFinding struct {
	ID                   string                         `json:"id"`
	Severity             ReviewReportModelSeverity      `json:"severity"`
	Status               ReviewReportModelFindingStatus `json:"status"`
	Confidence           ReviewReportModelConfidence    `json:"confidence"`
	Title                string                         `json:"title"`
	AffectedBehavior     string                         `json:"affected_behavior"`
	CausalChain          string                         `json:"causal_chain"`
	Location             *ReviewReportModelLocation     `json:"location,omitempty"`
	EvidenceRefs         []ReviewEvidenceRef            `json:"evidence_refs"`
	Reproduction         *string                        `json:"reproduction,omitempty"`
	RemediationDirection string                         `json:"remediation_direction"`
}

// ReviewReportModelLocation は finding の主な位置を表す。
type ReviewReportModelLocation struct {
	Path      string `json:"path"`
	LineStart int    `json:"line_start"`
	LineEnd   int    `json:"line_end"`
}

// ReviewReportModelCoverageGap は finding ではない coverage gap を表す。
type ReviewReportModelCoverageGap struct {
	Surface          string                     `json:"surface"`
	Reason           ReviewReportModelGapReason `json:"reason"`
	RecommendedCheck string                     `json:"recommended_check"`
}

// ReviewReportModelSeverity は provider DTO の P-level severity である。
type ReviewReportModelSeverity string

const (
	ReviewReportModelSeverityP0 ReviewReportModelSeverity = "P0"
	ReviewReportModelSeverityP1 ReviewReportModelSeverity = "P1"
	ReviewReportModelSeverityP2 ReviewReportModelSeverity = "P2"
	ReviewReportModelSeverityP3 ReviewReportModelSeverity = "P3"
)

// ReviewReportModelFindingStatus は provider DTO の finding confidence status である。
type ReviewReportModelFindingStatus string

const (
	ReviewReportModelFindingConfirmed ReviewReportModelFindingStatus = "confirmed"
	ReviewReportModelFindingProbable  ReviewReportModelFindingStatus = "probable"
)

// ReviewReportModelConfidence は provider DTO の confidence enum である。
type ReviewReportModelConfidence string

const (
	ReviewReportModelConfidenceHigh   ReviewReportModelConfidence = "high"
	ReviewReportModelConfidenceMedium ReviewReportModelConfidence = "medium"
	ReviewReportModelConfidenceLow    ReviewReportModelConfidence = "low"
)

// ReviewReportModelGapReason は coverage gap の理由 enum である。
type ReviewReportModelGapReason string

const (
	ReviewReportModelGapEnvironmentBlocked ReviewReportModelGapReason = "environment_blocked"
	ReviewReportModelGapMissingEvidence    ReviewReportModelGapReason = "missing_evidence"
	ReviewReportModelGapNotExercised       ReviewReportModelGapReason = "not_exercised"
)

type reviewReportModelOutputJSON struct {
	SchemaVersion             *string                              `json:"schema_version"`
	TargetKind                *domain.TargetKind                   `json:"target_kind"`
	CustomInstructions        string                               `json:"custom_instructions,omitempty"`
	GeneratedAt               *time.Time                           `json:"generated_at"`
	OverallVerificationStatus *ReviewVerificationStatus            `json:"overall_verification_status"`
	Verdict                   *ReviewVerdict                       `json:"verdict"`
	Summary                   *string                              `json:"summary"`
	SuggestedFindings         *[]ReviewReportModelSuggestedFinding `json:"suggested_findings"`
	CoverageGaps              *[]ReviewReportModelCoverageGap      `json:"coverage_gaps"`
	ProbeSummaries            *[]ReviewProbeSummary                `json:"probe_summaries"`
	ScopeCoverage             *ReviewReportScopeCoverage           `json:"scope_coverage"`
}

// IsReviewReportModelOutputJSON は raw JSON が provider-output DTO schema を名乗るかを返す。
func IsReviewReportModelOutputJSON(data []byte) bool {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return bytes.Contains(data, []byte(ReviewReportModelSchemaVersionV2))
	}
	raw, ok := fields["schema_version"]
	if !ok {
		return false
	}
	var schemaVersion string
	if err := json.Unmarshal(raw, &schemaVersion); err != nil {
		return false
	}
	return schemaVersion == ReviewReportModelSchemaVersionV2
}

// DecodeReviewReportModelOutputStrictJSON は provider-output DTO を strict JSON として decode する。
func DecodeReviewReportModelOutputStrictJSON(data []byte) (ReviewReportModelOutput, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var dto reviewReportModelOutputJSON
	if err := decoder.Decode(&dto); err != nil {
		return ReviewReportModelOutput{}, fmt.Errorf("decode review report model output: %w", err)
	}
	var trailing json.RawMessage
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err != nil {
			return ReviewReportModelOutput{}, fmt.Errorf("review report model output must contain a single JSON value: %w", err)
		}
		return ReviewReportModelOutput{}, errors.New("review report model output must contain a single JSON value")
	}
	model, err := reviewReportModelOutputFromJSON(dto)
	if err != nil {
		return ReviewReportModelOutput{}, err
	}
	if err := ValidateReviewReportModelOutput(model); err != nil {
		return ReviewReportModelOutput{}, err
	}
	return model, nil
}

// ValidateReviewReportModelOutput は provider-output DTO の required fields と enum を検証する。
func ValidateReviewReportModelOutput(model ReviewReportModelOutput) error {
	if model.SchemaVersion != ReviewReportModelSchemaVersionV2 {
		return fmt.Errorf("schema_version must be %q: got %q", ReviewReportModelSchemaVersionV2, model.SchemaVersion)
	}
	if model.TargetKind != domain.TargetCurrentChanges {
		return fmt.Errorf("target_kind must be %q: got %q", domain.TargetCurrentChanges, model.TargetKind)
	}
	if model.GeneratedAt.IsZero() {
		return errors.New("generated_at must be non-zero")
	}
	if !isKnownReviewVerificationStatus(model.OverallVerificationStatus) {
		return fmt.Errorf("overall_verification_status must be known enum value: got %q", model.OverallVerificationStatus)
	}
	if !isKnownReviewVerdict(model.Verdict) {
		return fmt.Errorf("verdict must be known enum value: got %q", model.Verdict)
	}
	if model.ScopeCoverage == nil {
		return errors.New("scope_coverage is required")
	}
	if err := validateReviewReportScopeCoverageShape("scope_coverage", model.ScopeCoverage); err != nil {
		return err
	}
	if _, err := validateProbeSummaries(model.ProbeSummaries); err != nil {
		return err
	}
	for i, finding := range model.SuggestedFindings {
		if err := validateReviewReportModelSuggestedFinding(i, finding); err != nil {
			return err
		}
	}
	for i, gap := range model.CoverageGaps {
		if err := validateReviewReportModelCoverageGap(i, gap); err != nil {
			return err
		}
	}
	if err := validateReviewReportModelCoverageGapsAgainstScopeCoverage(model.CoverageGaps, model.ScopeCoverage); err != nil {
		return err
	}
	if model.Verdict == ReviewVerdictClean && len(model.CoverageGaps) > 0 {
		return fmt.Errorf("verdict %q requires coverage_gaps to be empty", ReviewVerdictClean)
	}
	return nil
}

// ToReviewReport は provider-output DTO を final artifact 用の review_report.v2 に変換する。
func (m ReviewReportModelOutput) ToReviewReport() ReviewReport {
	report := ReviewReport{
		SchemaVersion:             ReviewReportSchemaVersionV2,
		TargetKind:                m.TargetKind,
		CustomInstructions:        m.CustomInstructions,
		GeneratedAt:               m.GeneratedAt,
		OverallVerificationStatus: m.OverallVerificationStatus,
		Verdict:                   m.Verdict,
		Summary:                   m.Summary,
		ProbeSummaries:            CopyReviewProbeSummaries(m.ProbeSummaries),
		ScopeCoverage:             copyReviewReportScopeCoverage(m.ScopeCoverage),
	}
	for _, finding := range m.SuggestedFindings {
		report.RootCauseGroups = append(report.RootCauseGroups, rootCauseGroupFromModelFinding(finding))
	}
	for _, gap := range m.CoverageGaps {
		report.UnverifiedSurfaces = append(report.UnverifiedSurfaces, ReviewSurfaceCoverage{
			SurfaceID: gap.Surface,
			Summary:   strings.TrimSpace(string(gap.Reason) + ": " + gap.RecommendedCheck),
		})
	}
	return report
}

func reviewReportModelOutputFromJSON(dto reviewReportModelOutputJSON) (ReviewReportModelOutput, error) {
	missing := []string{}
	if dto.SchemaVersion == nil {
		missing = append(missing, "schema_version")
	}
	if dto.TargetKind == nil {
		missing = append(missing, "target_kind")
	}
	if dto.GeneratedAt == nil {
		missing = append(missing, "generated_at")
	}
	if dto.OverallVerificationStatus == nil {
		missing = append(missing, "overall_verification_status")
	}
	if dto.Verdict == nil {
		missing = append(missing, "verdict")
	}
	if dto.Summary == nil {
		missing = append(missing, "summary")
	}
	if dto.SuggestedFindings == nil {
		missing = append(missing, "suggested_findings")
	}
	if dto.CoverageGaps == nil {
		missing = append(missing, "coverage_gaps")
	}
	if dto.ProbeSummaries == nil {
		missing = append(missing, "probe_summaries")
	}
	if dto.ScopeCoverage == nil {
		missing = append(missing, "scope_coverage")
	}
	if len(missing) > 0 {
		return ReviewReportModelOutput{}, fmt.Errorf("review report model output missing keys: %s", strings.Join(missing, ", "))
	}
	return ReviewReportModelOutput{
		SchemaVersion:             *dto.SchemaVersion,
		TargetKind:                *dto.TargetKind,
		CustomInstructions:        dto.CustomInstructions,
		GeneratedAt:               *dto.GeneratedAt,
		OverallVerificationStatus: *dto.OverallVerificationStatus,
		Verdict:                   *dto.Verdict,
		Summary:                   *dto.Summary,
		SuggestedFindings:         append([]ReviewReportModelSuggestedFinding(nil), (*dto.SuggestedFindings)...),
		CoverageGaps:              append([]ReviewReportModelCoverageGap(nil), (*dto.CoverageGaps)...),
		ProbeSummaries:            CopyReviewProbeSummaries(*dto.ProbeSummaries),
		ScopeCoverage:             copyReviewReportScopeCoverage(dto.ScopeCoverage),
	}, nil
}

func validateReviewReportModelSuggestedFinding(index int, finding ReviewReportModelSuggestedFinding) error {
	field := fmt.Sprintf("suggested_findings[%d]", index)
	if _, err := validateRequiredReportID(field+".id", finding.ID); err != nil {
		return err
	}
	if !isKnownReviewReportModelSeverity(finding.Severity) {
		return fmt.Errorf("%s.severity must be one of %q, %q, %q, %q: got %q", field, ReviewReportModelSeverityP0, ReviewReportModelSeverityP1, ReviewReportModelSeverityP2, ReviewReportModelSeverityP3, finding.Severity)
	}
	if !isKnownReviewReportModelFindingStatus(finding.Status) {
		return fmt.Errorf("%s.status must be one of %q, %q: got %q", field, ReviewReportModelFindingConfirmed, ReviewReportModelFindingProbable, finding.Status)
	}
	if !isKnownReviewReportModelConfidence(finding.Confidence) {
		return fmt.Errorf("%s.confidence must be one of %q, %q, %q: got %q", field, ReviewReportModelConfidenceHigh, ReviewReportModelConfidenceMedium, ReviewReportModelConfidenceLow, finding.Confidence)
	}
	if strings.TrimSpace(finding.Title) == "" {
		return fmt.Errorf("%s.title must be non-empty", field)
	}
	if strings.TrimSpace(finding.AffectedBehavior) == "" {
		return fmt.Errorf("%s.affected_behavior must be non-empty", field)
	}
	if strings.TrimSpace(finding.CausalChain) == "" {
		return fmt.Errorf("%s.causal_chain must be non-empty", field)
	}
	if strings.TrimSpace(finding.RemediationDirection) == "" {
		return fmt.Errorf("%s.remediation_direction must be non-empty", field)
	}
	if len(finding.EvidenceRefs) == 0 {
		return fmt.Errorf("%s.evidence_refs must contain at least one evidence ref", field)
	}
	if err := validateEvidenceRefsShape(field+".evidence_refs", finding.EvidenceRefs); err != nil {
		return err
	}
	if finding.Location != nil {
		if err := validateReviewReportModelLocation(field+".location", *finding.Location); err != nil {
			return err
		}
	}
	return nil
}

func validateReviewReportModelLocation(field string, location ReviewReportModelLocation) error {
	path, err := validateOptionalEvidencePath(field+".path", location.Path)
	if err != nil {
		return err
	}
	if path == "" {
		return fmt.Errorf("%s.path must be non-empty", field)
	}
	if err := validateEvidencePath(field+".path", path); err != nil {
		return err
	}
	if location.LineStart < 1 {
		return fmt.Errorf("%s.line_start must be >= 1: got %d", field, location.LineStart)
	}
	if location.LineEnd < location.LineStart {
		return fmt.Errorf("%s.line_end must be >= line_start: got %d < %d", field, location.LineEnd, location.LineStart)
	}
	return nil
}

func validateReviewReportModelCoverageGap(index int, gap ReviewReportModelCoverageGap) error {
	field := fmt.Sprintf("coverage_gaps[%d]", index)
	if _, err := validateRequiredReportID(field+".surface", gap.Surface); err != nil {
		return err
	}
	if !isKnownReviewReportModelGapReason(gap.Reason) {
		return fmt.Errorf("%s.reason must be one of %q, %q, %q: got %q", field, ReviewReportModelGapEnvironmentBlocked, ReviewReportModelGapMissingEvidence, ReviewReportModelGapNotExercised, gap.Reason)
	}
	if strings.TrimSpace(gap.RecommendedCheck) == "" {
		return fmt.Errorf("%s.recommended_check must be non-empty", field)
	}
	return nil
}

func validateReviewReportModelCoverageGapsAgainstScopeCoverage(gaps []ReviewReportModelCoverageGap, coverage *ReviewReportScopeCoverage) error {
	if len(gaps) == 0 {
		return nil
	}
	surfaces := make(map[string]ReviewReportImpactSurfaceStatus, len(coverage.ReviewedImpactSurfaces))
	for _, surface := range coverage.ReviewedImpactSurfaces {
		surfaces[surface.SurfaceID] = surface.Status
	}
	for i, gap := range gaps {
		status, exists := surfaces[gap.Surface]
		if !exists {
			return fmt.Errorf("coverage_gaps[%d].surface %q must reference scope_coverage.reviewed_impact_surfaces[].surface_id", i, gap.Surface)
		}
		switch status {
		case ReviewReportImpactSurfaceUnverified, ReviewReportImpactSurfaceResidualRisk:
		default:
			return fmt.Errorf("coverage_gaps[%d].surface %q requires scope_coverage.reviewed_impact_surfaces status %q or %q: got %q", i, gap.Surface, ReviewReportImpactSurfaceUnverified, ReviewReportImpactSurfaceResidualRisk, status)
		}
	}
	return nil
}

func rootCauseGroupFromModelFinding(finding ReviewReportModelSuggestedFinding) ReviewRootCauseGroup {
	findingID := strings.TrimSpace(finding.ID)
	summaryParts := []string{
		"affected_behavior: " + strings.TrimSpace(finding.AffectedBehavior),
		"causal_chain: " + strings.TrimSpace(finding.CausalChain),
		"confidence: " + string(finding.Confidence),
	}
	if finding.Reproduction != nil && strings.TrimSpace(*finding.Reproduction) != "" {
		summaryParts = append(summaryParts, "reproduction: "+strings.TrimSpace(*finding.Reproduction))
	}
	return ReviewRootCauseGroup{
		ID:                 "group-" + findingID,
		Title:              strings.TrimSpace(finding.Title),
		Summary:            strings.Join(summaryParts, "\n"),
		Severity:           reviewGroupSeverityFromModelSeverity(finding.Severity),
		VerificationStatus: reviewVerificationStatusFromModelFindingStatus(finding.Status),
		FixStrategy:        strings.TrimSpace(finding.RemediationDirection),
		VerificationPlan:   []string{"Verify remediation for " + findingID + "."},
		Findings: []ReviewFinding{
			{
				ID:           findingID,
				Title:        strings.TrimSpace(finding.Title),
				Summary:      strings.Join(summaryParts, "\n"),
				EvidenceRefs: append([]ReviewEvidenceRef(nil), finding.EvidenceRefs...),
			},
		},
	}
}

func reviewGroupSeverityFromModelSeverity(severity ReviewReportModelSeverity) ReviewGroupSeverity {
	switch severity {
	case ReviewReportModelSeverityP0:
		return ReviewGroupSeverityCritical
	case ReviewReportModelSeverityP1:
		return ReviewGroupSeverityHigh
	case ReviewReportModelSeverityP2:
		return ReviewGroupSeverityMedium
	default:
		return ReviewGroupSeverityLow
	}
}

func reviewVerificationStatusFromModelFindingStatus(status ReviewReportModelFindingStatus) ReviewVerificationStatus {
	switch status {
	case ReviewReportModelFindingConfirmed:
		return ReviewVerificationVerified
	case ReviewReportModelFindingProbable:
		return ReviewVerificationPartiallyVerified
	default:
		return ReviewVerificationUnverified
	}
}

func isKnownReviewReportModelSeverity(severity ReviewReportModelSeverity) bool {
	switch severity {
	case ReviewReportModelSeverityP0, ReviewReportModelSeverityP1, ReviewReportModelSeverityP2, ReviewReportModelSeverityP3:
		return true
	default:
		return false
	}
}

func isKnownReviewReportModelFindingStatus(status ReviewReportModelFindingStatus) bool {
	switch status {
	case ReviewReportModelFindingConfirmed, ReviewReportModelFindingProbable:
		return true
	default:
		return false
	}
}

func isKnownReviewReportModelConfidence(confidence ReviewReportModelConfidence) bool {
	switch confidence {
	case ReviewReportModelConfidenceHigh, ReviewReportModelConfidenceMedium, ReviewReportModelConfidenceLow:
		return true
	default:
		return false
	}
}

func isKnownReviewReportModelGapReason(reason ReviewReportModelGapReason) bool {
	switch reason {
	case ReviewReportModelGapEnvironmentBlocked, ReviewReportModelGapMissingEvidence, ReviewReportModelGapNotExercised:
		return true
	default:
		return false
	}
}

func copyReviewReportScopeCoverage(src *ReviewReportScopeCoverage) *ReviewReportScopeCoverage {
	if src == nil {
		return nil
	}
	dst := &ReviewReportScopeCoverage{
		ReviewedImpactSurfaces:    append([]ReviewReportImpactSurfaceCoverage(nil), src.ReviewedImpactSurfaces...),
		ReviewedCandidateRisks:    append([]ReviewReportCandidateRiskCoverage(nil), src.ReviewedCandidateRisks...),
		NewFindingsFromReportPass: append([]ReviewReportPassFindingCoverage(nil), src.NewFindingsFromReportPass...),
	}
	for i := range dst.ReviewedImpactSurfaces {
		dst.ReviewedImpactSurfaces[i].EvidenceRefs = append([]ReviewEvidenceRef(nil), src.ReviewedImpactSurfaces[i].EvidenceRefs...)
		dst.ReviewedImpactSurfaces[i].FindingIDs = append([]string(nil), src.ReviewedImpactSurfaces[i].FindingIDs...)
	}
	for i := range dst.ReviewedCandidateRisks {
		dst.ReviewedCandidateRisks[i].EvidenceRefs = append([]ReviewEvidenceRef(nil), src.ReviewedCandidateRisks[i].EvidenceRefs...)
		dst.ReviewedCandidateRisks[i].FindingIDs = append([]string(nil), src.ReviewedCandidateRisks[i].FindingIDs...)
	}
	for i := range dst.NewFindingsFromReportPass {
		dst.NewFindingsFromReportPass[i].EvidenceRefs = append([]ReviewEvidenceRef(nil), src.NewFindingsFromReportPass[i].EvidenceRefs...)
		dst.NewFindingsFromReportPass[i].FindingIDs = append([]string(nil), src.NewFindingsFromReportPass[i].FindingIDs...)
	}
	return dst
}
