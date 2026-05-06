package review

import (
	"bytes"
	"encoding/json"
)

// RenderReviewEvidenceJSON は ReviewEvidenceBundle を LLM 入力 JSON に変換する。
// 収集済み evidence の render 境界であり、git や file system からの収集は行わない。
func RenderReviewEvidenceJSON(bundle ReviewEvidenceBundle) ([]byte, error) {
	return marshalReviewEvidenceJSONIndent(BuildReviewEvidenceModelInput(bundle))
}

func marshalReviewEvidenceJSONIndent(value any) ([]byte, error) {
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(value); err != nil {
		return nil, err
	}
	return bytes.TrimSuffix(buf.Bytes(), []byte("\n")), nil
}
