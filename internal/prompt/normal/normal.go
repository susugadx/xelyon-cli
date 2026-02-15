package normal

// NormalModePrompt は通常モード用の追加プロンプト
const NormalModePrompt = `
[NORMAL MODE]
1. Before changes: briefly list what you will do
2. For complex tasks with 3+ steps: use create_plan tool to register your plan, then STOP and wait for step-by-step instructions
3. Execute with brief comments per action
4. After completion: summarize all changes made
`
