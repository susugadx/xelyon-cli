package plan

import (
	"fmt"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/agent/plan"
)

// BuildStepExecutionPrompt は Worker がステップを実行するためのプロンプト
func BuildStepExecutionPrompt(step *plan.PlanStep, context string) string {
	toolsList := "any appropriate tools"
	if len(step.Tools) > 0 {
		toolsList = strings.Join(step.Tools, ", ")
	}

	return fmt.Sprintf(`## Execute Step %d

### Task
%s

### CRITICAL RULES (ALWAYS FOLLOW)

#### 1. Report Format (MANDATORY)
You MUST report your results in this exact format:

WHAT: [Detailed description of what you did]
FILES: [List of files changed with line numbers]
RESULT: [Success or Failure with specific reason]
NEXT: [Information that other workers need to know]

#### 2. Code Quality (ALWAYS)
- ALWAYS follow Project Context (XELYON.md) rules if present
- ALWAYS run the appropriate formatter for the language
- ALWAYS follow the project's existing code style
- ALWAYS write comments for new functions
- ALWAYS verify changes work without errors

#### 3. Reporting Rules (CRITICAL)
- NEVER report just "done" or "completed"
- NEVER give vague responses
- ALWAYS be specific about what changed
- ALWAYS include file paths and line numbers

### REMEMBER (IMPORTANT - Read 3 times)
1. Report in detail, not just "done"
2. Report in detail, not just "done"
3. Report in detail, not just "done"

### Previous Context
%s

### Suggested Tools
%s

### Instructions
Execute this step autonomously without asking questions.
Focus on completing this specific step only.
When done, report using the MANDATORY format above.`, step.ID, step.Description, context, toolsList)
}

// BuildInvestigationExecutionPrompt は Worker が調査を実行するためのプロンプト
func BuildInvestigationExecutionPrompt(query string) string {
	return fmt.Sprintf(`## INVESTIGATION QUERY
%s

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
- ❌ Git (write): git add, git commit, git push

#### Web Tools
- web_search: Search the web for:
- API documentation
- Error message solutions
- Best practices
- Library usage examples

### CRITICAL REMINDER (READ 5 TIMES)
1. RepoMap FIRST → LSP SECOND → Search LAST
2. RepoMap FIRST → LSP SECOND → Search LAST
3. RepoMap FIRST → LSP SECOND → Search LAST
4. RepoMap FIRST → LSP SECOND → Search LAST
5. RepoMap FIRST → LSP SECOND → Search LAST

NEVER use search_code for function definitions. Use lsp_definition.
NEVER use search_code for finding usages. Use lsp_references.
NEVER skip RepoMap check.

### CRITICAL RULES (ALWAYS FOLLOW)

#### Report Format (MANDATORY)
You MUST report your findings in this exact format:

QUERY: [The original query]
FINDINGS: [Detailed findings with file paths and line numbers]
FILES_EXAMINED: [List of files you read]
CONCLUSION: [Summary and recommendations]

#### Reporting Rules
- NEVER report just "found it" or "done"
- ALWAYS include specific file paths and line numbers
- ALWAYS provide comprehensive findings
- ALWAYS summarize what you discovered

### REMEMBER (IMPORTANT - Read 3 times)
1. Be specific with file paths and line numbers
2. Be specific with file paths and line numbers
3. Be specific with file paths and line numbers

### Instructions
Do NOT make any modifications to the codebase.
Focus on answering the query comprehensively.
Report using the MANDATORY format above.`, query)
}
