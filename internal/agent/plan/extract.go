package plan

import "strings"

const jsonCodeFence = "```json"

// ContainsPlanJSONForNormalModeRecovery は normal mode の direct-execution recovery 対象になる
// Plan JSON がレスポンスに含まれるかを判定する。
func ContainsPlanJSONForNormalModeRecovery(response string) bool {
	return ExtractPlanJSONForNormalModeRecovery(response) != ""
}

// ExtractPlanJSON はレスポンスからPlan JSONを抽出
// 見つからない場合は空文字列を返す
// NOTE: ツール呼び出し JSON ({"tool": ...}) は plan ではないので除外する
func ExtractPlanJSON(response string) string {
	if jsonStr := findPlanWrapperJSON(response); jsonStr != "" {
		return jsonStr
	}

	if jsonStr := findCodeBlockPlanJSON(response); jsonStr != "" {
		return jsonStr
	}

	return findLegacyPlanJSON(response)
}

// ExtractPlanJSONForNormalModeRecovery は normal mode でモデルが Plan JSON へ逸れた時の
// recovery 対象だけを抽出する。Plan Mode の修復用抽出より高精度にし、通常のユーザー要求 JSON を
// direct-execution recovery へ流さない。
func ExtractPlanJSONForNormalModeRecovery(response string) string {
	return findPlanWrapperJSON(response)
}

func findPlanWrapperJSON(response string) string {
	return findScopedPlanJSON(response, planJSONCandidateWrapper)
}

func findLegacyPlanJSON(response string) string {
	return findScopedPlanJSON(response, planJSONCandidateLegacy)
}

func findScopedPlanJSON(response string, scope planJSONCandidateScope) string {
	return findJSONObjectMatching(response, func(jsonStr string) bool {
		return isPlanJSONCandidateForScope(jsonStr, scope)
	})
}

func findCodeBlockPlanJSON(response string) string {
	idx := strings.Index(response, jsonCodeFence)
	if idx == -1 {
		return ""
	}

	start := strings.Index(response[idx:], "{")
	if start == -1 {
		return ""
	}
	start += idx

	end := findClosingBrace(response, start)
	if end == -1 {
		return ""
	}

	jsonStr := response[start:end]
	if !isPlanJSONCandidate(jsonStr) {
		return ""
	}
	return jsonStr
}

func findJSONObjectMatching(response string, match func(string) bool) string {
	for idx := 0; idx < len(response); {
		start := strings.Index(response[idx:], "{")
		if start == -1 {
			return ""
		}
		start += idx
		end := findClosingBrace(response, start)
		if end == -1 {
			idx = start + 1
			continue
		}

		jsonStr := response[start:end]
		if match(jsonStr) {
			return jsonStr
		}
		idx = end
	}
	return ""
}
