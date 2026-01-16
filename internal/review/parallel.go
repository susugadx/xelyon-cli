package review

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"
)

// ParallelFixResult は個々の修正適用結果（並列実行用）
type ParallelFixResult struct {
	ProposalID string
	FilePath   string
	Success    bool
	Error      error
	Duration   time.Duration
}

// ParallelExecutor は修正を並列適用するエグゼキューター
type ParallelExecutor struct {
	MaxWorkers int // デフォルト4
	Results    []ParallelFixResult
	mu         sync.Mutex
}

// NewParallelExecutor は新しいParallelExecutorを作成
func NewParallelExecutor(maxWorkers int) *ParallelExecutor {
	if maxWorkers <= 0 {
		maxWorkers = 4
	}
	return &ParallelExecutor{
		MaxWorkers: maxWorkers,
		Results:    make([]ParallelFixResult, 0),
	}
}

// FixGroup はファイル単位でグループ化された修正提案
type FixGroup struct {
	FilePath  string
	Proposals []FixProposal
}

// GroupByFile は修正提案をファイル単位でグループ化
// 同じファイルへの修正は順番に適用する必要がある
func GroupByFile(proposals []FixProposal) []FixGroup {
	if len(proposals) == 0 {
		return nil
	}

	// ファイルパスごとにグループ化
	grouped := make(map[string][]FixProposal)
	for _, p := range proposals {
		grouped[p.FilePath] = append(grouped[p.FilePath], p)
	}

	// 安定したソートのためにキーをソート
	var keys []string
	for k := range grouped {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	result := make([]FixGroup, 0, len(keys))
	for _, k := range keys {
		result = append(result, FixGroup{
			FilePath:  k,
			Proposals: grouped[k],
		})
	}

	return result
}

// IndependentGroup は独立して実行可能なグループ
type IndependentGroup struct {
	Groups []FixGroup
}

// GroupIndependent は独立して実行可能なグループに分類
// 異なるファイルへの修正は並列実行可能
func GroupIndependent(groups []FixGroup) []IndependentGroup {
	if len(groups) == 0 {
		return nil
	}

	// 各ファイルグループは独立して実行可能
	// ただし同一ファイル内の修正は順次実行
	independent := make([]IndependentGroup, 0)

	// シンプルな実装: すべてのファイルグループを1つの独立グループにまとめる
	// （異なるファイルは並列実行可能なため）
	independent = append(independent, IndependentGroup{
		Groups: groups,
	})

	return independent
}

// CanRunParallel は2つの修正提案が並列実行可能か判定
func CanRunParallel(a, b FixProposal) bool {
	// 同じファイルへの修正は並列実行不可
	if a.FilePath == b.FilePath {
		return false
	}

	// 異なるファイルへの修正は並列実行可能
	return true
}

// ApplyFunc は修正を適用する関数の型
type ApplyFunc func(ctx context.Context, proposal FixProposal) error

// Execute は修正を並列に適用
func (pe *ParallelExecutor) Execute(ctx context.Context, groups []FixGroup, applyFn ApplyFunc) []ParallelFixResult {
	if len(groups) == 0 {
		return nil
	}

	// セマフォとして使用するチャネル
	sem := make(chan struct{}, pe.MaxWorkers)
	var wg sync.WaitGroup

	// 各ファイルグループを並列に処理
	for _, group := range groups {
		wg.Add(1)
		go func(g FixGroup) {
			defer wg.Done()

			// セマフォを取得
			sem <- struct{}{}
			defer func() { <-sem }()

			// 同一ファイル内の修正は順次適用
			pe.applyGroupSequentially(ctx, g, applyFn)
		}(group)
	}

	wg.Wait()

	pe.mu.Lock()
	results := make([]ParallelFixResult, len(pe.Results))
	copy(results, pe.Results)
	pe.mu.Unlock()

	return results
}

// applyGroupSequentially はファイルグループ内の修正を順次適用
func (pe *ParallelExecutor) applyGroupSequentially(ctx context.Context, group FixGroup, applyFn ApplyFunc) {
	for _, proposal := range group.Proposals {
		select {
		case <-ctx.Done():
			pe.addResult(ParallelFixResult{
				ProposalID: proposal.IssueID,
				FilePath:   proposal.FilePath,
				Success:    false,
				Error:      ctx.Err(),
			})
			return
		default:
		}

		start := time.Now()
		err := applyFn(ctx, proposal)
		duration := time.Since(start)

		pe.addResult(ParallelFixResult{
			ProposalID: proposal.IssueID,
			FilePath:   proposal.FilePath,
			Success:    err == nil,
			Error:      err,
			Duration:   duration,
		})

		// エラーが発生した場合、同一ファイルの残りの修正はスキップ
		if err != nil {
			for _, remaining := range group.Proposals {
				if remaining.IssueID != proposal.IssueID {
					pe.addResult(ParallelFixResult{
						ProposalID: remaining.IssueID,
						FilePath:   remaining.FilePath,
						Success:    false,
						Error:      fmt.Errorf("skipped due to previous error in same file"),
					})
				}
			}
			return
		}
	}
}

// addResult は結果を追加（スレッドセーフ）
func (pe *ParallelExecutor) addResult(result ParallelFixResult) {
	pe.mu.Lock()
	defer pe.mu.Unlock()
	pe.Results = append(pe.Results, result)
}

// Summary は実行結果のサマリーを返す
func (pe *ParallelExecutor) Summary() (total, success, failed int) {
	pe.mu.Lock()
	defer pe.mu.Unlock()

	total = len(pe.Results)
	for _, r := range pe.Results {
		if r.Success {
			success++
		} else {
			failed++
		}
	}
	return
}

// TotalDuration は全体の実行時間を返す
func (pe *ParallelExecutor) TotalDuration() time.Duration {
	pe.mu.Lock()
	defer pe.mu.Unlock()

	var total time.Duration
	for _, r := range pe.Results {
		total += r.Duration
	}
	return total
}
