package file

import (
	"fmt"
	"strconv"
	"strings"
	"sync"

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

// ReadFilesTotalBudget は read_files 全体の総行数バジェット
const ReadFilesTotalBudget = 500

// perFileBudget はファイル数に応じた1ファイルあたりのアウトライン閾値を返す。
// totalBudget / n で算出し、DefaultFullLines を上限、30 を下限とする。
func perFileBudget(n int) int {
	b := ReadFilesTotalBudget / n
	if b > DefaultFullLines {
		return DefaultFullLines
	}
	if b < 30 {
		return 30
	}
	return b
}

// ExecuteReadFiles は複数ファイルを一括読み込みする
func ExecuteReadFiles(paths []string) string {
	return ExecuteReadFilesWithOutput(common.DefaultOutput(), paths)
}

// ExecuteReadFilesWithOutput は出力先を指定して複数ファイルを一括読み込みする。
func ExecuteReadFilesWithOutput(out common.Output, paths []string) string {
	if len(paths) == 0 {
		return "Error: paths is empty"
	}
	if len(paths) > MaxReadFilesPaths {
		return fmt.Sprintf("Error: too many paths (max %d), got %d", MaxReadFilesPaths, len(paths))
	}

	budget := perFileBudget(len(paths))
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
				// 明示的な行範囲指定: そのまま読み込み
				result = executeReadFileCore(out, nil, nil, path, startLine, endLine, DefaultFullLines)
			} else {
				// 行範囲なし: budget をアウトライン閾値として適用
				result = executeReadFileCore(out, nil, nil, path, 0, 0, budget)
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
