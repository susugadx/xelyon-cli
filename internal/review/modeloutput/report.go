package modeloutput

import (
	"fmt"

	reviewanalysis "github.com/susugadx/xelyon-cli/internal/review/analysis"
	"github.com/susugadx/xelyon-cli/internal/review/externaldoc"
	reviewprobe "github.com/susugadx/xelyon-cli/internal/review/probe"
	reviewreport "github.com/susugadx/xelyon-cli/internal/review/report"
)

// Redactor は final report に注入する runner-owned probe summary の表示用 redaction 境界。
type Redactor interface {
	RedactText(string) string
	RedactTexts([]string) []string
	RedactPath(string) string
	RedactPaths([]string) []string
}

// ReportModelOutputInput は model の report JSON 出力を final report に確定する入力。
type ReportModelOutputInput struct {
	Content               string
	Plan                  reviewprobe.ReviewProbePlan
	TrustedProbeSummaries []reviewreport.ReviewProbeSummary
	Redactor              Redactor
	ExternalDocs          []externaldoc.Evidence
}

// ReportFinalizationInput は decode 済み report を runner final report に確定する入力。
type ReportFinalizationInput struct {
	Report                reviewreport.ReviewReport
	Plan                  reviewprobe.ReviewProbePlan
	TrustedProbeSummaries []reviewreport.ReviewProbeSummary
	Redactor              Redactor
	ExternalDocs          []externaldoc.Evidence
}

// FinalizeReportModelOutput は model の raw report JSON を strict decode し、runner final report へ確定する。
func FinalizeReportModelOutput(input ReportModelOutputInput) (reviewreport.ReviewReport, error) {
	data := []byte(input.Content)
	var report reviewreport.ReviewReport
	if reviewreport.IsReviewReportModelOutputJSON(data) {
		model, err := reviewreport.DecodeReviewReportModelOutputStrictJSON(data)
		if err != nil {
			return reviewreport.ReviewReport{}, fmt.Errorf("review runner decode report model: %w", err)
		}
		report = model.ToReviewReport()
	} else {
		var err error
		report, err = reviewreport.DecodeReviewReportModelStrictJSON(data)
		if err != nil {
			return reviewreport.ReviewReport{}, fmt.Errorf("review runner decode report: %w", err)
		}
	}
	return FinalizeReport(ReportFinalizationInput{
		Report:                report,
		Plan:                  input.Plan,
		TrustedProbeSummaries: input.TrustedProbeSummaries,
		Redactor:              input.Redactor,
		ExternalDocs:          input.ExternalDocs,
	})
}

// FinalizeReport は decode 済み report に runner-owned trusted probe summary と computed summary を注入する。
func FinalizeReport(input ReportFinalizationInput) (reviewreport.ReviewReport, error) {
	report := input.Report
	redactor := normalizeRedactor(input.Redactor)

	// LLM が返す probe_summaries は信頼元にしない。runner が probe results から作った
	// raw trusted summaries を内部 audit/debug 契約として保ち、final report には redacted copy だけを注入する。
	probeSummaries := reviewreport.CopyReviewProbeSummaries(input.TrustedProbeSummaries)
	reviewreport.CanonicalizeReviewProbeSummaryMutationOutcomes(probeSummaries)
	if len(probeSummaries) == 0 {
		report.ProbeSummaries = nil
	} else {
		report.ProbeSummaries = redactReviewProbeSummaries(probeSummaries, redactor)
	}

	report = reviewreport.NormalizeReviewReportForTrustedProbeOutcomes(report)
	if err := reviewreport.ValidateReviewReportAgainstPlanScope(report, reviewanalysis.PlanScopeFromProbePlan(input.Plan), input.TrustedProbeSummaries); err != nil {
		return reviewreport.ReviewReport{}, fmt.Errorf("review runner finalize report: %w", err)
	}
	if err := reviewanalysis.ValidateReportExternalDocRefs(report, input.ExternalDocs); err != nil {
		return reviewreport.ReviewReport{}, fmt.Errorf("review runner finalize report: %w", err)
	}

	computedSummary := reviewreport.ComputeReviewReportComputedSummary(report, probeSummaries)
	report.ComputedSummary = &computedSummary
	return report, nil
}

func redactReviewProbeSummaries(summaries []reviewreport.ReviewProbeSummary, redactor Redactor) []reviewreport.ReviewProbeSummary {
	redacted := make([]reviewreport.ReviewProbeSummary, 0, len(summaries))
	for _, summary := range summaries {
		redacted = append(redacted, reviewreport.ReviewProbeSummary{
			ProbeID:         summary.ProbeID,
			Mode:            summary.Mode,
			Status:          summary.Status,
			MutatedWorktree: summary.MutatedWorktree,
			MutatedFiles:    redactor.RedactPaths(summary.MutatedFiles),
			OutputTruncated: summary.OutputTruncated,
			Error:           redactor.RedactText(summary.Error),
			Commands:        redactReviewProbeCommandSummaries(summary.Commands, redactor),
		})
	}
	return redacted
}

func redactReviewProbeCommandSummaries(summaries []reviewreport.ReviewProbeCommandSummary, redactor Redactor) []reviewreport.ReviewProbeCommandSummary {
	redacted := make([]reviewreport.ReviewProbeCommandSummary, 0, len(summaries))
	for _, summary := range summaries {
		redacted = append(redacted, reviewreport.ReviewProbeCommandSummary{
			Command:         redactor.RedactText(summary.Command),
			Args:            redactor.RedactTexts(summary.Args),
			WorkDir:         redactor.RedactPath(summary.WorkDir),
			Status:          summary.Status,
			ExitCode:        summary.ExitCode,
			OutputTruncated: summary.OutputTruncated,
			Error:           redactor.RedactText(summary.Error),
			DurationMs:      summary.DurationMs,
		})
	}
	return redacted
}

func normalizeRedactor(redactor Redactor) Redactor {
	if redactor == nil {
		return noopRedactor{}
	}
	return redactor
}

type noopRedactor struct{}

func (noopRedactor) RedactText(text string) string {
	return text
}

func (noopRedactor) RedactTexts(values []string) []string {
	if len(values) == 0 {
		return []string{}
	}
	return append([]string(nil), values...)
}

func (noopRedactor) RedactPath(path string) string {
	return path
}

func (noopRedactor) RedactPaths(paths []string) []string {
	if len(paths) == 0 {
		return []string{}
	}
	return append([]string(nil), paths...)
}
