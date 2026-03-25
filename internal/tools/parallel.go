package tools

import (
	"context"
	"strings"
	"sync"
)

// MaxParallelTools は並列実行する tool の最大数。
// CLI での過剰並列を避ける conservative default として 4 を採用。
// 将来的に config.yaml の parallel.max_workers 等で設定可能にする余地がある。
const MaxParallelTools = 4

// parallelSafeTools は静的に parallel-safe と判定されるツール名のセット。
// read-only かつ副作用のないツールのみ含める。
var parallelSafeTools = map[string]bool{
	"read_file":    true,
	"read_files":   true,
	"list_dir":     true,
	"search_code":  true,
	"web_search":   true,
	"git_status":   true,
	"git_log":      true,
	"git_diff":     true,
	"git_branch":   true,
	"git_ls_files": true,
	"spawn_agent":  true,
	"wait_agent":   true,
}

// IsParallelSafe は ToolCall が並列実行可能かを判定する。
// 静的リストに含まれるツール、または bash の read-only コマンドが対象。
func IsParallelSafe(tc *ToolCall) bool {
	if parallelSafeTools[tc.Tool] {
		return true
	}
	// bash: コマンド文字列が read-only なら parallel-safe
	if tc.Tool == "bash" || tc.Tool == "command" {
		cmd := tc.Args["command"]
		if cmd == "" {
			// Args が未設定の場合は RawArgs を参照
			if raw, ok := tc.RawArgs["command"]; ok {
				if s, ok := raw.(string); ok {
					cmd = s
				}
			}
		}
		return IsBashParallelSafe(cmd)
	}
	return false
}

// IsBashParallelSafe は bash コマンドが並列実行に安全かを判定する。
// isBashReadOnly（キャッシュ用）よりも保守的な allowlist を使用する。
// 確認UI を伴う可能性があるコマンドは除外（並列時に stdin 競合するため）。
func IsBashParallelSafe(command string) bool {
	trimmed := strings.TrimSpace(command)
	if trimmed == "" {
		return false
	}

	// パイプ・リダイレクト・連結を含む場合は unsafe（出力順が不定になる）
	for _, ch := range []string{"|", ">", ">>", "&&", ";"} {
		if strings.Contains(trimmed, ch) {
			return false
		}
	}

	// allowlist ベースで判定
	for _, prefix := range bashParallelSafeCommands {
		if trimmed == prefix {
			return true
		}
		if strings.HasPrefix(trimmed, prefix) && len(trimmed) > len(prefix) {
			next := trimmed[len(prefix)]
			if next == ' ' || next == '\t' {
				return true
			}
		}
	}
	return false
}

// bashParallelSafeCommands は並列実行に安全な bash コマンドのプレフィックス。
// isBashReadOnly の readOnlyCommands よりも保守的（確認UI なし + 副作用なし）。
var bashParallelSafeCommands = []string{
	"pwd",
	"ls",
	"find",
	"rg",
	"grep",
	"cat",
	"head",
	"tail",
	"wc",
	"git status",
	"git diff",
	"git diff --stat",
	"git log",
	"git log --oneline",
	"git branch",
	"git ls-files",
	"git show",
	"git remote",
}

// ToolExecResult は並列実行における1つのツール実行結果を保持する。
type ToolExecResult struct {
	Index  int         // 元の toolCalls スライス内のインデックス
	Result string      // ツール実行結果
	Change *FileChange // ファイル変更情報（nil の場合あり）
	TC     *ToolCall   // 実行した ToolCall への参照
	Err    error       // executor レベルのエラー（通常は nil）
}

// ClassifyToolCalls は toolCalls を parallel-safe 群と sequential 群に分類する。
// 返り値: parallelIndices（parallel-safe な元インデックス群）, sequentialIndices（sequential な元インデックス群）
func ClassifyToolCalls(toolCalls []*ToolCall) (parallelIndices, sequentialIndices []int) {
	for i, tc := range toolCalls {
		// NormalizeArgs が未実行の場合に備え、RawArgs から判定
		if IsParallelSafe(tc) {
			parallelIndices = append(parallelIndices, i)
		} else {
			sequentialIndices = append(sequentialIndices, i)
		}
	}
	return
}

// ExecuteToolCallsParallel は parallel-safe な tool を並列実行し、sequential な tool を順次実行する。
// 結果は元の toolCalls 順で返る。
//
// execFn: 各 ToolCall を実行する関数（spinner・キャッシュ等の処理を呼び出し元が注入）
// ctx: 親コンテキスト（cancel/timeout を各 goroutine に伝播）
//
// mixed case では parallel-safe 群を先に並列実行し、完了後に sequential 群を順次実行する。
func ExecuteToolCallsParallel(
	ctx context.Context,
	toolCalls []*ToolCall,
	execFn func(ctx context.Context, tc *ToolCall) (string, *FileChange),
) []ToolExecResult {
	n := len(toolCalls)
	if n == 0 {
		return nil
	}

	results := make([]ToolExecResult, n)
	for i, tc := range toolCalls {
		results[i] = ToolExecResult{Index: i, TC: tc}
	}

	parallelIndices, sequentialIndices := ClassifyToolCalls(toolCalls)

	// Phase 1: parallel-safe 群を並列実行
	if len(parallelIndices) > 0 {
		sem := make(chan struct{}, MaxParallelTools)
		var wg sync.WaitGroup

		for _, idx := range parallelIndices {
			// コンテキストがキャンセル済みならスキップ
			if ctx.Err() != nil {
				results[idx].Result = "Error: context cancelled"
				results[idx].Err = ctx.Err()
				continue
			}

			wg.Add(1)
			sem <- struct{}{} // セマフォ取得
			go func(i int) {
				defer wg.Done()
				defer func() { <-sem }() // セマフォ解放

				tc := toolCalls[i]
				result, change := execFn(ctx, tc)
				results[i].Result = result
				results[i].Change = change
			}(idx)
		}
		wg.Wait()
	}

	// Phase 2: sequential 群を順次実行
	for _, idx := range sequentialIndices {
		if ctx.Err() != nil {
			results[idx].Result = "Error: context cancelled"
			results[idx].Err = ctx.Err()
			continue
		}

		tc := toolCalls[idx]
		result, change := execFn(ctx, tc)
		results[idx].Result = result
		results[idx].Change = change
	}

	return results
}
