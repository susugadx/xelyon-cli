package agent

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/tools"
)

// ── Same-turn duplicate suppression ──
//
// 同一レスポンス内で完全に同じ tool call（tool名 + 全引数が一致）が
// 複数回含まれる場合、2回目以降を canonical メッセージに置き換える。
// read-only かつ冪等なツールのみ対象。write ツールは状態変更するため除外。

// sameTurnDupMsgFmt は同一ターン内の重複 tool call に対する canonical 結果。
// モデルが「前の結果を参照すべき」と分かる安定文言にする。
const sameTurnDupMsgFmt = "[Duplicate] Same %s call already executed in this turn; reuse previous result."

// sameTurnDuplicateMessage は同一ターン内の重複 tool call の canonical 結果を返す。
func sameTurnDuplicateMessage(toolName string) string {
	return fmt.Sprintf(sameTurnDupMsgFmt, toolName)
}

// isDedupableForTurn は同一ターン内の重複排除対象ツールかを返す。
// read-only かつ冪等なツールのみ対象。
func isDedupableForTurn(toolName string) bool {
	switch toolName {
	case "read_file", "search_code", "list_dir", "inspect_symbol":
		return true
	}
	return false
}

// turnDedupKey は tool call の同一ターン内重複判定用キーを生成する。
// tool名 + sorted args JSON。Go の json.Marshal は map キーをアルファベット順に
// ソートするため、同一引数は同一キーになる。
func turnDedupKey(tc *tools.ToolCall) string {
	argsJSON, _ := json.Marshal(tc.Args)
	return tc.Tool + ":" + string(argsJSON)
}

// ── read_file batching eligibility ──

// isBatchableReadFile は read_file call が batch 対象（plain path read）かを判定する。
// symbol read / range read / paths 指定済みの call は対象外。
func isBatchableReadFile(tc *tools.ToolCall) bool {
	if tc.Tool != "read_file" {
		return false
	}
	if tc.Args["symbol"] != "" {
		return false
	}
	if tc.Args["start_line"] != "" || tc.Args["end_line"] != "" {
		return false
	}
	if tc.Args["paths"] != "" {
		return false
	}
	return tc.Args["path"] != ""
}

// ── read_file batch merge ──

// maxReadFileBatchPaths は read_file batch merge のパス上限。
// read_file(paths=...) の MaxReadFilesPaths に合わせる。
const maxReadFileBatchPaths = 10

// buildReadFileBatchToolCall は複数の plain read_file call を
// 1 つの read_file(paths=[...]) call にまとめた ToolCall を生成する。
func buildReadFileBatchToolCall(paths []string) *tools.ToolCall {
	pathsJSON, _ := json.Marshal(paths)
	return &tools.ToolCall{
		Tool: "read_file",
		Args: map[string]string{
			"paths": string(pathsJSON),
		},
		RawArgs: map[string]any{
			"paths": paths,
		},
	}
}

// readFileBatchHeaderPrefix は read_file(paths=...) の結果でファイル区切りに使われるプレフィックス。
const readFileBatchHeaderPrefix = "📄 File: "

// splitReadFileBatchResult は read_file(paths=...) の結果をファイルパスごとに分割する。
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
