package plan

const planJSONSchemaInstructions = `### Plan JSON Schema
Use exactly one JSON object in this shape:
` + "```json" + `
{
  "plan": {
    "summary": "User-facing implementation goal with scope",
    "findings": [
      "Important investigation fact that implementation mode should know"
    ],
    "evidence": [
      "internal/agent/plan/handoff.go: ImplementationHandoff.NormalModeInput builds the implementation handoff"
    ],
    "constraints": [
      "Do not carry raw investigation history into implementation mode"
    ],
    "steps": [
      {
        "id": 1,
        "description": "Concrete implementation action and intended outcome",
        "purpose": "Why this step is needed or what risk/contract it closes",
        "tools": ["read_file", "str_replace"],
        "files": [
          "internal/agent/plan/handoff.go",
          "internal/agent/plan/handoff_test.go"
        ],
        "verification": ["go test ./internal/agent"]
      }
    ]
  }
}
` + "```" + `

Field rules:
- plan.summary: one concise, reviewable sentence describing the goal, scope, and important constraint when known.
- plan.findings: concise investigation facts that are useful during implementation. Include only stable facts discovered from the codebase; omit guesses.
- plan.evidence: files, functions, tests, commands, or concrete observations supporting the plan. Preserve file/function/test names when known.
- plan.constraints: constraints, compatibility requirements, existing design boundaries, and changes to avoid.
- steps: keep the plan short and ordered (normally 2-6 steps). Each step must be understandable without the investigation transcript.
- steps[].description: an implementation action and expected outcome, not an investigation action. Mention tests/docs/config in the step when that is the point of the work.
- steps[].purpose: a short review-facing reason for the step. Say what user-visible behavior, contract, risk, or cleanup it addresses. Omit only when description already fully explains the reason.
- steps[].tools: expected implementation or inspection tools for the step, not test commands.
- steps[].files: implementation-relevant repo-relative files to confirm first in implementation mode. Include source, tests, docs, and config files when known; omit only when no relevant file is known yet.
- steps[].verification: focused commands or checks that prove this step, or the whole plan, worked. Include package-level tests when known; use an empty array only when no concrete check is known yet.`

// PlanJSONSchemaInstructions は Plan mode が要求する Plan JSON schema 文面を返す。
func PlanJSONSchemaInstructions() string {
	return planJSONSchemaInstructions
}

// BuildPlanJSONRetryMessage は Plan JSON 修復用のユーザーメッセージを生成する。
func BuildPlanJSONRetryMessage() string {
	return "Plan JSON retry: 必ず次の schema に沿って、```json``` で囲んだ1つのJSONとして出力してください（箇条書き/番号付きリスト/文章のみは禁止）。\n\n" +
		PlanJSONSchemaInstructions()
}
