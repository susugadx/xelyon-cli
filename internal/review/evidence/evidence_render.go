package evidence

// RenderReviewEvidenceJSON は ReviewEvidenceBundle を LLM 入力 JSON に変換する。
// 収集済み evidence の render 境界であり、git や file system からの収集は行わない。
func RenderReviewEvidenceJSON(bundle ReviewEvidenceBundle) ([]byte, error) {
	return marshalReviewJSONIndent(BuildReviewEvidenceModelInput(bundle))
}
