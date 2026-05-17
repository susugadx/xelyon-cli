package plan

const planJSONSchemaInstructions = `### Plan JSON Schema
Use exactly one JSON object in this shape:
` + "```json" + `
{
  "plan": {
    "summary": "User-facing implementation goal with scope",
    "steps": [
      {
        "id": 1,
        "description": "Concrete implementation action and intended outcome",
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
- plan.summary: one concise, reviewable sentence describing the goal, scope, and important constraint when known.
- steps: keep the plan short and ordered (normally 2-6 steps). Each step must be understandable without the investigation transcript.
- steps[].description: an implementation action and expected outcome, not an investigation action. Mention tests/docs/config in the step when that is the point of the work.
- steps[].tools: expected tools or verification commands for the step. Include focused test commands when known.
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
