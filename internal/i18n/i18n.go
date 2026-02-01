// Package i18n は多言語対応を提供する
package i18n

import (
	"fmt"
	"sync"
)

var (
	currentLang = "ja"
	mu          sync.RWMutex
)

// SetLang は言語を設定
func SetLang(lang string) {
	mu.Lock()
	defer mu.Unlock()
	if lang == "en" || lang == "ja" {
		currentLang = lang
	}
}

// GetLang は現在の言語を取得
func GetLang() string {
	mu.RLock()
	defer mu.RUnlock()
	return currentLang
}

// T はキーに対応するメッセージを返す
func T(key string, args ...interface{}) string {
	mu.RLock()
	lang := currentLang
	mu.RUnlock()

	msgs, ok := messages[lang]
	if !ok {
		msgs = messages["en"]
	}

	msg, ok := msgs[key]
	if !ok {
		return key // フォールバック: キーをそのまま返す
	}

	if len(args) > 0 {
		return fmt.Sprintf(msg, args...)
	}
	return msg
}

var messages = map[string]map[string]string{
	"ja": {
		// Plan
		"plan.created":    "計画を作成しました: %s",
		"plan.saved":      "計画を保存しました: %s",
		"plan.load_error": "計画の読み込みに失敗しました: %v",
		"plan.not_found":  "計画が見つかりません: %s",
		"plan.deleted":    "計画を削除しました: %s",
		"plan.updated":    "計画を更新しました: %s",
		"plan.list_empty": "保存された計画はありません",

		// Questionnaire
		"q.header":        "いくつか確認させてください：",
		"q.choice_prompt": "選択してください",
		"q.multi_prompt":  "複数選択（カンマ区切り）",
		"q.text_prompt":   "入力してください",
		"q.default_hint":  "(Enter でデフォルト: %s)",
		"q.selected":      "選択: %s",

		// Errors
		"error.invalid_option": "無効な選択肢です",
		"error.required":       "入力は必須です",

		// Supervisor / Worker
		"supervisor.investigation_start":    "🔍 調査フェーズを開始（並列実行）",
		"supervisor.investigation_complete": "📊 調査完了（%s、並列実行）",
		"supervisor.execution_start":        "🚀 実装フェーズを開始（並列実行）",
		"supervisor.step_retry":             "🔄 Step %d 失敗、リトライ中... (%d/%d)",
		"supervisor.step_escalate":          "⬆️ Step %d を高精度モデルにエスカレーション",
		"supervisor.step_completed_heavy":   "✅ Step %d 完了（高精度モデル）",
		"supervisor.step_skipped":           "⏭️ Step %d をスキップしました",
		"supervisor.step_aborted":           "❌ Step %d がユーザーにより中止されました",
		"supervisor.all_completed":          "✅ 全ステップ完了",
		"worker.step_timeout":               "⏱️ Step %d がタイムアウト（%d秒）",
		"worker.step_failed":                "❌ Step %d 失敗: %s",
		"worker.step_completed":             "✅ Step %d 完了（%s）",
		"worker.investigation_completed":    "✅ 調査完了: %s",
	},
	"en": {
		// Plan
		"plan.created":    "Plan created: %s",
		"plan.saved":      "Plan saved: %s",
		"plan.load_error": "Failed to load plan: %v",
		"plan.not_found":  "Plan not found: %s",
		"plan.deleted":    "Plan deleted: %s",
		"plan.updated":    "Plan updated: %s",
		"plan.list_empty": "No saved plans",

		// Questionnaire
		"q.header":        "A few questions:",
		"q.choice_prompt": "Choose one",
		"q.multi_prompt":  "Select multiple (comma-separated)",
		"q.text_prompt":   "Enter your answer",
		"q.default_hint":  "(Enter for default: %s)",
		"q.selected":      "Selected: %s",

		// Errors
		"error.invalid_option": "Invalid option",
		"error.required":       "Input is required",

		// Supervisor / Worker
		"supervisor.investigation_start":    "🔍 Starting investigation phase (parallel)",
		"supervisor.investigation_complete": "📊 Investigation complete (%s, parallel)",
		"supervisor.execution_start":        "🚀 Starting execution phase (parallel)",
		"supervisor.step_retry":             "🔄 Step %d failed, retrying... (%d/%d)",
		"supervisor.step_escalate":          "⬆️ Escalating step %d to heavy model",
		"supervisor.step_completed_heavy":   "✅ Step %d completed (heavy model)",
		"supervisor.step_skipped":           "⏭️ Step %d skipped",
		"supervisor.step_aborted":           "❌ Step %d aborted by user",
		"supervisor.all_completed":          "✅ All steps completed",
		"worker.step_timeout":               "⏱️ Step %d timed out (%d seconds)",
		"worker.step_failed":                "❌ Step %d failed: %s",
		"worker.step_completed":             "✅ Step %d completed (%s)",
		"worker.investigation_completed":    "✅ Investigation complete: %s",
	},
}
