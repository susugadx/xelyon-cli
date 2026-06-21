package report

import (
	"strings"

	"github.com/susugadx/xelyon-cli/internal/review/domain"
)

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
		case domain.ReviewProbeBlocked, domain.ReviewProbeTimedOut, domain.ReviewProbeMutatedWorktree:
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
