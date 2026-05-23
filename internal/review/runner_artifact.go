package review

import "fmt"

// ReviewRunArtifactWriter は /review runner の debug artifact 保存境界を表す。
// nil の場合、runner は artifact を完全に無効化する。
type ReviewRunArtifactWriter interface {
	WriteReviewRunArtifact(name string, content []byte) error
}

func (r *ReviewRunner) saveReviewRunTextArtifact(name, content string, redactor reviewRunnerPromptRedactor) {
	r.saveReviewRunArtifact(name, []byte(redactor.redactText(content)))
}

func (r *ReviewRunner) saveReviewRunJSONArtifact(name string, value any, redactor reviewRunnerPromptRedactor) {
	data, err := marshalReviewJSONIndent(value)
	if err != nil {
		r.warnReviewRunArtifact(name, fmt.Errorf("marshal artifact: %w", err))
		return
	}
	r.saveReviewRunTextArtifact(name, string(data), redactor)
}

func (r *ReviewRunner) saveReviewRunArtifact(name string, content []byte) {
	if r == nil || r.artifactWriter == nil {
		return
	}
	if err := r.artifactWriter.WriteReviewRunArtifact(name, content); err != nil {
		r.warnReviewRunArtifact(name, err)
	}
}

func (r *ReviewRunner) warnReviewRunArtifact(name string, err error) {
	if r == nil || r.artifactWarningWriter == nil || err == nil {
		return
	}
	_, _ = fmt.Fprintf(r.artifactWarningWriter, "Warning: failed to save review artifact %s: %v\n", name, err)
}
