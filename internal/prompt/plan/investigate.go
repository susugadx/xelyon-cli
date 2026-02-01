package plan

import "fmt"

// BuildInvestigationPrompt generates the prompt for the investigation phase.
// userRequest is the original user request to investigate.
func BuildInvestigationPrompt(userRequest string) string {
	return fmt.Sprintf(`USER REQUEST: %s

You are in PLAN MODE (Investigation Phase).

### CRITICAL TOOLS PRIORITY (YOU MUST FOLLOW THIS ORDER)

#### ⚠️ MANDATORY: RepoMap FIRST (NEVER SKIP)
RepoMap is at the END of system prompt. It contains ALL files and functions.

**ALWAYS check RepoMap BEFORE any search.**
**ALWAYS check RepoMap BEFORE any search.**
**ALWAYS check RepoMap BEFORE any search.**

Example:
- Query: "Find ExecuteStrReplace function"
- WRONG: search_code "ExecuteStrReplace" (wasteful)
- RIGHT: Check RepoMap → "str_replace.go:22" → read_file str_replace.go:22-50

If you use search_code without checking RepoMap first, you are wasting tokens.

#### ⚠️ MANDATORY: LSP Tools SECOND (MORE ACCURATE THAN SEARCH)
LSP tools are FASTER and MORE ACCURATE than search_code.

**ALWAYS use LSP for definitions and references.**
**ALWAYS use LSP for definitions and references.**
**ALWAYS use LSP for definitions and references.**

| Query | WRONG | RIGHT |
|-------|-------|-------|
| "Where is X defined?" | search_code "X" | lsp_definition |
| "Where is X used?" | search_code "X" | lsp_references |
| "What type is X?" | read_file | lsp_hover |
| "Any errors?" | run_test | lsp_diagnostics |

- lsp_definition: Jump to definition (1 call, exact location)
- lsp_references: Find all usages (accurate, ignores comments)
- lsp_hover: Get type info (no file reading needed)
- lsp_diagnostics: Get errors/warnings (faster than run_test)

#### Search & File Tools (USE ONLY WHEN ABOVE FAILS)
- search_code: Keyword search (ONLY for patterns not in RepoMap/LSP)
- search_file: Find files by name
- list_dir: List directory contents
- read_file: Read file contents (ALWAYS use line range to save tokens)

#### Shell Tools (READ-ONLY ONLY)
- bash: Run read-only shell commands
  - ✅ OK: ls, cat, find, grep, head, tail, wc, tree, du
  - ✅ Git (read): git status, git diff, git log, git show, git branch
  - ❌ NG: rm, mv, cp, mkdir, touch, write operations

#### Web Tools
- web_search: Search the web for API docs, error solutions, best practices

### CRITICAL REMINDER (READ 5 TIMES)
1. RepoMap FIRST → LSP SECOND → Search LAST
2. RepoMap FIRST → LSP SECOND → Search LAST
3. RepoMap FIRST → LSP SECOND → Search LAST
4. RepoMap FIRST → LSP SECOND → Search LAST
5. RepoMap FIRST → LSP SECOND → Search LAST

NEVER use search_code for function definitions. Use lsp_definition.
NEVER use search_code for finding usages. Use lsp_references.
NEVER skip RepoMap check.

### FORBIDDEN TOOLS (DO NOT USE)
- write_file, str_replace, delete_file, git_commit, git_checkout
- Any tool that modifies the codebase

### INVESTIGATION CHECKLIST
- [ ] Understand the current implementation (read relevant files)
- [ ] Find related code (search for usages, dependencies)
- [ ] Check for existing patterns to follow
- [ ] Identify potential impacts of changes

### AFTER INVESTIGATION
When you have gathered enough information:
1. If you need clarification from the user, use the ask_user_question tool
2. When ready, use the create_plan tool to create your implementation plan

IMPORTANT:
- The plan should contain IMPLEMENTATION steps, not investigation steps
- Do NOT create a plan that says "investigate X" or "read file Y"
- Each step should be an ACTION that modifies the codebase
- If you still need to investigate, continue using read-only tools first

Do NOT output JSON directly. Always use the create_plan tool.

Start investigation now.`, userRequest)
}

// BuildPlanRequestMessage generates the system message asking for a plan
// when a modification tool is detected during investigation.
// toolName is the name of the detected modification tool.
func BuildPlanRequestMessage(toolName string) string {
	return fmt.Sprintf(`[SYSTEM] You tried to use a modification tool (%s) during the investigation phase.

Before using modification tools, you must create an implementation plan.
Use the create_plan tool now to create your plan.

Do NOT output JSON directly.`, toolName)
}