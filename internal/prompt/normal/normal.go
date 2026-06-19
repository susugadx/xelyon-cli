package normal

// NormalModePrompt は旧 runtime/session が user message へ付けていた suffix です。
// 新規 request では使わず、履歴互換の stripping 用に残します。
const NormalModePrompt = `
[NORMAL MODE]
Investigate -> implement -> verify. Summarize changes when done.
`
