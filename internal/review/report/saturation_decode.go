package report

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
)

// DecodeReviewSaturationCheckJSON は strict JSON として saturation check を decode して検証する。
func DecodeReviewSaturationCheckJSON(data []byte, plan PlanScope, finalizedReport ReviewReport) (ReviewSaturationCheck, error) {
	check, err := decodeReviewSaturationCheckStrictJSON(data)
	if err != nil {
		return ReviewSaturationCheck{}, err
	}
	if err := ValidateReviewSaturationCheck(check, plan, finalizedReport); err != nil {
		return ReviewSaturationCheck{}, err
	}
	return check, nil
}

func decodeReviewSaturationCheckStrictJSON(data []byte) (ReviewSaturationCheck, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()

	var check ReviewSaturationCheck
	if err := decoder.Decode(&check); err != nil {
		return ReviewSaturationCheck{}, fmt.Errorf("decode review saturation check: %w", err)
	}

	var trailing json.RawMessage
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err != nil {
			return ReviewSaturationCheck{}, fmt.Errorf("review saturation check must contain a single JSON value: %w", err)
		}
		return ReviewSaturationCheck{}, fmt.Errorf("review saturation check must contain a single JSON value")
	}

	return check, nil
}
