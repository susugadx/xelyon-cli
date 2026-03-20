package file

import (
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/tools/common"
)

// MaxReadFilesPaths は一度に読み込めるファイル数の上限
const MaxReadFilesPaths = 10

// parsePath はパス文字列を解析し、ファイルパスと行範囲を返す
// "path" → path, 0, 0
// "path:10" → path, 10, 0
// "path:10-20" → path, 10, 20
func parsePath(entry string) (string, int, int) {
	// 最後のコロンを探す（Windows パス C:\... 対策で lastIndex を使用）
	lastColon := strings.LastIndex(entry, ":")
	if lastColon < 0 {
		return entry, 0, 0
	}

	// コロン以降が数字または数字-数字でなければパスとして扱う
	suffix := entry[lastColon+1:]
	path := entry[:lastColon]

	// "start-end" 形式
	if dashIdx := strings.Index(suffix, "-"); dashIdx >= 0 {
		startStr := suffix[:dashIdx]
		endStr := suffix[dashIdx+1:]
		start, err1 := strconv.Atoi(startStr)
		end, err2 := strconv.Atoi(endStr)
		if err1 == nil && err2 == nil && start > 0 && end > 0 {
			return path, start, end
		}
		// パースできなければ全体をパスとして扱う
		return entry, 0, 0
	}

	// "start" のみ
	start, err := strconv.Atoi(suffix)
	if err == nil && start > 0 {
		return path, start, 0
	}

	// パースできなければ全体をパスとして扱う
	return entry, 0, 0
}

// ExecuteReadFiles は複数ファイルを一括読み込みする
func ExecuteReadFiles(paths []string) string {
	return ExecuteReadFilesWithOutput(common.DefaultOutput(), paths)
}

// ExecuteReadFilesWithOutput は出力先を指定して複数ファイルを一括読み込みする。
func ExecuteReadFilesWithOutput(out common.Output, paths []string) string {
	return ExecuteReadFilesWithBudget(out, paths, 0)
}

// ExecuteReadFilesWithRuntime は config/cache を指定して複数ファイルを一括読み込みする。
// 自動 batch merge で使用し、単発 read_file と同じ runtime 文脈（config 反映、cache 反映）を維持する。
func ExecuteReadFilesWithRuntime(out common.Output, cfg *config.Config, cache tools.ToolCacheInterface, paths []string, budgetOverride int) string {
	return executeReadFilesCore(out, cfg, cache, paths, budgetOverride)
}

// ExecuteReadFilesWithBudget は出力先とファイルあたりのアウトライン閾値を指定して
// 複数ファイルを一括読み込みする。budgetOverride が 0 の場合は DefaultFullLines を使用する。
// 自動 batch merge では DefaultFullLines を渡し、単発 read と同等の閾値を維持する。
func ExecuteReadFilesWithBudget(out common.Output, paths []string, budgetOverride int) string {
	return executeReadFilesCore(out, nil, nil, paths, budgetOverride)
}

// executeReadFilesCore は複数ファイル読み込みの内部実装。
// cfg/cache が nil の場合は単発 read_file のフォールバック動作（設定なし）を使用する。
func executeReadFilesCore(out common.Output, cfg *config.Config, cache tools.ToolCacheInterface, paths []string, budgetOverride int) string {
	if len(paths) == 0 {
		return "Error: paths is empty"
	}
	if len(paths) > MaxReadFilesPaths {
		return fmt.Sprintf("Error: too many paths (max %d), got %d", MaxReadFilesPaths, len(paths))
	}

	budget := budgetOverride
	if budget <= 0 {
		budget = DefaultFullLines
	}
	sem := make(chan struct{}, tools.MaxParallelTools)

	type readResult struct {
		entry  string
		result string
	}
	results := make([]readResult, len(paths))
	var wg sync.WaitGroup

	for i, entry := range paths {
		wg.Add(1)
		sem <- struct{}{}
		go func(idx int, rawEntry string) {
			defer wg.Done()
			defer func() { <-sem }()

			path, startLine, endLine := parsePath(rawEntry)

			var result string
			if startLine > 0 || endLine > 0 {
				result = executeReadFileCore(out, cfg, cache, path, startLine, endLine, DefaultFullLines)
			} else {
				result = executeReadFileCore(out, cfg, cache, path, 0, 0, budget)
			}

			results[idx] = readResult{
				entry:  rawEntry,
				result: result,
			}
		}(i, entry)
	}
	wg.Wait()

	var sb strings.Builder
	for i, result := range results {
		if i > 0 {
			sb.WriteString("\n")
		}
		fmt.Fprintf(&sb, "📄 File: %s\n", result.entry)
		sb.WriteString(result.result)
	}

	printReadStatus(out, "📄 Read: %d files\n", len(paths))
	return sb.String()
}
