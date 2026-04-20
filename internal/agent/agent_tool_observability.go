package agent

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"github.com/susugadx/xelyon-cli/internal/tools"
)

var readFileOutlineFooterPattern = regexp.MustCompile(`\(\d+ lines total(?:\.[^)]*)?\)`)

func isReadFileOutlineResult(result string) bool {
	return readFileOutlineFooterPattern.MatchString(result)
}

func (a *Agent) recordToolResultOptimizations(tc *tools.ToolCall, result string) {
	if tc.Tool == "read_file" && isReadFileOutlineResult(result) {
		a.addOptimizationMetrics(OptimizationMetrics{OutlineFirstCount: 1})
	}
	a.recordToolObservability(tc.Tool, tc.RawArgs, tc.Args, result)
}

// recordToolObservability はツール実行のobservabilityメトリクスを記録する。
// rawArgs は FC 経由の場合に map[string]any、XML rescue 経由では nil になるため、
// stringArgs（ToolCall.Args）をフォールバックとして参照する。
func (a *Agent) recordToolObservability(toolName string, rawArgs map[string]any, stringArgs map[string]string, result string) {
	a.statsMu.Lock()
	defer a.statsMu.Unlock()
	if a.Stats == nil {
		return
	}
	obs := &a.Stats.ToolObs

	switch toolName {
	case "read_file":
		// batch read: paths 引数が2つ以上の有効パスを持つ場合
		if pathsVal, ok := rawArgs["paths"]; ok {
			if isBatchPaths(pathsVal) {
				obs.ReadFileBatchCalls++
			}
		} else if pathsStr, ok := stringArgs["paths"]; ok {
			// XML rescue フォールバック: Args["paths"] は JSON 文字列
			if isBatchPaths(pathsStr) {
				obs.ReadFileBatchCalls++
			}
		}
		if hasReadFileTargets(rawArgs, stringArgs) {
			obs.ReadFileTargetCalls++
		}
		// empty-path error: canonical error string を検出
		if result == "Error: paths is empty" {
			obs.ReadFileEmptyPathsErrors++
		}
	case "search_code":
		if isSearchCodeImpact(rawArgs, stringArgs) {
			obs.SearchCodeImpactCalls++
			return
		}
		// explicit multi-pattern only. impact は独立メトリクスとして扱う。
		if isObservedSearchCodeExplicitMulti(rawArgs, stringArgs) {
			obs.SearchCodeExplicitMultiCalls++
			obs.SearchCodeMultiPatternCalls++
			return
		}
		a.recordSearchCodeMissedMultiLocked(rawArgs, stringArgs)
	}
}

func (a *Agent) resetSearchCodeTurnObservability() {
	a.statsMu.Lock()
	defer a.statsMu.Unlock()
	a.searchCodeRecentSinglePatternByFamily = make(map[string]string)
	a.searchCodeMissedMultiCountedFamilies = make(map[string]struct{})
}

func (a *Agent) recordSearchCodeMissedMultiLocked(rawArgs map[string]any, stringArgs map[string]string) {
	args := buildSearchCodeObservabilityArgs(rawArgs, stringArgs)
	pattern := strings.TrimSpace(args["pattern"])
	if pattern == "" || isMultiPatternArg(pattern) {
		return
	}

	exactPattern := normalizeSearchCodeObservedPattern(pattern, false)
	familyPattern := normalizeSearchCodeObservedPattern(pattern, true)
	if exactPattern == "" || familyPattern == "" {
		return
	}

	optionsKey := searchCodeOptionsKey(&tools.ToolCall{
		Tool: "search_code",
		Args: args,
	})
	if optionsKey == "" {
		return
	}

	if a.searchCodeRecentSinglePatternByFamily == nil {
		a.searchCodeRecentSinglePatternByFamily = make(map[string]string)
	}
	if a.searchCodeMissedMultiCountedFamilies == nil {
		a.searchCodeMissedMultiCountedFamilies = make(map[string]struct{})
	}

	familyKey := optionsKey + "|family=" + familyPattern
	if _, counted := a.searchCodeMissedMultiCountedFamilies[familyKey]; counted {
		return
	}

	prevPattern, seen := a.searchCodeRecentSinglePatternByFamily[familyKey]
	if !seen {
		a.searchCodeRecentSinglePatternByFamily[familyKey] = exactPattern
		return
	}
	if prevPattern == exactPattern {
		return
	}

	a.Stats.ToolObs.SearchCodeMissedMultiPattern++
	a.searchCodeMissedMultiCountedFamilies[familyKey] = struct{}{}
}

func buildSearchCodeObservabilityArgs(rawArgs map[string]any, stringArgs map[string]string) map[string]string {
	args := make(map[string]string, len(stringArgs)+len(rawArgs))
	for k, v := range stringArgs {
		args[k] = v
	}
	for k, v := range rawArgs {
		if s, ok := stringifySearchCodeArg(v); ok {
			args[k] = s
		}
	}
	return args
}

func isSearchCodeImpact(rawArgs map[string]any, stringArgs map[string]string) bool {
	args := buildSearchCodeObservabilityArgs(rawArgs, stringArgs)
	return strings.EqualFold(strings.TrimSpace(args["intent"]), "impact")
}

func isObservedSearchCodeExplicitMulti(rawArgs map[string]any, stringArgs map[string]string) bool {
	args := buildSearchCodeObservabilityArgs(rawArgs, stringArgs)
	pattern := strings.TrimSpace(args["pattern"])
	if pattern == "" {
		return false
	}
	return isMultiPatternArg(pattern)
}

func stringifySearchCodeArg(v any) (string, bool) {
	switch val := v.(type) {
	case string:
		return val, true
	case bool:
		if val {
			return "true", true
		}
		return "false", true
	case int:
		return fmt.Sprintf("%d", val), true
	case int64:
		return fmt.Sprintf("%d", val), true
	case float64:
		return fmt.Sprintf("%g", val), true
	default:
		return "", false
	}
}

func normalizeSearchCodeObservedPattern(pattern string, stripFamilyNoise bool) string {
	tokens := tokenizeObservedSearchCodePattern(pattern)
	if len(tokens) == 0 {
		return ""
	}
	if !stripFamilyNoise {
		return strings.Join(tokens, " ")
	}

	filtered := make([]string, 0, len(tokens))
	for _, token := range tokens {
		if _, noise := searchCodeFamilyNoiseTerms[token]; noise {
			continue
		}
		filtered = append(filtered, token)
	}
	if len(filtered) == 0 {
		return ""
	}
	return strings.Join(filtered, " ")
}

func tokenizeObservedSearchCodePattern(pattern string) []string {
	return strings.FieldsFunc(strings.ToLower(strings.TrimSpace(pattern)), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
}

var searchCodeFamilyNoiseTerms = map[string]struct{}{
	"definition":     {},
	"definitions":    {},
	"caller":         {},
	"callers":        {},
	"ref":            {},
	"refs":           {},
	"reference":      {},
	"references":     {},
	"test":           {},
	"tests":          {},
	"impl":           {},
	"implementation": {},
}

// isBatchPaths は paths 引数が実質的な batch（2パス以上）か判定する。
// XML rescue 経由ではタグ内に前後空白・改行が含まれるため TrimSpace してから判定する。
func isBatchPaths(pathsVal any) bool {
	switch v := pathsVal.(type) {
	case []any:
		return len(v) >= 2
	case string:
		// JSON 文字列の場合: "[" で始まりカンマを含む → 2要素以上
		s := strings.TrimSpace(v)
		return len(s) > 2 && s[0] == '[' && strings.Contains(s, ",")
	}
	return false
}

func hasReadFileTargets(rawArgs map[string]any, stringArgs map[string]string) bool {
	if targetsVal, ok := rawArgs["targets"]; ok {
		if targets, ok := stringifySearchCodeArg(targetsVal); ok {
			return strings.TrimSpace(targets) != ""
		}
	}
	if targets, ok := stringArgs["targets"]; ok {
		return strings.TrimSpace(targets) != ""
	}
	return false
}

// isMultiPatternArg は pattern 引数が multi-pattern（カンマ区切り2パターン以上）か判定する。
// search_code の splitPatterns と同じロジック: \, はリテラルカンマとして除外。
func isMultiPatternArg(patternVal any) bool {
	s, ok := patternVal.(string)
	if !ok || s == "" {
		return false
	}
	// \, をプレースホルダーに置換してからカンマでsplit
	replaced := strings.ReplaceAll(s, `\,`, "\x00")
	parts := strings.Split(replaced, ",")
	count := 0
	for _, p := range parts {
		if strings.TrimSpace(p) != "" {
			count++
		}
	}
	return count >= 2
}
