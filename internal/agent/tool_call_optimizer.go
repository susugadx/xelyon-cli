package agent

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/tools"
)

// ── read_file batching eligibility ──

// isBatchableReadFile は read_file call が batch 対象（range なし read）かを判定する。
// targeted read と内部 batch call は対象外。
func isBatchableReadFile(tc *tools.ToolCall) bool {
	if tc.Tool != "read_file" {
		return false
	}
	if strings.EqualFold(tc.Args["_full_budget"], "true") {
		return false
	}
	if readFileHasExplicitRange(tc.Args) {
		return false
	}
	return len(readFilePathsFromArgs(tc.Args)) > 0
}

// ── read_file batch merge ──

// maxReadFileBatchPaths は read_file batch merge のパス上限。
const maxReadFileBatchPaths = 10

// buildReadFileBatchToolCall は internal batch read 用の synthetic ToolCall を生成する。
// observability / negative cache 記録のため paths を保持し、internal 用に _full_budget も付与する。
func buildReadFileBatchToolCall(paths []string, fullBudget bool) *tools.ToolCall {
	pathsJSON, _ := json.Marshal(paths)
	args := map[string]string{
		"paths": string(pathsJSON),
	}
	rawArgs := map[string]any{
		"paths": paths,
	}
	if fullBudget {
		args["_full_budget"] = "true"
		rawArgs["_full_budget"] = true
	}
	return &tools.ToolCall{
		Tool:    "read_file",
		Args:    args,
		RawArgs: rawArgs,
	}
}

// readBatchSegment は変更系ツール境界間の batchable read_file 群を表す。
type readBatchSegment struct {
	indices    []int
	paths      []string
	pathCounts []int
}

// segmentReadFileBatches は tool call リストからバッチ可能な read_file のセグメントを収集する。
// 変更系ツール（非 parallel-safe）で区切り、各セグメント内の batchable read_file を返す。
// execFlags[i] == true のエントリのみ対象。
func segmentReadFileBatches(allToolCalls []*tools.ToolCall, execFlags []bool) []readBatchSegment {
	var segments []readBatchSegment
	var current readBatchSegment

	for i, tc := range allToolCalls {
		if !execFlags[i] {
			continue
		}
		if !tools.IsParallelSafe(tc) {
			// 変更系ツール: 現在のセグメントを確定してリセット
			if len(current.indices) >= 2 {
				segments = append(segments, current)
			}
			current = readBatchSegment{}
			continue
		}
		if !isBatchableReadFile(tc) {
			continue
		}
		paths := readFilePathsFromArgs(tc.Args)
		if len(paths) == 0 {
			continue
		}
		current.indices = append(current.indices, i)
		current.paths = append(current.paths, paths...)
		current.pathCounts = append(current.pathCounts, len(paths))
	}
	// 最後のセグメント（上限チェックは実行時に chunk 分割する）
	if len(current.indices) >= 2 {
		segments = append(segments, current)
	}
	return segments
}

// readFileBatchHeaderPrefix は internal batch read の結果でファイル区切りに使われるプレフィックス。
const readFileBatchHeaderPrefix = "📄 File: "

// splitReadFileBatchResult は internal batch read の結果をファイルパスごとに分割する。
// paths は merge 時の順序と一致していなければならない。
// 分割に失敗した場合は nil を返す。
func splitReadFileBatchResult(result string, paths []string) map[string]string {
	// "📄 File: <path>\n" ヘッダーの位置を検出
	type headerLoc struct {
		path       string
		contentIdx int // ヘッダー行末（コンテンツ開始）の位置
		headerIdx  int // ヘッダー行先頭の位置
	}

	var locs []headerLoc
	searchFrom := 0
	for _, p := range paths {
		header := readFileBatchHeaderPrefix + p + "\n"
		idx := strings.Index(result[searchFrom:], header)
		if idx < 0 {
			// ヘッダーが見つからない → 安全に分割できない
			return nil
		}
		absIdx := searchFrom + idx
		locs = append(locs, headerLoc{
			path:       p,
			headerIdx:  absIdx,
			contentIdx: absIdx + len(header),
		})
		searchFrom = absIdx + len(header)
	}

	if len(locs) != len(paths) {
		return nil
	}

	sections := make(map[string]string, len(paths))
	for i, loc := range locs {
		var contentEnd int
		if i+1 < len(locs) {
			// 次のヘッダーの前の空行を除去
			contentEnd = locs[i+1].headerIdx
			// ファイル間の区切り改行を除去
			for contentEnd > loc.contentIdx && result[contentEnd-1] == '\n' {
				contentEnd--
			}
		} else {
			contentEnd = len(result)
		}
		section := strings.TrimRight(result[loc.contentIdx:contentEnd], "\n ")
		sections[loc.path] = section
	}

	return sections
}

func joinReadFileBatchSections(perFile map[string]string, paths []string) (string, bool) {
	var sb strings.Builder
	for i, path := range paths {
		section, ok := perFile[path]
		if !ok {
			return "", false
		}
		if i > 0 {
			sb.WriteString("\n")
		}
		fmt.Fprintf(&sb, "%s%s\n", readFileBatchHeaderPrefix, path)
		sb.WriteString(section)
	}
	return sb.String(), true
}

// ── search_code multi-pattern batching ──

// maxSearchBatchPatterns は search_code batch のパターン上限。
// search_code の multi-pattern は最大 5 パターン。
const maxSearchBatchPatterns = 5

// searchCodeOptionsKey は search_code call の pattern 以外の option をキーとして返す。
// option が全て同一の call 同士が multi-pattern 化の候補になる。
// 空値の option はキーに含めない（未指定扱い）。
func searchCodeOptionsKey(tc *tools.ToolCall) string {
	if tc.Tool != "search_code" {
		return ""
	}
	var parts []string
	for k, v := range tc.Args {
		if k == "pattern" {
			continue
		}
		if v == "" {
			continue
		}
		parts = append(parts, k+"="+v)
	}
	sort.Strings(parts)
	return strings.Join(parts, "&")
}

// isSimpleSearchPattern は pattern が単一パターン（カンマ区切りでない）かを返す。
// 既に multi-pattern の call を更に merge すると予期しないパターン分割が起きるため、
// 単一パターンのみ batch 対象とする。
func isSimpleSearchPattern(pattern string) bool {
	// \, はエスケープされたリテラルカンマ → multi-pattern ではない
	temp := strings.ReplaceAll(pattern, `\,`, "")
	return !strings.Contains(temp, ",")
}

// cloneToolCallWithNewPattern は tc の shallow copy を作り pattern を差し替える。
// search_code batch で merged pattern を持つ ToolCall を生成するために使う。
func cloneToolCallWithNewPattern(tc *tools.ToolCall, newPattern string) *tools.ToolCall {
	newArgs := make(map[string]string, len(tc.Args))
	for k, v := range tc.Args {
		newArgs[k] = v
	}
	newArgs["pattern"] = newPattern

	newRawArgs := make(map[string]any, len(tc.RawArgs))
	for k, v := range tc.RawArgs {
		newRawArgs[k] = v
	}
	newRawArgs["pattern"] = newPattern

	return &tools.ToolCall{
		Tool:    tc.Tool,
		Args:    newArgs,
		RawArgs: newRawArgs,
	}
}

// multiPatternHeaderRe は multi-pattern search 結果のパターン区切りヘッダーにマッチする。
// format: ━━ Pattern 1/3: "pattern" ━━
var multiPatternHeaderRe = regexp.MustCompile(`━━ Pattern \d+/\d+: ".*?" ━━`)

// searchCodeTipPrefix は search_code 結果末尾の Tip 行を検出するためのプレフィックス。
// search_code.go の lineRangeHint と対応する。
const searchCodeTipPrefix = "\n\nTip: "

// splitMultiPatternResult は multi-pattern search 結果をパターンごとに分割する。
// patterns は merge 時の順序と一致していなければならない。
// 分割に失敗した場合（ヘッダー数不一致等）は nil を返す。
//
// multi-pattern 結果は末尾に lineRangeHint を 1 回だけ付与する仕様のため、
// 分割後の各 section にも Tip を付与して single-call 契約を維持する。
func splitMultiPatternResult(result string, patterns []string) map[string]string {
	// Tip 行を検出して分離（末尾の共通 hint）
	trailingHint := ""
	if tipIdx := strings.LastIndex(result, searchCodeTipPrefix); tipIdx >= 0 {
		trailingHint = result[tipIdx:]
		result = result[:tipIdx]
	}

	locs := multiPatternHeaderRe.FindAllStringIndex(result, -1)
	if len(locs) != len(patterns) {
		// ヘッダー数が期待パターン数と一致しない → 安全に分割できない
		return nil
	}

	sections := make(map[string]string, len(patterns))
	for i, loc := range locs {
		// section: header 行直後から次の header（or 末尾）まで
		sectionStart := loc[1]
		if sectionStart < len(result) && result[sectionStart] == '\n' {
			sectionStart++
		}

		var sectionEnd int
		if i+1 < len(locs) {
			sectionEnd = locs[i+1][0]
		} else {
			sectionEnd = len(result)
		}

		section := strings.TrimRight(result[sectionStart:sectionEnd], "\n ")
		// 各 section に Tip を付与して single-call 契約を維持
		if trailingHint != "" && section != "" &&
			!strings.HasPrefix(section, "⚠️ Error:") &&
			!isNoMatchesSearchSection(section) {
			section += trailingHint
		}
		sections[patterns[i]] = section
	}

	return sections
}

// isNoMatchesSearchSection reports whether a split search_code section is a
// no-match result. Warning lines may precede the final "No matches found".
func isNoMatchesSearchSection(section string) bool {
	lines := strings.Split(section, "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		trimmed := strings.TrimSpace(lines[i])
		if trimmed == "" {
			continue
		}
		return trimmed == "No matches found"
	}
	return false
}
