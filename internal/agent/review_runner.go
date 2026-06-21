package agent

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/review"
	reviewartifact "github.com/susugadx/xelyon-cli/internal/review/artifact"
	reviewpromptreduction "github.com/susugadx/xelyon-cli/internal/review/promptreduction"
	reviewreport "github.com/susugadx/xelyon-cli/internal/review/report"
	"github.com/susugadx/xelyon-cli/internal/reviewadapter"
)

const reviewRunArtifactsEnv = "XELYON_REVIEW_RUN_ARTIFACTS"

type reviewRunOptions struct {
	ProgressSink review.ReviewProgressSink
}

// RunReview は Agent runtime を使って /review current_changes runner を実行する。
func (a *Agent) RunReview(ctx context.Context, req review.ReviewRequest) (reviewreport.ReviewReport, error) {
	return a.runReview(ctx, req, reviewRunOptions{})
}

func (a *Agent) runReview(ctx context.Context, req review.ReviewRequest, opts reviewRunOptions) (reviewreport.ReviewReport, error) {
	if a == nil {
		return reviewreport.ReviewReport{}, fmt.Errorf("review run: agent is nil")
	}
	ctx, cleanup := a.beginReviewRequestContext(ctx)
	defer cleanup()

	cwd, err := os.Getwd()
	if err != nil {
		return reviewreport.ReviewReport{}, fmt.Errorf("review run resolve cwd: %w", err)
	}
	repoRoot, err := review.ResolveReviewRepoRoot(ctx, cwd)
	if err != nil {
		return reviewreport.ReviewReport{}, err
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
		return reviewreport.ReviewReport{}, fmt.Errorf("review run build runner: %w", err)
	}
	report, runErr := runner.Run(ctx, req)
	a.recordLastReviewPromptReductionReport(runner.PromptReductionReport())
	if err := artifactSink.flushArtifacts(); err != nil {
		fmt.Fprintf(a.errorOutput(), "Warning: failed to save review artifact %s: %v\n", artifactSink.runDir, err)
	}
	if runErr != nil {
		return reviewreport.ReviewReport{}, runErr
	}
	return report, nil
}

func (a *Agent) recordLastReviewPromptReductionReport(report reviewpromptreduction.ReviewPromptReductionReport) {
	if a == nil || a.Runtime == nil {
		return
	}
	a.Runtime.LastReviewPromptReductionReport = reviewpromptreduction.CloneReviewPromptReductionReport(report)
}

func (a *Agent) reviewPromptReductionMode() reviewpromptreduction.ReviewPromptReductionMode {
	if a == nil {
		return reviewpromptreduction.ReviewPromptReductionModeOff
	}
	switch providerHistoryReductionModeResolutionForRuntime(a.Runtime).effective {
	case ProviderHistoryReductionApply:
		return reviewpromptreduction.ReviewPromptReductionModeApply
	case ProviderHistoryReductionDryRun:
		return reviewpromptreduction.ReviewPromptReductionModeDryRun
	default:
		return reviewpromptreduction.ReviewPromptReductionModeOff
	}
}

func (a *Agent) reviewRawOutputArtifactsMode() reviewpromptreduction.ReviewRawOutputArtifactsMode {
	if a == nil {
		return reviewpromptreduction.ReviewRawOutputArtifactsModeOff
	}
	switch reviewRawOutputArtifactsConfigForRuntime(a.Runtime).Mode {
	case config.ProviderHistoryRawOutputArtifactsModeApply:
		return reviewpromptreduction.ReviewRawOutputArtifactsModeApply
	case config.ProviderHistoryRawOutputArtifactsModeDryRun:
		return reviewpromptreduction.ReviewRawOutputArtifactsModeDryRun
	default:
		return reviewpromptreduction.ReviewRawOutputArtifactsModeOff
	}
}

func (a *Agent) reviewRawOutputArtifactStore() reviewpromptreduction.ReviewRawOutputArtifactStore {
	if a == nil {
		return nil
	}
	if a.reviewPromptReductionMode() == reviewpromptreduction.ReviewPromptReductionModeOff {
		return nil
	}
	if a.reviewRawOutputArtifactsMode() == reviewpromptreduction.ReviewRawOutputArtifactsModeOff {
		return nil
	}
	store := a.providerHistoryRawOutputArtifactStore()
	if typed, ok := store.(reviewpromptreduction.ReviewRawOutputArtifactStore); ok {
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
	writer reviewartifact.ReviewRunArtifactWriter
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
	buffer := reviewartifact.NewBufferedReviewRunArtifactWriter()
	return reviewRunArtifactSink{
		writer: buffer,
		runDir: runDir,
		flush: func() error {
			if buffer.Len() == 0 {
				return nil
			}
			writer, err := reviewartifact.NewReviewRunRepoArtifactWriter(repoRoot, runID)
			if err != nil {
				return err
			}
			return buffer.FlushTo(writer)
		},
	}
}
