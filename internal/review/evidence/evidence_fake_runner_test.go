package evidence

import (
	"context"
	"strings"
	"time"
)

type fakeReviewEvidenceRunner struct {
	outputs  map[string]string
	failures map[string]error
}

func (r fakeReviewEvidenceRunner) RunGit(_ context.Context, _ string, _ string, args []string, _ time.Duration, maxOutputBytes int64) (string, bool, error) {
	key := fakeReviewEvidenceGitKey(args...)
	if err := r.failures[key]; err != nil {
		return "", false, err
	}
	output := r.outputs[key]
	if maxOutputBytes <= 0 || int64(len(output)) <= maxOutputBytes {
		return output, false, nil
	}
	return output[:maxOutputBytes], true, nil
}

func fakeReviewEvidenceGitKey(args ...string) string {
	return strings.Join(args, "\x00")
}
