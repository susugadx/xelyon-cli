package toolruntime

import (
	"encoding/json"
	"regexp"
	"sort"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/tools"
)

// ── read_file batching eligibility ──

// IsBatchableReadFile は read_file call が batch 対象（detail=auto の range なし read）かを判定する。
// targeted read と内部 batch call は対象外。
func IsBatchableReadFile(tc *tools.ToolCall) bool {
	if tc.Tool != "read_file" {
		return false
	}
	if strings.EqualFold(tc.Args["_full_budget"], "true") {
		return false
	}
	detail := strings.TrimSpace(strings.ToLower(tc.Args["detail"]))
	if detail != "" && detail != "auto" {
		return false
	}
	if ReadFileHasExplicitRange(tc.Args) {
		return false
	}
	return len(ReadFilePathsFromArgs(tc.Args)) > 0
}

// ── read_file batch merge ──

// MaxReadFileBatchPaths は read_file batch merge のパス上限。
const MaxReadFileBatchPaths = 10

// BuildReadFileBatchToolCall は internal batch read 用の synthetic ToolCall を生成する。
// observability / negative cache 記録のため paths を保持し、internal 用に _full_budget も付与する。
func BuildReadFileBatchToolCall(paths []string, fullBudget bool) *tools.ToolCall {
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

// ReadBatchSegment は変更系ツール境界間の batchable read_file 群を表す。
type ReadBatchSegment struct {
	Indices    []int
	Paths      []string
	PathCounts []int
}

// SegmentReadFileBatches は tool call リストからバッチ可能な read_file のセグメントを収集する。
// 変更系ツール（非 parallel-safe）で区切り、各セグメント内の batchable read_file を返す。
// execFlags[i] == true のエントリのみ対象。
func SegmentReadFileBatches(allToolCalls []*tools.ToolCall, execFlags []bool) []ReadBatchSegment {
	var segments []ReadBatchSegment
	var current ReadBatchSegment

	for i, tc := range allToolCalls {
		if !execFlags[i] {
			continue
		}
		if !tools.IsParallelSafe(tc) {
			// 変更系ツール: 現在のセグメントを確定してリセット
			if len(current.Indices) >= 2 {
				segments = append(segments, current)
			}
			current = ReadBatchSegment{}
			continue
		}
		if !IsBatchableReadFile(tc) {
			continue
		}
		paths := ReadFilePathsFromArgs(tc.Args)
		if len(paths) == 0 {
			continue
		}
		current.Indices = append(current.Indices, i)
		current.Paths = append(current.Paths, paths...)
		current.PathCounts = append(current.PathCounts, len(paths))
	}
	// 最後のセグメント（上限チェックは実行時に chunk 分割する）
	if len(current.Indices) >= 2 {
		segments = append(segments, current)
	}
	return segments
}

// ── search_code multi-pattern batching ──

// MaxSearchBatchPatterns は search_code batch のパターン上限。
// search_code の multi-pattern は最大 5 パターン。
const MaxSearchBatchPatterns = 5

// SearchCodeOptionsKey は search_code call の pattern 以外の option をキーとして返す。
// option が全て同一の call 同士が multi-pattern 化の候補になる。
// 空値の option はキーに含めない（未指定扱い）。
func SearchCodeOptionsKey(tc *tools.ToolCall) string {
	if tc.Tool != "search_code" {
		return ""
	}
	mode := strings.TrimSpace(strings.ToLower(tc.Args["mode"]))
	if mode == "" {
		if legacy, ok := tc.Args["is_regex"]; ok && legacy != "" {
			if strings.EqualFold(legacy, "true") {
				mode = "regex"
			} else {
				mode = "literal"
			}
		} else {
			mode = "auto"
		}
	}

	var parts []string
	for k, v := range tc.Args {
		if k == "pattern" {
			continue
		}
		if k == "mode" || k == "is_regex" {
			continue
		}
		if v == "" {
			continue
		}
		parts = append(parts, k+"="+v)
	}
	parts = append(parts, "mode="+mode)
	sort.Strings(parts)
	return strings.Join(parts, "&")
}

// IsSimpleSearchPattern は pattern が単一パターン（カンマ区切りでない）かを返す。
// 既に multi-pattern の call を更に merge すると予期しないパターン分割が起きるため、
// 単一パターンのみ batch 対象とする。
func IsSimpleSearchPattern(pattern string) bool {
	// \, はエスケープされたリテラルカンマ → multi-pattern ではない
	temp := strings.ReplaceAll(pattern, `\,`, "")
	return !strings.Contains(temp, ",")
}

// CloneToolCallWithNewPattern は tc の shallow copy を作り pattern を差し替える。
// search_code batch で merged pattern を持つ ToolCall を生成するために使う。
func CloneToolCallWithNewPattern(tc *tools.ToolCall, newPattern string) *tools.ToolCall {
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

// SplitMultiPatternResult は multi-pattern search 結果をパターンごとに分割する。
// patterns は merge 時の順序と一致していなければならない。
// 分割に失敗した場合（ヘッダー数不一致等）は nil を返す。
//
// multi-pattern 結果は末尾に lineRangeHint を 1 回だけ付与する仕様のため、
// 分割後の各 section にも Tip を付与して single-call 契約を維持する。
func SplitMultiPatternResult(result string, patterns []string) map[string]string {
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
