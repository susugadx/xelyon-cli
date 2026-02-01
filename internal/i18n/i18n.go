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
	},
	"en": {
		// Plan
		"plan.created":    "Plan created: %s",
		"plan.saved":      "Plan saved: %s",
		"plan.load_error": "Failed to load plan: %v",

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
	},
}
