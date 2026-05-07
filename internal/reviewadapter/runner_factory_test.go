package reviewadapter

import (
	"context"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/review"
)

func TestRunnerFactoryRejectsNilModel(t *testing.T) {
	factory := NewRunnerFactory(RunnerFactoryOptions{
		RepoRoot: t.TempDir(),
		CWD:      t.TempDir(),
	})

	_, err := factory.NewReviewRunner()
	if err == nil {
		t.Fatal("NewReviewRunner() error = nil, want nil model error")
	}
	if got, want := err.Error(), "review runner model is nil"; !strings.Contains(got, want) {
		t.Fatalf("NewReviewRunner() error = %q, want %q", got, want)
	}
}

func TestRunnerFactoryBuildsRunnerWithInjectedDependencies(t *testing.T) {
	factory := NewRunnerFactory(RunnerFactoryOptions{
		RepoRoot:        t.TempDir(),
		CWD:             t.TempDir(),
		Model:           fakeReviewModel{},
		EvidenceBuilder: fakeEvidenceBuilder{},
		ProbeRunner:     fakeProbeRunner{},
	})

	runner, err := factory.NewReviewRunner()
	if err != nil {
		t.Fatalf("NewReviewRunner() error = %v, want nil", err)
	}
	if runner == nil {
		t.Fatal("NewReviewRunner() runner = nil, want non-nil")
	}
}

func TestRunnerFactoryBuildsRunnerWithDefaultEvidenceAndProbe(t *testing.T) {
	factory := NewRunnerFactory(RunnerFactoryOptions{
		RepoRoot: t.TempDir(),
		CWD:      t.TempDir(),
		Model:    fakeReviewModel{},
	})

	runner, err := factory.NewReviewRunner()
	if err != nil {
		t.Fatalf("NewReviewRunner() error = %v, want nil", err)
	}
	if runner == nil {
		t.Fatal("NewReviewRunner() runner = nil, want non-nil")
	}
}

type fakeReviewModel struct{}

func (fakeReviewModel) CompleteReview(context.Context, review.ReviewModelRequest) (review.ReviewModelResponse, error) {
	return review.ReviewModelResponse{}, nil
}

type fakeEvidenceBuilder struct{}

func (fakeEvidenceBuilder) BuildCurrentChanges(context.Context) (review.ReviewEvidenceBundle, error) {
	return review.ReviewEvidenceBundle{}, nil
}

type fakeProbeRunner struct{}

func (fakeProbeRunner) Run(context.Context, review.ReviewProbeRequest) (review.ReviewProbeResult, error) {
	return review.ReviewProbeResult{}, nil
}
