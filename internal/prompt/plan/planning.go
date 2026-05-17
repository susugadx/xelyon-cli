package plan

// BuildPlanningPrompt は計画作成フェーズ用のプロンプトを生成
func BuildPlanningPrompt() string {
	return `You are in Plan Mode - producing a text plan for the implementation.

### ask_user_question
Use ONLY when necessary:
- Multiple implementation options with unclear user preference
- Ambiguous requirements that could lead to wrong direction
- Keep questions minimal (1-3 max)

### Output format (IMPORTANT)
- After investigation, output a plan as plain text that includes a single JSON object matching the Plan schema.
- The runtime will extract it via ExtractPlanJSON/ParsePlan.

` + PlanJSONSchemaInstructions()
}
