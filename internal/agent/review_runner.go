package agent

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/review"
	"github.com/susugadx/xelyon-cli/internal/reviewadapter"
)

const reviewRunArtifactsEnv = "XELYON_REVIEW_RUN_ARTIFACTS"

type reviewRunOptions struct {
	ProgressSink review.ReviewProgressSink
}

// RunReview は Agent runtime を使って /review current_changes runner を実行する。
func (a *Agent) RunReview(ctx context.Context, req review.ReviewRequest) (review.ReviewReport, error) {
	return a.runReview(ctx, req, reviewRunOptions{})
}

func (a *Agent) runReview(ctx context.Context, req review.ReviewRequest, opts reviewRunOptions) (review.ReviewReport, error) {
	if a == nil {
		return review.ReviewReport{}, fmt.Errorf("review run: agent is nil")
	}
	ctx, cleanup := a.beginReviewRequestContext(ctx)
	defer cleanup()

	cwd, err := os.Getwd()
	if err != nil {
		return review.ReviewReport{}, fmt.Errorf("review run resolve cwd: %w", err)
	}
	repoRoot, err := review.ResolveReviewRepoRoot(ctx, cwd)
	if err != nil {
		return review.ReviewReport{}, err
	}

	artifactSink := a.newReviewRunArtifactSink(repoRoot)
	cfg := a.cfg()
	rawOutputArtifacts := reviewRawOutputArtifactsConfigForRuntime(a.Runtime)
	factory := reviewadapter.NewRunnerFactory(reviewadapter.RunnerFactoryOptions{
		RepoRoot:                          repoRoot,
		CWD:                               cwd,
		Config:                            cfg,
		MainProvider:                      a.ProviderName,
		MainProviderConfigKey:             a.currentProviderConfigKey(),
		MainModel:                         a.CurrentModel,
		Model:                             agentReviewModel{agent: a},
		ArtifactWriter:                    artifactSink.writer,
		ArtifactWarningWriter:             a.errorOutput(),
		ProgressSink:                      opts.ProgressSink,
		UsageAttribution:                  a.providerUsageAttributionCallback(),
		PromptReductionMode:               a.reviewPromptReductionMode(),
		RawOutputArtifactsMode:            a.reviewRawOutputArtifactsMode(),
		RawOutputArtifactStore:            a.reviewRawOutputArtifactStore(),
		RawOutputSessionID:                a.providerHistoryRawOutputArtifactSessionID(),
		RawOutputRehydrateBudgetTokens:    rawOutputArtifacts.ActiveContextBudgetTokens,
		RawOutputRehydrateBudgetMaxTokens: rawOutputArtifacts.ActiveContextBudgetMaxTokens,
	})
	runner, err := factory.NewReviewRunner()
	if err != nil {
		return review.ReviewReport{}, fmt.Errorf("review run build runner: %w", err)
	}
	report, runErr := runner.Run(ctx, req)
	a.recordLastReviewPromptReductionReport(runner.PromptReductionReport())
	if err := artifactSink.flushArtifacts(); err != nil {
		fmt.Fprintf(a.errorOutput(), "Warning: failed to save review artifact %s: %v\n", artifactSink.runDir, err)
	}
	if runErr != nil {
		return review.ReviewReport{}, runErr
	}
	return report, nil
}

func (a *Agent) recordLastReviewPromptReductionReport(report review.ReviewPromptReductionReport) {
	if a == nil || a.Runtime == nil {
		return
	}
	a.Runtime.LastReviewPromptReductionReport = review.CloneReviewPromptReductionReport(report)
}

func (a *Agent) reviewPromptReductionMode() review.ReviewPromptReductionMode {
	if a == nil {
		return review.ReviewPromptReductionModeOff
	}
	switch providerHistoryReductionModeResolutionForRuntime(a.Runtime).effective {
	case ProviderHistoryReductionApply:
		return review.ReviewPromptReductionModeApply
	case ProviderHistoryReductionDryRun:
		return review.ReviewPromptReductionModeDryRun
	default:
		return review.ReviewPromptReductionModeOff
	}
}

func (a *Agent) reviewRawOutputArtifactsMode() review.ReviewRawOutputArtifactsMode {
	if a == nil {
		return review.ReviewRawOutputArtifactsModeOff
	}
	switch reviewRawOutputArtifactsConfigForRuntime(a.Runtime).Mode {
	case config.ProviderHistoryRawOutputArtifactsModeApply:
		return review.ReviewRawOutputArtifactsModeApply
	case config.ProviderHistoryRawOutputArtifactsModeDryRun:
		return review.ReviewRawOutputArtifactsModeDryRun
	default:
		return review.ReviewRawOutputArtifactsModeOff
	}
}

func (a *Agent) reviewRawOutputArtifactStore() review.ReviewRawOutputArtifactStore {
	if a == nil {
		return nil
	}
	if a.reviewPromptReductionMode() == review.ReviewPromptReductionModeOff {
		return nil
	}
	if a.reviewRawOutputArtifactsMode() == review.ReviewRawOutputArtifactsModeOff {
		return nil
	}
	store := a.providerHistoryRawOutputArtifactStore()
	if typed, ok := store.(review.ReviewRawOutputArtifactStore); ok {
		return typed
	}
	return nil
}

func reviewRawOutputArtifactsConfigForRuntime(runtime *AgentRuntime) config.ProviderHistoryRawOutputArtifactsConfig {
	defaults := config.DefaultProviderHistoryRawOutputArtifactsConfig()
	if runtime == nil {
		return defaults
	}
	cfg := runtime.Options.ProviderHistoryRawOutputArtifacts
	if cfg.Mode == "" {
		cfg.Mode = defaults.Mode
	}
	if cfg.MaxArtifactBytes <= 0 {
		cfg.MaxArtifactBytes = defaults.MaxArtifactBytes
	}
	if cfg.SessionQuotaBytes <= 0 {
		cfg.SessionQuotaBytes = defaults.SessionQuotaBytes
	}
	if cfg.ChunkBytes <= 0 {
		cfg.ChunkBytes = defaults.ChunkBytes
	}
	if cfg.ActiveContextBudgetTokens <= 0 {
		cfg.ActiveContextBudgetTokens = defaults.ActiveContextBudgetTokens
	}
	if cfg.ActiveContextBudgetMaxTokens <= 0 {
		cfg.ActiveContextBudgetMaxTokens = defaults.ActiveContextBudgetMaxTokens
	}
	if cfg.Retention == "" {
		cfg.Retention = defaults.Retention
	}
	return cfg
}

type reviewRunArtifactSink struct {
	writer review.ReviewRunArtifactWriter
	runDir string
	flush  func() error
}

func (s reviewRunArtifactSink) flushArtifacts() error {
	if s.flush == nil {
		return nil
	}
	return s.flush()
}

func (a *Agent) newReviewRunArtifactSink(repoRoot string) reviewRunArtifactSink {
	if os.Getenv(reviewRunArtifactsEnv) != "1" {
		return reviewRunArtifactSink{}
	}
	runID := time.Now().UTC().Format("20060102T150405.000000000Z")
	runDir := filepath.Join(repoRoot, ".xelyon", "review-runs", runID)
	buffer := review.NewBufferedReviewRunArtifactWriter()
	return reviewRunArtifactSink{
		writer: buffer,
		runDir: runDir,
		flush: func() error {
			if buffer.Len() == 0 {
				return nil
			}
			writer, err := review.NewReviewRunRepoArtifactWriter(repoRoot, runID)
			if err != nil {
				return err
			}
			return buffer.FlushTo(writer)
		},
	}
}
