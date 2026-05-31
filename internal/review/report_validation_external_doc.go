package review

import reviewanalysis "github.com/susugadx/xelyon-cli/internal/review/analysis"

func validateReviewReportExternalDocRefsAgainstEvidence(report ReviewReport, bundle ReviewEvidenceBundle) error {
	return reviewanalysis.ValidateReportExternalDocRefs(report, bundle.WebSearchEvidence.ExternalDocs)
}

func validateReviewSaturationExternalDocRefsAgainstEvidence(check ReviewSaturationCheck, bundle ReviewEvidenceBundle) error {
	return reviewanalysis.ValidateSaturationExternalDocRefs(check, bundle.WebSearchEvidence.ExternalDocs)
}
