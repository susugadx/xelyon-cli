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
- ONE STEP = ONE CONCERN: each step should target a single function, feature, or fix
  Bad: "Implement index.go (Build, Save, Load, Search, Update)"
  Good: Split into separate steps for Build+Save, Load, Search, Update+Git integration
- Do NOT combine multiple features into one step. More steps = better tracking.
- Each step description MUST include:
  - Target file path (e.g. internal/tools/dev/bash.go)
  - What specifically changes (e.g. function name, variable, what is added/modified/deleted)
  Bad: "Add read-only commands to defaultSafeCommands in bash.go"
  Good: "Add sed -n, diff, file, du, stat, md5sum, sha256sum to defaultSafeCommands in internal/tools/dev/bash.go"
- Each step MUST include a files field with target file paths.

Plans are saved to .xelyon/plans/ as Markdown.`
}
