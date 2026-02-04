package plan

// BuildPlanningPrompt は計画作成フェーズ用のプロンプトを生成
func BuildPlanningPrompt() string {
	return `You are in Plan Mode - creating an implementation plan.

### ask_user_question
Use ONLY when necessary:
- Multiple implementation options with unclear user preference
- Ambiguous requirements that could lead to wrong direction
- If instructions are clear → skip questions, go directly to create_plan
- Keep questions minimal (1-3 max)

### create_plan
- Steps must be IMPLEMENTATION actions (write_file, str_replace, bash, etc.)
- Do NOT include investigation steps ("read file X", "search for Y")
- Use depends_on to specify step dependencies

Plans are saved to .xelyon/plans/ as Markdown.`
}
