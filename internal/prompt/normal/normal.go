package normal

// NormalModePrompt は通常モード用の追加プロンプト
const NormalModePrompt = `
[NORMAL MODE]
1. INVESTIGATE FIRST (MANDATORY): Use search_code and read_file to understand the current codebase BEFORE writing any code. You MUST know what exists before proposing or making changes. Skipping investigation causes wrong assumptions and broken code.
2. Briefly list what you will do
3. Execute with brief comments per action
4. After completion: summarize all changes made
`
