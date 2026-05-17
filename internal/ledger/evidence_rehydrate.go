package ledger

import (
	"context"
	"fmt"
)

// EvidencePointer は現在ファイルから再読込できる evidence の位置情報。
type EvidencePointer struct {
	Path       string
	StartLine  int
	EndLine    int
	Source     string
	ToolCallID string
	FileHash   string
	Stale      bool
	PathBase   EvidencePointerPathBase
}

// EvidenceRehydrateOptions は evidence pointer の再読込に使う workspace 情報。
type EvidenceRehydrateOptions struct {
	RepoRoot      string
	InvocationCWD string
}

// EvidenceRehydrateResult は evidence pointer の現在ファイル再読込結果。
type EvidenceRehydrateResult struct {
	Path            string
	StartLine       int
	EndLine         int
	Content         string
	CurrentFileHash string
	Stale           bool
	Reason          EvidenceRehydrateErrorReason
}

// EvidenceRehydrateErrorReason は evidence rehydrate が失敗した理由。
type EvidenceRehydrateErrorReason string

const (
	EvidenceRehydrateReasonWorkspaceUnavailable EvidenceRehydrateErrorReason = "workspace_unavailable"
	EvidenceRehydrateReasonInvalidPath          EvidenceRehydrateErrorReason = "invalid_path"
	EvidenceRehydrateReasonPathEscape           EvidenceRehydrateErrorReason = "path_escape"
	EvidenceRehydrateReasonMissingFile          EvidenceRehydrateErrorReason = "missing_file"
	EvidenceRehydrateReasonNotRegularFile       EvidenceRehydrateErrorReason = "not_regular_file"
	EvidenceRehydrateReasonUnreadableFile       EvidenceRehydrateErrorReason = "unreadable_file"
	EvidenceRehydrateReasonBinaryFile           EvidenceRehydrateErrorReason = "binary_file"
	EvidenceRehydrateReasonInvalidRange         EvidenceRehydrateErrorReason = "invalid_range"
	EvidenceRehydrateReasonRangeOutOfBounds     EvidenceRehydrateErrorReason = "range_out_of_bounds"
	EvidenceRehydrateReasonContextCancelled     EvidenceRehydrateErrorReason = "context_cancelled"
)

// EvidenceRehydrateError は evidence rehydrate の structured error。
type EvidenceRehydrateError struct {
	Reason EvidenceRehydrateErrorReason
	Path   string
	Err    error
}

// EvidencePointerPathBase は relative Path の解決基準を表す。
type EvidencePointerPathBase string

const (
	// EvidencePointerPathBaseAuto は repo root を先に試し、必要なら invocation cwd を fallback に使う。
	EvidencePointerPathBaseAuto EvidencePointerPathBase = ""
	// EvidencePointerPathBaseRepoRoot は Path を repo-relative としてだけ解決する。
	EvidencePointerPathBaseRepoRoot EvidencePointerPathBase = "repo_root"
)

func (e *EvidenceRehydrateError) Error() string {
	if e == nil {
		return "rehydrate evidence pointer: unknown error"
	}
	message := fmt.Sprintf("rehydrate evidence pointer: %s", e.Reason)
	if e.Path != "" {
		message += ": " + e.Path
	}
	if e.Err != nil {
		message += ": " + e.Err.Error()
	}
	return message
}

func (e *EvidenceRehydrateError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

// RehydrateEvidencePointer は evidence pointer の対象 range を現在ファイルから読み直す。
func RehydrateEvidencePointer(ctx context.Context, pointer EvidencePointer, opts EvidenceRehydrateOptions) (EvidenceRehydrateResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return evidenceRehydrateFailure(EvidenceRehydrateResult{Path: pointer.Path}, EvidenceRehydrateReasonContextCancelled, err)
	}

	workspace, result, err := newEvidenceRehydrateWorkspace(opts)
	if err != nil {
		return result, err
	}
	resolved, result, err := resolveEvidencePointerPath(pointer, workspace)
	if err != nil {
		return result, err
	}

	return rehydrateEvidenceResolvedPointer(ctx, pointer, workspace, resolved)
}

// RehydrateEvidencePointer は Store の workspace 情報で evidence pointer を再読込する。
func (s *Store) RehydrateEvidencePointer(ctx context.Context, pointer EvidencePointer) (EvidenceRehydrateResult, error) {
	if s == nil {
		return evidenceRehydrateFailure(EvidenceRehydrateResult{Path: pointer.Path}, EvidenceRehydrateReasonWorkspaceUnavailable, nil)
	}
	return RehydrateEvidencePointer(ctx, pointer, EvidenceRehydrateOptions{
		RepoRoot:      s.repoRoot,
		InvocationCWD: s.invocationCWD,
	})
}

func evidenceRehydrateFailure(result EvidenceRehydrateResult, reason EvidenceRehydrateErrorReason, err error) (EvidenceRehydrateResult, error) {
	result.Reason = reason
	return result, newEvidenceRehydrateError(reason, result.Path, err)
}

func newEvidenceRehydrateError(reason EvidenceRehydrateErrorReason, path string, err error) *EvidenceRehydrateError {
	return &EvidenceRehydrateError{
		Reason: reason,
		Path:   path,
		Err:    err,
	}
}
