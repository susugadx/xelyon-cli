package artifact

import "fmt"

// WriteReviewRunArtifact は artifact をメモリに保持する。
func (w *BufferedReviewRunArtifactWriter) WriteReviewRunArtifact(name string, content []byte) error {
	if w == nil {
		return fmt.Errorf("artifact writer is nil")
	}
	if err := validateReviewRunArtifactName(name); err != nil {
		return err
	}
	copied := append([]byte(nil), content...)
	w.artifacts = append(w.artifacts, bufferedReviewRunArtifact{
		name:    name,
		content: copied,
	})
	return nil
}

// Len は保持中の artifact 数を返す。
func (w *BufferedReviewRunArtifactWriter) Len() int {
	if w == nil {
		return 0
	}
	return len(w.artifacts)
}

// FlushTo は保持中の artifact を指定 writer へ登録順で書き出す。
func (w *BufferedReviewRunArtifactWriter) FlushTo(dst ReviewRunArtifactWriter) error {
	if w == nil {
		return fmt.Errorf("artifact writer is nil")
	}
	if dst == nil {
		return fmt.Errorf("artifact destination writer is nil")
	}
	for _, artifact := range w.artifacts {
		if err := dst.WriteReviewRunArtifact(artifact.name, artifact.content); err != nil {
			return err
		}
	}
	return nil
}
