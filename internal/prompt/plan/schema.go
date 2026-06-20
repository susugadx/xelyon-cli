package plan

import "github.com/susugadx/xelyon-cli/internal/plancontract"

// PlanJSONSchemaInstructions は Plan mode が要求する Plan JSON schema 文面を返す。
func PlanJSONSchemaInstructions() string {
	return plancontract.SchemaInstructions()
}

// BuildPlanJSONRetryMessage は Plan JSON 修復用のユーザーメッセージを生成する。
func BuildPlanJSONRetryMessage() string {
	return "Plan JSON retry: 必ず次の schema に沿って、```json``` で囲んだ1つのJSONとして出力してください（箇条書き/番号付きリスト/文章のみは禁止）。\n\n" +
		PlanJSONSchemaInstructions()
}
