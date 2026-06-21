package review

import (
	"testing"
)

func newReviewRunnerForTest(t *testing.T, evidence ReviewEvidenceProvider, probes ReviewProbeExecutor, model ReviewModel) *ReviewRunner {
	t.Helper()

	runner, err := NewReviewRunner(ReviewRunnerOptions{
		EvidenceBuilder: evidence,
		ProbeRunner:     probes,
		Model:           model,
	})
	if err != nil {
		t.Fatalf("NewReviewRunner() error = %v, want nil", err)
	}
	return runner
}

func newRunnerNonNilDependenciesForTest() ReviewRunnerOptions {
	return ReviewRunnerOptions{
		EvidenceBuilder: &runnerFakeEvidenceBuilder{bundle: newRunnerEvidenceBundleForTest("/tmp/review-runner/repo")},
		ProbeRunner:     &runnerFakeProbeRunner{},
		Model:           &runnerFakeModel{},
	}
}
