package ledger

import (
	"context"
	"errors"
)

const (
	defaultRehydratePlanExecutionMaxItems      = 8
	defaultRehydratePlanExecutionMaxTotalLines = 240
	defaultRehydratePlanExecutionMaxTotalBytes = 65536
)

// RehydratePlanExecutionOptions は RehydratePlan 実行時の model-facing 予算を表す。
type RehydratePlanExecutionOptions struct {
	MaxItems      int
	MaxTotalLines int
	MaxTotalBytes int
}

// RehydratedEvidenceBlock は provider active context に載せる再読込済み evidence。
type RehydratedEvidenceBlock struct {
	Items     []RehydratedEvidenceItem
	Truncated bool
}

// RehydratedEvidenceItem は現在ファイルから読み直した evidence range。
type RehydratedEvidenceItem struct {
	Path            string
	StartLine       int
	EndLine         int
	Source          string
	Reason          string
	ToolCallID      string
	Content         string
	CurrentFileHash string
	Stale           bool
}

// RehydratePlanExecutionReport は実行結果と、provider input から除外した失敗を分けて保持する。
type RehydratePlanExecutionReport struct {
	Block     RehydratedEvidenceBlock
	Failures  []RehydratePlanExecutionFailure
	Truncated bool
}

// RehydratePlanExecutionFailure は実読込前後で拒否された plan item の診断情報。
type RehydratePlanExecutionFailure struct {
	Path        string
	StartLine   int
	EndLine     int
	Source      string
	PlanReason  string
	ToolCallID  string
	ErrorReason EvidenceRehydrateErrorReason
}

// ExecuteRehydratePlan は dry-run plan に従い、現在ファイルから provider 用 evidence を読み直す。
// 失敗 item は report に残すだけで、model-facing block には含めない。
func ExecuteRehydratePlan(ctx context.Context, plan RehydratePlan, workspace EvidenceRehydrateOptions, opts RehydratePlanExecutionOptions) RehydratePlanExecutionReport {
	if ctx == nil {
		ctx = context.Background()
	}
	opts = normalizeRehydratePlanExecutionOptions(opts)
	report := RehydratePlanExecutionReport{}
	if len(plan.Items) == 0 {
		return report
	}

	if err := ctx.Err(); err != nil {
		return rehydratePlanWorkspaceFailureReport(plan, opts, EvidenceRehydrateReasonContextCancelled)
	}

	_, _, err := newEvidenceRehydrateWorkspace(workspace)
	if err != nil {
		return rehydratePlanWorkspaceFailureReport(plan, opts, EvidenceRehydrateReasonWorkspaceUnavailable)
	}

	totalLines := 0
	for index, planItem := range plan.Items {
		if len(report.Block.Items) >= opts.MaxItems {
			report.Truncated = true
			report.Block.Truncated = true
			break
		}
		if err := ctx.Err(); err != nil {
			report.Failures = append(report.Failures, rehydratePlanExecutionFailure(planItem, EvidenceRehydrateReasonContextCancelled))
			report.Truncated = index < len(plan.Items)-1
			report.Block.Truncated = report.Truncated
			break
		}

		pointer, reason, ok := rehydratePlanExecutionPointer(planItem)
		if !ok {
			report.Failures = append(report.Failures, rehydratePlanExecutionFailure(planItem, reason))
			continue
		}

		result, err := RehydrateEvidencePointer(ctx, pointer, workspace)
		if err != nil {
			report.Failures = append(report.Failures, rehydratePlanExecutionFailure(planItem, rehydratePlanExecutionErrorReason(result, err)))
			continue
		}

		item := rehydratedEvidenceItemForResult(planItem, result)
		itemLines := rehydratedEvidenceItemLineCount(item)
		if itemLines <= 0 {
			report.Failures = append(report.Failures, rehydratePlanExecutionFailure(planItem, EvidenceRehydrateReasonInvalidRange))
			continue
		}
		if totalLines+itemLines > opts.MaxTotalLines {
			report.Truncated = true
			report.Block.Truncated = true
			break
		}

		candidateBlock := report.Block
		candidateBlock.Items = append(append([]RehydratedEvidenceItem(nil), report.Block.Items...), item)
		if len(RenderRehydratedEvidenceBlock(candidateBlock)) > opts.MaxTotalBytes {
			report.Truncated = true
			report.Block.Truncated = true
			break
		}

		report.Block.Items = append(report.Block.Items, item)
		totalLines += itemLines
	}
	return report
}

// ExecuteRehydratePlan は Store の workspace 情報で rehydrate plan を実行する。
func (s *Store) ExecuteRehydratePlan(ctx context.Context, plan RehydratePlan, opts RehydratePlanExecutionOptions) RehydratePlanExecutionReport {
	if s == nil {
		return rehydratePlanWorkspaceFailureReport(plan, normalizeRehydratePlanExecutionOptions(opts), EvidenceRehydrateReasonWorkspaceUnavailable)
	}
	s.mu.Lock()
	workspace := EvidenceRehydrateOptions{RepoRoot: s.repoRoot, InvocationCWD: s.invocationCWD}
	s.mu.Unlock()
	return ExecuteRehydratePlan(ctx, plan, workspace, opts)
}

func normalizeRehydratePlanExecutionOptions(opts RehydratePlanExecutionOptions) RehydratePlanExecutionOptions {
	if opts.MaxItems <= 0 {
		opts.MaxItems = defaultRehydratePlanExecutionMaxItems
	}
	if opts.MaxTotalLines <= 0 {
		opts.MaxTotalLines = defaultRehydratePlanExecutionMaxTotalLines
	}
	if opts.MaxTotalBytes <= 0 {
		opts.MaxTotalBytes = defaultRehydratePlanExecutionMaxTotalBytes
	}
	return opts
}

func rehydratePlanWorkspaceFailureReport(plan RehydratePlan, opts RehydratePlanExecutionOptions, reason EvidenceRehydrateErrorReason) RehydratePlanExecutionReport {
	report := RehydratePlanExecutionReport{}
	limit := len(plan.Items)
	if limit > opts.MaxItems {
		limit = opts.MaxItems
		report.Truncated = true
		report.Block.Truncated = true
	}
	for _, item := range plan.Items[:limit] {
		report.Failures = append(report.Failures, rehydratePlanExecutionFailure(item, reason))
	}
	return report
}

func rehydratePlanExecutionPointer(item RehydratePlanItem) (EvidencePointer, EvidenceRehydrateErrorReason, bool) {
	if reason, ok := invalidRawEvidencePointerPathReason(item.Path); ok {
		return EvidencePointer{}, reason, false
	}
	candidate := cleanPathCandidate(item.Path)
	if reason, ok := invalidEvidencePointerPathReason(candidate); ok {
		return EvidencePointer{}, reason, false
	}
	relativePath, reason, ok := cleanEvidencePointerRelativePath(candidate)
	if !ok {
		return EvidencePointer{}, reason, false
	}
	return EvidencePointer{
		Path:       relativePath,
		StartLine:  item.StartLine,
		EndLine:    item.EndLine,
		Source:     item.Source,
		ToolCallID: item.ToolCallID,
		FileHash:   item.FileHash,
		Stale:      item.Stale,
		PathBase:   EvidencePointerPathBaseRepoRoot,
	}, "", true
}

func rehydratePlanExecutionErrorReason(result EvidenceRehydrateResult, err error) EvidenceRehydrateErrorReason {
	if result.Reason != "" {
		return result.Reason
	}
	var rehydrateErr *EvidenceRehydrateError
	if errors.As(err, &rehydrateErr) && rehydrateErr.Reason != "" {
		return rehydrateErr.Reason
	}
	return EvidenceRehydrateReasonUnreadableFile
}

func rehydratedEvidenceItemForResult(planItem RehydratePlanItem, result EvidenceRehydrateResult) RehydratedEvidenceItem {
	return RehydratedEvidenceItem{
		Path:            result.Path,
		StartLine:       result.StartLine,
		EndLine:         result.EndLine,
		Source:          planItem.Source,
		Reason:          planItem.Reason,
		ToolCallID:      planItem.ToolCallID,
		Content:         result.Content,
		CurrentFileHash: result.CurrentFileHash,
		Stale:           planItem.Stale || result.Stale,
	}
}

func rehydratedEvidenceItemLineCount(item RehydratedEvidenceItem) int {
	return item.EndLine - item.StartLine + 1
}

func rehydratePlanExecutionFailure(item RehydratePlanItem, reason EvidenceRehydrateErrorReason) RehydratePlanExecutionFailure {
	return RehydratePlanExecutionFailure{
		Path:        item.Path,
		StartLine:   item.StartLine,
		EndLine:     item.EndLine,
		Source:      item.Source,
		PlanReason:  item.Reason,
		ToolCallID:  item.ToolCallID,
		ErrorReason: reason,
	}
}
