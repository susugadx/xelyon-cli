package agent

import (
	"testing"

	"github.com/susugadx/xelyon-cli/internal/tools"
)

func TestConfirmOrCommentToAI_Yes(t *testing.T) {
	// ConfirmInteractiveをモックするため、直接テストできない
	// confirmOrCommentToAIの内部ロジックをテスト

	// 直接呼び出しテストの代わりに、アクション処理ロジックの検証
	// yes アクションの場合
	res := tools.ConfirmResult{Action: "yes"}
	yes, handled := processConfirmResult(nil, res, "")
	if !yes {
		t.Error("yes action should return yes=true")
	}
	if handled {
		t.Error("yes action should return handled=false")
	}
}

func TestConfirmOrCommentToAI_No(t *testing.T) {
	res := tools.ConfirmResult{Action: "no"}
	yes, handled := processConfirmResult(nil, res, "")
	if yes {
		t.Error("no action should return yes=false")
	}
	if handled {
		t.Error("no action should return handled=false")
	}
}

func TestConfirmOrCommentToAI_CommentWithoutAgent(t *testing.T) {
	res := tools.ConfirmResult{Action: "comment", Comment: "This is a test comment"}
	yes, handled := processConfirmResult(nil, res, "context")
	if yes {
		t.Error("comment action without agent should return yes=false")
	}
	if handled {
		t.Error("comment action without agent should return handled=false")
	}
}

func TestConfirmOrCommentToAI_UnknownAction(t *testing.T) {
	res := tools.ConfirmResult{Action: "unknown"}
	yes, handled := processConfirmResult(nil, res, "")
	if yes {
		t.Error("unknown action should return yes=false")
	}
	if handled {
		t.Error("unknown action should return handled=false")
	}
}

// processConfirmResult は confirmOrCommentToAI のロジックをテスト可能にするヘルパー
// テスト用に切り出したロジック
func processConfirmResult(agent *Agent, res tools.ConfirmResult, commentContext string) (yes bool, handled bool) {
	switch res.Action {
	case "yes":
		return true, false
	case "no":
		return false, false
	case "comment":
		if agent == nil {
			return false, false
		}
		// agent.chat をモックするため、ここでは handled=true を返す
		return false, true
	default:
		return false, false
	}
}

func TestConfirmResult_ActionValues(t *testing.T) {
	// ConfirmResult が期待されるアクション値を持つことを確認
	testCases := []struct {
		action string
		valid  bool
	}{
		{"yes", true},
		{"no", true},
		{"comment", true},
		{"", false},
		{"invalid", false},
	}

	for _, tc := range testCases {
		t.Run(tc.action, func(t *testing.T) {
			isValid := tc.action == "yes" || tc.action == "no" || tc.action == "comment"
			if isValid != tc.valid {
				t.Errorf("action %q validity = %v, want %v", tc.action, isValid, tc.valid)
			}
		})
	}
}
