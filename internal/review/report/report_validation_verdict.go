package report

import (
	"fmt"
	"strings"
)

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
