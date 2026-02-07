package agent

import (
	"encoding/json"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/tools"
)

// isSameToolCall は2つのToolCallが同じかを判定
func isSameToolCall(tc1, tc2 *tools.ToolCall) bool {
	if tc1 == nil || tc2 == nil {
		return false
	}
	if tc1.Tool != tc2.Tool {
		return false
	}

	// args を JSON 化して比較
	args1, _ := json.Marshal(tc1.Args)
	args2, _ := json.Marshal(tc2.Args)
	return string(args1) == string(args2)
}

// extractExplanationAndTool はレスポンスから説明部分とツールJSONを分離
// NOTE: agent_plan.goでも使用されるため、パッケージレベルの関数として定義
func extractExplanationAndTool(response string) (explanation, toolJSON string) {
	// ツール呼び出しのJSON部分を探す
	toolStartIdx := -1
	patterns := []string{"{\"tool\"", "{ \"tool\"", "{\"tool\":", "{ \"tool\":", "{\"id\"", "{ \"id\""}
	for _, pattern := range patterns {
		idx := strings.Index(response, pattern)
		if idx != -1 && (toolStartIdx == -1 || idx < toolStartIdx) {
			toolStartIdx = idx
		}
	}

	if toolStartIdx == -1 {
		// ツール呼び出しなし
		return response, ""
	}

	// ツール呼び出しより前の部分が説明
	explanation = strings.TrimSpace(response[:toolStartIdx])

	// ツール呼び出しのJSON部分を抽出
	depth := 0
	inString := false
	escaped := false
	for i := toolStartIdx; i < len(response); i++ {
		ch := response[i]

		if escaped {
			escaped = false
			continue
		}
		if ch == '\\' {
			escaped = true
			continue
		}
		if ch == '"' {
			inString = !inString
			continue
		}
		if inString {
			continue
		}

		switch ch {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				toolJSON = response[toolStartIdx : i+1]
				return explanation, toolJSON
			}
		}
	}

	// 閉じ括弧が見つからない場合は全体をツールJSONとみなす
	return explanation, response[toolStartIdx:]
}

// getLastReasoningContent はプロバイダーから最後の reasoning_content を取得する
// DeepSeek Reasoner などの思考モデルで使用
func (a *Agent) getLastReasoningContent() string {
	return api.GetReasoningContent(a.CurrentProvider)
}

// handleNormalResponse は通常の回答（ツール呼び出しなし）を処理
func (a *Agent) handleNormalResponse(response string) {
	a.History = append(a.History, api.Message{
		Role:             "assistant",
		Content:          response,
		ReasoningContent: a.getLastReasoningContent(),
	})

	// 統計情報更新: Assistantメッセージ数をカウント
	if a.Stats != nil {
		a.Stats.AssistantMessages++
	}

	// 最後の出力を記録（最大保存数: config.MaxLastOutputs）
	a.lastOutputs = append(a.lastOutputs, response)
	if len(a.lastOutputs) > config.MaxLastOutputs {
		a.lastOutputs = a.lastOutputs[1:]
	}

	// セッションに保存
	if a.session != nil {
		a.session.AddMessage("assistant", response, a.CurrentModel)
		if a.storage != nil {
			if err := a.storage.Save(a.session); err != nil {
				// セッション保存失敗を警告（データ損失の可能性を通知）
				yellow.Printf("⚠️  Warning: Failed to save session: %v\n", err)
			}
		}
	}
}
