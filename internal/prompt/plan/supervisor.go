package plan

import "fmt"

// BuildInvestigationQueryPrompt は Supervisor が調査クエリを生成するためのプロンプト
func BuildInvestigationQueryPrompt(userRequest string, maxWorkers int) string {
	return fmt.Sprintf(`USER REQUEST: %s

You are the Supervisor in Plan Mode. Analyze this request and generate 1-%d investigation queries that can be executed IN PARALLEL by Workers.

Each query should:
1. Focus on a SPECIFIC aspect (file structure, existing implementation, dependencies, etc.)
2. Be independent of other queries (no dependencies between queries)
3. Be clear and specific so Workers can investigate effectively

### IMPORTANT: Workers have these tools (in priority order)
1. RepoMap - Already contains all file paths and function names
2. LSP tools - lsp_definition, lsp_references, lsp_hover
3. Search tools - search_code, search_file (use only when above fails)
4. bash - read-only commands (git status, git diff, etc.)
5. web_search - for external documentation

When writing queries, consider what tools Workers will use.
- "Find function X" → Worker uses RepoMap + lsp_definition
- "Find all usages of X" → Worker uses lsp_references
- "Check project structure" → Worker uses RepoMap + list_dir

Return JSON:
{
  "queries": [
    "Query 1: ...",
    "Query 2: ..."
  ]
}

If only one query is needed, return a single query.
If the request is trivial (greeting, simple question), return empty queries: {"queries": []}`, userRequest, maxWorkers)
}

// BuildPlanGenerationPrompt は depends_on 付きの計画を生成するためのプロンプト
func BuildPlanGenerationPrompt(userRequest, investigationResults string) string {
	return fmt.Sprintf(`USER REQUEST: %s

INVESTIGATION RESULTS:
%s

Based on the investigation, create an implementation plan.

### CRITICAL: Plan Quality Rules

#### 1. Step Granularity (IMPORTANT)
- Each step should be ONE focused task
- NEVER combine multiple file edits in one step
- NEVER make steps too vague like "implement the feature"
- GOOD: "Add User struct to models/user.go"
- BAD: "Implement user management system"

#### 2. Dependencies (depends_on)
- Step that modifies file A depends on any step that also modifies file A
- Step that reads file A depends on any step that writes to file A
- Steps with NO dependencies can be executed IN PARALLEL
- Steps with the SAME depends_on values can be executed in parallel

#### 3. Step Description (CRITICAL)
Each step description should include:
- WHAT to do (specific action)
- WHERE to do it (file path)
- HOW to do it (brief approach)

Example:
- GOOD: "Add validateEmail function to internal/utils/validation.go that checks email format using regex"
- BAD: "Add validation"

#### 4. Worker Context
Workers will execute steps autonomously. They have:
- RepoMap (all files and functions)
- LSP tools (definitions, references)
- Full file access

Write step descriptions clearly so Workers can execute without asking questions.

### Instructions
Use the create_plan tool to save the plan.

If no implementation is needed, use create_plan with an empty steps array and summary explaining why.

Do NOT output JSON directly. Always use the create_plan tool.`, userRequest, investigationResults)
}
