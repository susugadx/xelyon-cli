package plan

const planJSONSchemaInstructions = `### Plan JSON Schema
Use exactly one JSON object in this shape:
` + "```json" + `
{
  "plan": {
    "summary": "Short implementation goal",
    "steps": [
      {
        "id": 1,
        "description": "Concrete implementation action",
        "tools": ["str_replace", "go test ./internal/agent"],
        "files": [
          "internal/agent/plan_handoff.go",
          "internal/agent/plan_handoff_test.go"
        ]
      }
    ]
  }
}
` + "```" + `

Field rules:
- plan.summary: one concise sentence describing the implementation goal.
- steps[].description: an implementation action, not an investigation action.
- steps[].tools: expected tools or verification commands for the step.
- steps[].files: implementation-relevant repo-relative files to confirm first in implementation mode. Include source, tests, docs, and config files when known; omit only when no relevant file is known yet.`

// PlanJSONSchemaInstructions は Plan mode が要求する Plan JSON schema 文面を返す。
func PlanJSONSchemaInstructions() string {
	return planJSONSchemaInstructions
}

// BuildPlanJSONRetryMessage は Plan JSON 修復用のユーザーメッセージを生成する。
func BuildPlanJSONRetryMessage() string {
	return "[SYSTEM] Plan JSON を**必ず**次の schema に沿って、```json``` で囲んだ1つのJSONとして出力してください（箇条書き/番号付きリスト/文章のみは禁止）。\n\n" +
		PlanJSONSchemaInstructions()
}
