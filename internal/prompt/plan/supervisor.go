package plan

import "fmt"

// BuildInvestigationQueryPrompt は Supervisor が調査クエリを生成するためのプロンプト
func BuildInvestigationQueryPrompt(userRequest string) string {
	return fmt.Sprintf(`USER REQUEST: %s

You are the Supervisor in Plan Mode. Analyze this request and generate 1-3 investigation queries that can be executed IN PARALLEL by Workers.

Each query should:
1. Focus on a SPECIFIC aspect (file structure, existing implementation, dependencies, etc.)
2. Be independent of other queries (no dependencies between queries)
3. Use only read-only tools: read_file, search_code, search_file, list_dir, git_status, git_diff

Return JSON:
{
  "queries": [
    "Query 1: ...",
    "Query 2: ..."
  ]
}

If only one query is needed, return a single query.
If the request is trivial (greeting, simple question), return empty queries: {"queries": []}`, userRequest)
}

// BuildPlanGenerationPrompt は depends_on 付きの計画を生成するためのプロンプト
func BuildPlanGenerationPrompt(userRequest, investigationResults string) string {
	return fmt.Sprintf(`USER REQUEST: %s

INVESTIGATION RESULTS:
%s

Based on the investigation, create an implementation plan.

IMPORTANT: Include "depends_on" for each step to specify dependencies:
- Step that modifies file A depends on any step that also modifies file A
- Step that reads file A depends on any step that writes to file A
- Steps with no dependencies can be executed IN PARALLEL

Use the create_plan tool with:
- title: Brief title for the plan
- summary: Brief description of what will be done
- steps: Array of step objects, each with:
  - id: Step number (1, 2, 3...)
  - description: What this step does
  - tools: Array of tool names to use
  - depends_on: Array of step IDs this step depends on

Steps with the SAME depends_on values can be executed in parallel.

If no implementation is needed, use create_plan with an empty steps array and summary explaining why.

Do NOT output JSON directly. Always use the create_plan tool.`, userRequest, investigationResults)
}
