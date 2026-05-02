package review

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var (
	// ErrUnsupportedReviewProbeMode は未実装 mode の実行時に返す。
	ErrUnsupportedReviewProbeMode = errors.New("unsupported review probe mode")
)

// ProbeRunner は review probe 実行を担当する。
type ProbeRunner struct {
	repoRoot string
}

// NewProbeRunner は repo root を基準に probe runner を構築する。
func NewProbeRunner(repoRoot string) *ProbeRunner {
	return &ProbeRunner{
		repoRoot: repoRoot,
	}
}

// Run は ReviewProbeRequest を検証してから実行する。
func (r *ProbeRunner) Run(ctx context.Context, req ReviewProbeRequest) (ReviewProbeResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	req = normalizeProbeRequestExecutionLimits(req)

	repoRoot, err := r.resolveRepoRoot()
	if err != nil {
		return ReviewProbeResult{}, err
	}

	switch req.Mode {
	case ReviewProbeHostReadOnly:
		return newHostReadOnlyExecutor(repoRoot).run(ctx, req), nil
	case ReviewProbeScratchOnly, ReviewProbeRepoSandbox:
		result := newBlockedModeResult(req, fmt.Sprintf("probe mode %q is not implemented yet", req.Mode))
		return result, fmt.Errorf("%w: %s", ErrUnsupportedReviewProbeMode, req.Mode)
	default:
		result := newBlockedModeResult(req, fmt.Sprintf("probe mode %q is not supported", req.Mode))
		return result, fmt.Errorf("%w: %s", ErrUnsupportedReviewProbeMode, req.Mode)
	}
}

func newBlockedModeResult(req ReviewProbeRequest, message string) ReviewProbeResult {
	return ReviewProbeResult{
		ID:     req.ID,
		Mode:   req.Mode,
		Status: ReviewProbeBlocked,
		Error:  message,
	}
}

func (r *ProbeRunner) resolveRepoRoot() (string, error) {
	root := strings.TrimSpace(r.repoRoot)
	if root == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("failed to resolve current directory: %w", err)
		}
		root = cwd
	}

	abs, err := filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("failed to resolve repo root %q: %w", root, err)
	}
	return filepath.Clean(abs), nil
}

func formatProbeCommand(command string, args []string) string {
	if len(args) == 0 {
		return command
	}
	return command + " " + strings.Join(args, " ")
}

func appendError(base, extra string) string {
	if strings.TrimSpace(extra) == "" {
		return base
	}
	if strings.TrimSpace(base) == "" {
		return extra
	}
	return base + "; " + extra
}
