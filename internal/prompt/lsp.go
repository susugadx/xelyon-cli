package prompt

// BuildLSPToolsPrompt generates the LSP tools description for the system prompt.
// This is added AFTER the main SystemPrompt, so it won't be removed by removeToolsSection.
func BuildLSPToolsPrompt() string {
	return `

### Code Navigation (LSP - Use First)

**IMPORTANT**: LSP tools are faster and more accurate than read_file/search_code.
Always start code investigation with LSP. Use other tools only when LSP can't solve it.

#### Tool Details

**lsp_definition** - Jump to definition
- Use: Find where a function/variable/type is defined
- Args: {"path": "file.go", "line": 10, "character": 5}
- Returns: File path and line number of definition
- Example: Want to see getUserById() implementation → lsp_definition finds it instantly
- Faster than reading 200 lines with read_file

**lsp_references** - Find all references (Most Important)
- Use: Find everywhere a function/variable is used
- Args: {"path": "file.go", "line": 10, "character": 5}
- Returns: List of all files and line numbers that reference it
- Example: Can I delete this function? → lsp_references shows all callers
- More accurate than search_code (ignores same-name strings in comments)

**lsp_hover** - Get type info and documentation
- Use: Check variable type or function signature
- Args: {"path": "file.go", "line": 10, "character": 5}
- Returns: Type information, documentation comments
- Example: What type is this variable? → lsp_hover tells you instantly
- No need to read the entire file

**lsp_diagnostics** - Get errors and warnings
- Use: Check for compile errors and warnings in a file
- Args: {"path": "file.go"}
- Returns: List of errors/warnings with line numbers and messages
- Example: Any errors after my fix? → lsp_diagnostics
- Faster than run_test (catches syntax errors without execution)

**lsp_rename** - Preview rename changes
- Use: See what would change if you rename something
- Args: {"path": "file.go", "line": 10, "character": 5, "new_name": "newName"}
- Returns: Preview of all locations that would change
- Actual changes are made with str_replace

#### Quick Reference

| Goal | Use First | Fallback (no LSP) |
|------|-----------|-------------------|
| See function implementation | lsp_definition | Repo Map → read_file |
| Where is this used? | lsp_references | search_code (less accurate) |
| What's the type? | lsp_hover | read_file (heavy) |
| Check for errors | lsp_diagnostics | run_test (slow) |
| Rename impact | lsp_rename or lsp_references | search_code |

#### Investigation Flow Examples

**GOOD (LSP first)**:
1. Check Repo Map for file structure
2. lsp_definition to find target function (1 call)
3. lsp_references to check all callers (1 call)
4. read_file only for sections needing detail
5. str_replace to make changes
6. lsp_diagnostics to verify no errors
→ Few calls, accurate, token-efficient

**BAD (read_file loop)**:
1. read_file: file1.go (200 lines) → not here
2. read_file: file2.go (200 lines) → not here
3. read_file: file3.go (200 lines) → finally found
4. search_code for callers → too many results
5. read_file: file4.go...
→ 10+ calls, massive token usage, risk of missing references

#### CRITICAL: Always use lsp_references before destructive changes

**MUST** check impact with lsp_references before:
- Renaming functions/variables
- Deleting files or functions
- Changing function signatures
- Refactoring

Skipping this breaks dependent code.
`
}
