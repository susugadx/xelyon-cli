package probe

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
)

// DecodeReviewProbePlanJSON は strict JSON として ReviewProbePlan を decode して検証する。
func DecodeReviewProbePlanJSON(data []byte) (ReviewProbePlan, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()

	var plan ReviewProbePlan
	if err := decoder.Decode(&plan); err != nil {
		return ReviewProbePlan{}, fmt.Errorf("decode review probe plan: %w", err)
	}

	var trailing json.RawMessage
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err != nil {
			return ReviewProbePlan{}, fmt.Errorf("review probe plan must contain a single JSON value: %w", err)
		}
		return ReviewProbePlan{}, fmt.Errorf("review probe plan must contain a single JSON value")
	}

	if err := ValidateReviewProbePlan(plan); err != nil {
		return ReviewProbePlan{}, err
	}
	return plan, nil
}
