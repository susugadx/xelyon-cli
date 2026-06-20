package probe

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode"

	"github.com/susugadx/xelyon-cli/internal/review/domain"
)

var (
	// ErrUnsupportedReviewProbeMode は未知または未対応 mode の実行時に返す。
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

	repoRoot, err := r.resolveRepoRoot()
	if err != nil {
		return ReviewProbeResult{}, err
	}

	switch req.Mode {
	case domain.ReviewProbeHostReadOnly:
		return newHostReadOnlyExecutor(repoRoot).run(ctx, req), nil
	case domain.ReviewProbeScratchOnly:
		return newScratchOnlyExecutor(repoRoot).run(ctx, req), nil
	case domain.ReviewProbeRepoSandbox:
		return newRepoSandboxExecutor(repoRoot).run(ctx, req), nil
	default:
		result := newBlockedModeResult(req, fmt.Sprintf("probe mode %q is not supported", req.Mode))
		return result, fmt.Errorf("%w: %s", ErrUnsupportedReviewProbeMode, req.Mode)
	}
}

func newBlockedModeResult(req ReviewProbeRequest, message string) ReviewProbeResult {
	return ReviewProbeResult{
		ID:     req.ID,
		Mode:   req.Mode,
		Status: domain.ReviewProbeBlocked,
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

// FormatProbeCommand は command と args を人間向け表示文字列へ整形する。
func FormatProbeCommand(command string, args []string) string {
	command = strings.TrimSpace(command)
	if command == "" && len(args) == 0 {
		return ""
	}
	parts := make([]string, 0, 1+len(args))
	if command != "" {
		parts = append(parts, command)
	}
	for _, arg := range args {
		parts = append(parts, formatProbeCommandArg(arg))
	}
	return strings.Join(parts, " ")
}

func formatProbeCommand(command string, args []string) string {
	return FormatProbeCommand(command, args)
}

func formatProbeCommandArg(arg string) string {
	if !probeCommandArgNeedsShellQuote(arg) {
		return arg
	}
	return strconv.Quote(arg)
}

func probeCommandArgNeedsShellQuote(arg string) bool {
	if arg == "" {
		return true
	}
	for _, r := range arg {
		if unicode.IsSpace(r) {
			return true
		}
		switch r {
		case '\n', '\r', ';', '|', '&', '<', '>', '`':
			return true
		}
	}
	return strings.Contains(arg, "$(")
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
