package agent

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/susugadx/xelyon-cli/internal/review"
	"github.com/susugadx/xelyon-cli/internal/reviewadapter"
)

const reviewRunArtifactsEnv = "XELYON_REVIEW_RUN_ARTIFACTS"

// RunReview は Agent runtime を使って /review current_changes runner を実行する。
func (a *Agent) RunReview(ctx context.Context, req review.ReviewRequest) (review.ReviewReport, error) {
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
	factory := reviewadapter.NewRunnerFactory(reviewadapter.RunnerFactoryOptions{
		RepoRoot:              repoRoot,
		CWD:                   cwd,
		Model:                 agentReviewModel{agent: a},
		ArtifactWriter:        artifactSink.writer,
		ArtifactWarningWriter: a.errorOutput(),
	})
	runner, err := factory.NewReviewRunner()
	if err != nil {
		return review.ReviewReport{}, fmt.Errorf("review run build runner: %w", err)
	}
	report, runErr := runner.Run(ctx, req)
	if err := artifactSink.flushArtifacts(); err != nil {
		fmt.Fprintf(a.errorOutput(), "Warning: failed to save review artifact %s: %v\n", artifactSink.runDir, err)
	}
	if runErr != nil {
		return review.ReviewReport{}, runErr
	}
	return report, nil
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
