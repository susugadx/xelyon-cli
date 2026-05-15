package review

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
)

// DecodeReviewReportJSON は strict JSON として ReviewReport を decode して検証する。
func DecodeReviewReportJSON(data []byte) (ReviewReport, error) {
	report, err := decodeReviewReportStrictJSON(data)
	if err != nil {
		return ReviewReport{}, err
	}

	if err := ValidateReviewReport(report); err != nil {
		return ReviewReport{}, err
	}
	return report, nil
}

func decodeReviewReportStrictJSON(data []byte) (ReviewReport, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()

	var report ReviewReport
	if err := decoder.Decode(&report); err != nil {
		return ReviewReport{}, fmt.Errorf("decode review report: %w", err)
	}

	var trailing json.RawMessage
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err != nil {
			return ReviewReport{}, fmt.Errorf("review report must contain a single JSON value: %w", err)
		}
		return ReviewReport{}, fmt.Errorf("review report must contain a single JSON value")
	}

	return report, nil
}

func decodeReviewReportModelStrictJSON(data []byte) (ReviewReport, error) {
	if err := rejectModelComputedSummaryInReviewReportJSON(data); err != nil {
		return ReviewReport{}, err
	}
	return decodeReviewReportStrictJSON(data)
}

func rejectModelComputedSummaryInReviewReportJSON(data []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(data))

	var fields map[string]json.RawMessage
	if err := decoder.Decode(&fields); err != nil {
		return nil
	}
	if _, exists := fields["computed_summary"]; exists {
		return fmt.Errorf("computed_summary is runner-computed and must not be supplied by model output")
	}
	return nil
}
