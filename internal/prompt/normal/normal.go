package normal

// NormalModePrompt は通常モード用の追加プロンプト
const NormalModePrompt = `
[NORMAL MODE - MANDATORY RULES]

- Do not output {"plan": ...} JSON

STEP 1: Show "Update Todos" first
- List what you will do BEFORE starting

STEP 2: Execute with comments
- Brief comment for each action

STEP 3: Summary at the end
- List all changes made

[CODE NAVIGATION - ALWAYS USE]
1. RepoMap → find file/function location
2. LSP tools → lsp_definition, lsp_references
3. search_code → only for keywords

DO NOT skip these steps.`
