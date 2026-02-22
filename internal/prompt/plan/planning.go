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
- Each step description MUST include:
  - Target file path (e.g. internal/tools/dev/bash.go)
  - What specifically changes (e.g. function name, variable, what is added/modified/deleted)
  Bad: "Add read-only commands to defaultSafeCommands in bash.go"
  Good: "internal/tools/dev/bash.go の defaultSafeCommands に sed -n, diff, file, du, stat, md5sum, sha256sum を追加"
- Each step MUST include a files field with target file paths.

Plans are saved to .xelyon/plans/ as Markdown.`
}
