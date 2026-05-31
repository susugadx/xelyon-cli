package reviewadapter

import (
	"context"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/api/websearch"
	"github.com/susugadx/xelyon-cli/internal/config"
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

func TestReviewWebSearchRunnerPropagatesTruncationAndUsageAttribution(t *testing.T) {
	const provider = "gemini"
	websearch.RegisterWithContextForTest(t, provider, func(ctx context.Context, query, model string) (string, error) {
		if query != "OpenAI web_search docs" {
			t.Fatalf("query = %q, want OpenAI web_search docs", query)
		}
		if model != "gemini-review-search" {
			t.Fatalf("model = %q, want gemini-review-search", model)
		}
		callback := websearch.UsageCallbackFromContext(ctx)
		if callback == nil {
			t.Fatal("UsageCallbackFromContext() = nil, want callback")
		}
		callback(api.Usage{InputTokens: 11, OutputTokens: 2})
		return "1. First\n   URL: https://docs.example.test/first\n\n2. Second\n   URL: https://docs.example.test/second", nil
	})
	cfg := config.DefaultConfig()
	cfg.WebSearch.Provider = provider
	cfg.WebSearch.CacheEnabled = false
	cfg.SetProviderModelConfig(provider, config.ProviderModelConfig{DefaultModel: "gemini-review-search"})

	var gotProvider string
	var gotModel string
	var gotUsage api.Usage
	result, err := newReviewWebSearchRunner(RunnerFactoryOptions{
		Config: cfg,
		UsageAttribution: func(provider, model string, usage api.Usage) {
			gotProvider = provider
			gotModel = model
			gotUsage = usage
		},
	}).SearchReviewWeb(context.Background(), "OpenAI web_search docs", 1)
	if err != nil {
		t.Fatalf("SearchReviewWeb() error = %v", err)
	}
	if result.Provider != provider {
		t.Fatalf("Provider = %q, want %q", result.Provider, provider)
	}
	if !result.Truncated {
		t.Fatal("Truncated = false, want true from SearchWeb")
	}
	if len(result.Results) != 1 || result.Results[0].URL != "https://docs.example.test/first" {
		t.Fatalf("Results = %#v, want first bounded result", result.Results)
	}
	if gotProvider != provider || gotModel != "gemini-review-search" {
		t.Fatalf("usage owner = %s/%s, want gemini/gemini-review-search", gotProvider, gotModel)
	}
	if gotUsage.InputTokens != 11 || gotUsage.OutputTokens != 2 {
		t.Fatalf("usage = %+v, want input=11 output=2", gotUsage)
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
