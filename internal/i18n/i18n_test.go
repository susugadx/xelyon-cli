package i18n

import (
	"testing"
)

func TestSetLang(t *testing.T) {
	// デフォルトは日本語
	if got := GetLang(); got != "ja" {
		t.Errorf("GetLang() = %v, want ja", got)
	}

	// 英語に変更
	SetLang("en")
	if got := GetLang(); got != "en" {
		t.Errorf("GetLang() = %v, want en", got)
	}

	// 日本語に戻す
	SetLang("ja")
	if got := GetLang(); got != "ja" {
		t.Errorf("GetLang() = %v, want ja", got)
	}

	// 無効な言語は無視
	SetLang("fr")
	if got := GetLang(); got != "ja" {
		t.Errorf("GetLang() = %v, want ja (invalid lang should be ignored)", got)
	}
}

func TestT(t *testing.T) {
	// 日本語
	SetLang("ja")
	if got := T("plan.created", "テスト計画"); got != "計画を作成しました: テスト計画" {
		t.Errorf("T() = %v, want 計画を作成しました: テスト計画", got)
	}

	// 英語
	SetLang("en")
	if got := T("plan.created", "Test Plan"); got != "Plan created: Test Plan" {
		t.Errorf("T() = %v, want Plan created: Test Plan", got)
	}

	// 引数なし
	if got := T("q.header"); got != "A few questions:" {
		t.Errorf("T() = %v, want A few questions:", got)
	}

	// 存在しないキーはそのまま返す
	if got := T("nonexistent.key"); got != "nonexistent.key" {
		t.Errorf("T() = %v, want nonexistent.key", got)
	}

	// 日本語に戻す
	SetLang("ja")
}

func TestTConcurrent(t *testing.T) {
	// 並行アクセスのテスト
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			SetLang("en")
			T("plan.created", "test")
			SetLang("ja")
			T("q.header")
			done <- true
		}()
	}
	for i := 0; i < 10; i++ {
		<-done
	}
}
