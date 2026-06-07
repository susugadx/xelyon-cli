package review

import (
	"fmt"
	"strings"
)

type reviewStateSummaryInput struct {
	bundle          ReviewEvidenceBundle
	plan            ReviewProbePlan
	probeSummaries  []ReviewProbeSummary
	finalizedReport ReviewReport
	saturationCheck ReviewSaturationCheck
	phase           ReviewModelPhase
}

func buildReviewStateSummary(input reviewStateSummaryInput) ReviewStateSummary {
	summary := ReviewStateSummary{
		Target: string(TargetCurrentChanges),
	}
	summary.ChangedFiles = reviewStateChangedFiles(input.bundle)
	summary.ImpactSurfaces = reviewStateImpactSurfaces(input.plan)
	summary.CandidateRisks = reviewStateCandidateRisks(input.plan)
	summary.UnresolvedRisks = reviewStateUnresolvedRisks(input.plan, input.finalizedReport, input.saturationCheck)
	summary.ConfirmedFindings = reviewStateConfirmedFindings(input.finalizedReport)
	summary.FindingEvidenceRefs = reviewStateFindingEvidenceRefs(input.finalizedReport)
	summary.DismissedRisks = reviewStateDismissedRisks(input.finalizedReport)
	summary.ScopeCoverage = reviewStateScopeCoverage(input.finalizedReport)
	summary.ExternalEvidence = reviewStateExternalEvidence(input.bundle.WebSearchEvidence)
	summary.LatestReportStatus = reviewStateLatestReportStatus(input.finalizedReport)
	summary.SaturationStatus = reviewStateSaturationStatus(input.saturationCheck)
	summary.NextProbeFocus = reviewStateNextProbeFocus(input.plan, input.saturationCheck)
	summary.AbsorbedIntermediateRefs = reviewStateAbsorbedIntermediateRefs(input.phase, input.probeSummaries, input.finalizedReport)
	return summary
}

func reviewStateChangedFiles(bundle ReviewEvidenceBundle) []string {
	values := make([]string, 0, len(bundle.ChangedFiles)+len(bundle.UntrackedFiles))
	for _, file := range bundle.ChangedFiles {
		path := strings.TrimSpace(file.Path)
		if path == "" {
			continue
		}
		status := strings.TrimSpace(file.Status)
		if status == "" {
			status = "changed"
		}
		values = append(values, fmt.Sprintf("%s %s", status, path))
	}
	for _, file := range bundle.UntrackedFiles {
		path := strings.TrimSpace(file.Path)
		if path != "" {
			values = append(values, "untracked "+path)
		}
	}
	return values
}

func reviewStateImpactSurfaces(plan ReviewProbePlan) []string {
	values := make([]string, 0, len(plan.ImpactSurfaces))
	for _, surface := range plan.ImpactSurfaces {
		id := strings.TrimSpace(surface.ID)
		if id == "" {
			continue
		}
		values = append(values, fmt.Sprintf("%s status=%s category=%s summary=%s", id, surface.Status, surface.Category, oneLine(surface.Summary)))
	}
	return values
}

func reviewStateCandidateRisks(plan ReviewProbePlan) []string {
	values := make([]string, 0, len(plan.CandidateRisks))
	for _, risk := range plan.CandidateRisks {
		id := strings.TrimSpace(risk.ID)
		if id == "" {
			continue
		}
		values = append(values, fmt.Sprintf("%s status=%s severity=%s summary=%s", id, risk.Status, risk.Severity, oneLine(risk.Summary)))
	}
	return values
}

func reviewStateUnresolvedRisks(plan ReviewProbePlan, report ReviewReport, check ReviewSaturationCheck) []string {
	values := make([]string, 0)
	if report.ScopeCoverage != nil {
		for _, risk := range report.ScopeCoverage.ReviewedCandidateRisks {
			switch risk.Status {
			case ReviewReportCandidateRiskUnverified, ReviewReportCandidateRiskResidualRisk:
				values = append(values, fmt.Sprintf("%s status=%s summary=%s", risk.RiskID, risk.Status, oneLine(risk.Summary)))
			}
		}
	}
	if len(values) > 0 {
		return values
	}
	for _, risk := range plan.CandidateRisks {
		switch risk.Status {
		case ReviewProbeCandidateRiskNeedsProbe, ReviewProbeCandidateRiskUnverified:
			values = append(values, fmt.Sprintf("%s pass1_status=%s summary=%s", risk.ID, risk.Status, oneLine(risk.Summary)))
		}
	}
	for _, riskID := range check.MissingRiskIDs {
		if strings.TrimSpace(riskID) != "" {
			values = append(values, "saturation_missing "+riskID)
		}
	}
	return values
}

func reviewStateConfirmedFindings(report ReviewReport) []string {
	values := make([]string, 0)
	for _, group := range report.RootCauseGroups {
		for _, finding := range group.Findings {
			title := oneLine(finding.Title)
			if title == "" {
				title = oneLine(group.Title)
			}
			if title != "" {
				values = append(values, fmt.Sprintf("%s severity=%s", title, group.Severity))
			}
		}
	}
	return values
}

func reviewStateFindingEvidenceRefs(report ReviewReport) []string {
	values := make([]string, 0)
	for _, group := range report.RootCauseGroups {
		for _, finding := range group.Findings {
			for _, ref := range finding.EvidenceRefs {
				values = append(values, reviewStateEvidenceRefSummary(ref))
			}
		}
	}
	return values
}

func reviewStateDismissedRisks(report ReviewReport) []string {
	if report.ScopeCoverage == nil {
		return nil
	}
	values := make([]string, 0)
	for _, risk := range report.ScopeCoverage.ReviewedCandidateRisks {
		if risk.Status == ReviewReportCandidateRiskDismissed {
			values = append(values, fmt.Sprintf("%s dismissed summary=%s", risk.RiskID, oneLine(risk.Summary)))
		}
	}
	return values
}

func reviewStateScopeCoverage(report ReviewReport) []string {
	if report.ScopeCoverage == nil {
		return nil
	}
	values := make([]string, 0, len(report.ScopeCoverage.ReviewedImpactSurfaces)+len(report.ScopeCoverage.ReviewedCandidateRisks))
	for _, surface := range report.ScopeCoverage.ReviewedImpactSurfaces {
		values = append(values, fmt.Sprintf("surface %s status=%s", surface.SurfaceID, surface.Status))
	}
	for _, risk := range report.ScopeCoverage.ReviewedCandidateRisks {
		values = append(values, fmt.Sprintf("risk %s status=%s", risk.RiskID, risk.Status))
	}
	return values
}

func reviewStateExternalEvidence(evidence ReviewWebSearchEvidence) []string {
	if !evidence.Enabled {
		return nil
	}
	values := []string{
		fmt.Sprintf("queries=%d external_docs=%d truncated=%t inconclusive=%t", len(evidence.Queries), len(evidence.ExternalDocs), evidence.Truncated, evidence.Inconclusive),
	}
	for _, doc := range evidence.ExternalDocs {
		docID := strings.TrimSpace(doc.DocID)
		if docID == "" {
			continue
		}
		values = append(values, fmt.Sprintf("external_doc %s credibility=%s snippets=%d truncated=%t", docID, doc.SourceCredibility, len(doc.Snippets), doc.Truncated))
	}
	return values
}

func reviewStateLatestReportStatus(report ReviewReport) string {
	if strings.TrimSpace(report.SchemaVersion) == "" {
		return ""
	}
	findingCount := 0
	for _, group := range report.RootCauseGroups {
		findingCount += len(group.Findings)
	}
	return fmt.Sprintf("schema=%s verdict=%s findings=%d", report.SchemaVersion, report.Verdict, findingCount)
}

func reviewStateSaturationStatus(check ReviewSaturationCheck) string {
	if strings.TrimSpace(string(check.Status)) == "" {
		return ""
	}
	return fmt.Sprintf("status=%s missing_surfaces=%d missing_risks=%d", check.Status, len(check.MissingSurfaceIDs), len(check.MissingRiskIDs))
}

func reviewStateNextProbeFocus(plan ReviewProbePlan, check ReviewSaturationCheck) []string {
	values := make([]string, 0)
	for _, probe := range plan.Probes {
		if strings.TrimSpace(probe.ID) != "" {
			values = append(values, fmt.Sprintf("%s purpose=%s", probe.ID, oneLine(probe.Purpose)))
		}
	}
	if strings.TrimSpace(check.RevisionInstructions) != "" {
		values = append(values, "revision: "+oneLine(check.RevisionInstructions))
	}
	return values
}

func reviewStateAbsorbedIntermediateRefs(phase ReviewModelPhase, probeSummaries []ReviewProbeSummary, report ReviewReport) []string {
	values := make([]string, 0)
	switch phase {
	case ReviewModelPhaseSaturationCheck, ReviewModelPhaseReportRevision:
		if len(probeSummaries) > 0 {
			values = append(values, fmt.Sprintf("probe_result count=%d -> trusted_probe_summaries/latest_report", len(probeSummaries)))
		}
		if strings.TrimSpace(report.SchemaVersion) != "" {
			values = append(values, "report_draft -> latest_report")
		}
	}
	return values
}

func reviewStateEvidenceRefSummary(ref ReviewEvidenceRef) string {
	parts := []string{string(ref.Kind)}
	if ref.ProbeID != "" {
		parts = append(parts, "probe="+ref.ProbeID)
	}
	if ref.Path != "" {
		parts = append(parts, "path="+ref.Path)
	}
	if ref.DocID != "" {
		parts = append(parts, "doc="+ref.DocID)
	}
	if ref.SnippetID != "" {
		parts = append(parts, "snippet="+ref.SnippetID)
	}
	return strings.Join(parts, " ")
}

func oneLine(value string) string {
	value = strings.TrimSpace(value)
	value = strings.NewReplacer("\r", " ", "\n", " ", "\t", " ").Replace(value)
	if len(value) > 240 {
		value = value[:240]
	}
	return value
}
