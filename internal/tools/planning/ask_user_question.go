// Package planning は Plan Mode 用のツールを提供する
package planning

import (
	"encoding/json"
	"fmt"

	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

// AskUserQuestionTool はLLMがユーザーに質問するためのツール
type AskUserQuestionTool struct{}

// Name はツール名を返す
func (t *AskUserQuestionTool) Name() string {
	return "ask_user_question"
}

// Description はツールの説明を返す
func (t *AskUserQuestionTool) Description() string {
	return `計画を立てる前にユーザーに確認事項を質問します。

【重要】必要な場合のみ使用してください。

✓ 質問すべき場合：
- 実装方法に複数の選択肢があり、ユーザーの好みが不明
- 要件が曖昧で、間違った方向に進むリスクがある
- スコープが広すぎて確認が必要

✗ 質問すべきでない場合：
- ユーザーが具体的に指示している（「JWT使って」「既存DBに」など）
- 一般的なベストプラクティスで判断できる
- 小さなタスクで、間違っても修正が容易

原則：
- 明確な指示なら直接 create_plan へ進む
- 質問は最小限に（1-3問程度）
- 「念のため確認」は避ける`
}

// Parameters はツールのパラメータ定義を返す
func (t *AskUserQuestionTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"question": map[string]interface{}{
				"type":        "string",
				"description": "ユーザーに聞く質問",
			},
			"question_type": map[string]interface{}{
				"type":        "string",
				"enum":        []string{"single_choice", "multi_choice", "free_text"},
				"description": "質問タイプ: single_choice, multi_choice, free_text",
			},
			"options": map[string]interface{}{
				"type":        "array",
				"items":       map[string]interface{}{"type": "string"},
				"description": "選択肢（single_choice/multi_choice の場合必須）",
			},
			"default_option": map[string]interface{}{
				"type":        "string",
				"description": "デフォルトの選択肢（オプション）",
			},
		},
		"required":             []string{"question", "question_type"},
		"additionalProperties": false,
	}
}

// Run はツールを実行
func (t *AskUserQuestionTool) Run(args map[string]string) (string, *tools.FileChange, error) {
	question := args["question"]
	questionType := args["question_type"]
	optionsJSON := args["options"]
	defaultOption := args["default_option"]

	// options のパース
	var options []string
	if optionsJSON != "" {
		if err := json.Unmarshal([]byte(optionsJSON), &options); err != nil {
			// カンマ区切りとして試す
			options = []string{optionsJSON}
		}
	}

	// バリデーション
	if questionType == "single_choice" || questionType == "multi_choice" {
		if len(options) == 0 {
			return "", nil, fmt.Errorf("options required for %s question", questionType)
		}
	}

	// Questionnaire で質問を表示
	q := &ui.Questionnaire{
		Question:     question,
		QuestionType: questionType,
		Options:      options,
		Default:      defaultOption,
	}

	answer, err := q.Ask()
	if err != nil {
		return "", nil, fmt.Errorf("failed to get user answer: %w", err)
	}

	// 回答を JSON で返す
	response := map[string]interface{}{
		"question":      question,
		"question_type": questionType,
		"answer":        answer.Value,
	}
	if answer.IsMultiple {
		response["answers"] = answer.Values
	}

	responseJSON, _ := json.Marshal(response)
	return string(responseJSON), nil, nil
}
