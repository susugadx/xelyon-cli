package review

import (
	"context"
	"strings"
	"testing"

	reviewdomain "github.com/susugadx/xelyon-cli/internal/review/domain"
)

func TestNewReviewRunnerRejectsNilDependencies(t *testing.T) {
	valid := newRunnerNonNilDependenciesForTest()
	tests := []struct {
		name        string
		opts        ReviewRunnerOptions
		errContains string
	}{
		{
			name:        "model",
			opts:        ReviewRunnerOptions{EvidenceBuilder: valid.EvidenceBuilder, ProbeRunner: valid.ProbeRunner},
			errContains: "review runner model is nil",
		},
		{
			name:        "evidence builder",
			opts:        ReviewRunnerOptions{Model: valid.Model, ProbeRunner: valid.ProbeRunner},
			errContains: "review runner evidence builder is nil",
		},
		{
			name:        "probe runner",
			opts:        ReviewRunnerOptions{Model: valid.Model, EvidenceBuilder: valid.EvidenceBuilder},
			errContains: "review runner probe runner is nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewReviewRunner(tt.opts)
			if err == nil {
				t.Fatal("NewReviewRunner() error = nil, want error")
			}
			if got := err.Error(); got != tt.errContains {
				t.Fatalf("NewReviewRunner() error = %q, want %q", got, tt.errContains)
			}
		})
	}
}

func TestReviewRunnerRunRejectsNilDependencies(t *testing.T) {
	valid := newRunnerNonNilDependenciesForTest()
	tests := []struct {
		name        string
		runner      *ReviewRunner
		errContains string
	}{
		{
			name: "model",
			runner: &ReviewRunner{
				evidenceBuilder: valid.EvidenceBuilder,
				probeRunner:     valid.ProbeRunner,
			},
			errContains: "review runner model is nil",
		},
		{
			name: "evidence builder",
			runner: &ReviewRunner{
				model:       valid.Model,
				probeRunner: valid.ProbeRunner,
			},
			errContains: "review runner evidence builder is nil",
		},
		{
			name: "probe runner",
			runner: &ReviewRunner{
				model:           valid.Model,
				evidenceBuilder: valid.EvidenceBuilder,
			},
			errContains: "review runner probe runner is nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.runner.Run(context.Background(), NewCurrentChangesRequest(""))
			if err == nil {
				t.Fatal("Run() error = nil, want error")
			}
			if got := err.Error(); got != tt.errContains {
				t.Fatalf("Run() error = %q, want %q", got, tt.errContains)
			}
		})
	}
}

func TestReviewRunnerRunRejectsUnknownTargetBeforeEvidenceBuild(t *testing.T) {
	evidence := &runnerFakeEvidenceBuilder{bundle: newRunnerEvidenceBundleForTest("/tmp/review-runner/repo")}
	probes := &runnerFakeProbeRunner{}
	model := &runnerFakeModel{}
	runner := newReviewRunnerForTest(t, evidence, probes, model)

	_, err := runner.Run(context.Background(), ReviewRequest{TargetKind: reviewdomain.TargetKind("staged")})
	if err == nil {
		t.Fatal("Run() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "target_kind") {
		t.Fatalf("Run() error = %q, want target_kind", err.Error())
	}
	if got, want := evidence.calls, 0; got != want {
		t.Fatalf("evidence calls = %d, want %d", got, want)
	}
}
