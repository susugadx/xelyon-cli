package skills

import (
	"strings"
)

// skillKeywords maps skill names to trigger keywords.
var skillKeywords = map[string][]string{
	"ci": {
		"CI", "Actions", "workflow", "gh run", "lint", "pipeline", "build", "failed",
		"ビルド", "ワークフロー", "パイプライン", "リント",
		"通らない", "落ちた", "失敗",
	},
	"github": {
		"PR", "Issue", "pull request", "review", "approve", "merge", "label",
		"プルリク", "プルリクエスト", "イシュー", "レビュー", "マージ", "ラベル", "アプルーブ",
	},
	"git": {
		"git", "commit", "push", "pull", "branch", "merge", "rebase", "stash",
		"checkout", "reset", "diff", "log", "clone", "fetch", "cherry-pick",
		"コミット", "プッシュ", "プル", "ブランチ", "マージ", "リベース", "スタッシュ",
		"チェックアウト", "リセット", "クローン", "フェッチ",
		"履歴", "戻す", "取り消し",
	},
	"testing": {
		"test", "coverage", "pytest", "jest", "rspec", "spec", "assert", "mock",
		"vitest", "unittest", "phpunit", "junit",
		"テスト", "カバレッジ", "アサート", "モック",
		"単体", "結合", "E2E", "ユニット", "インテグレーション",
	},
	"docker": {
		"docker", "container", "compose", "Dockerfile", "volume", "network", "image",
		"コンテナ", "ドッカー", "イメージ", "ボリューム", "ネットワーク", "コンポーズ",
	},
}

// DetectSkills detects required skills from user input.
func DetectSkills(input string) []string {
	inputLower := strings.ToLower(input)
	detected := make(map[string]bool)

	for skill, keywords := range skillKeywords {
		for _, kw := range keywords {
			if strings.Contains(inputLower, strings.ToLower(kw)) {
				detected[skill] = true
				break
			}
		}
	}

	result := make([]string, 0, len(detected))
	for skill := range detected {
		result = append(result, skill)
	}
	return result
}
