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

Return JSON:
{
  "plan": {
    "summary": "Brief description of the plan",
    "steps": [
      {
        "id": 1,
        "description": "Step description",
        "tools": ["tool1", "tool2"],
        "depends_on": []
      },
      {
        "id": 2,
        "description": "Step description",
        "tools": ["tool1"],
        "depends_on": [1]
      },
      {
        "id": 3,
        "description": "Step description (can run parallel with step 2)",
        "tools": ["tool1"],
        "depends_on": [1]
      }
    ]
  }
}

Steps with the SAME depends_on values can be executed in parallel.

If the investigation reveals that no implementation is needed (e.g., simple question answered, no code changes required), return an empty plan:
{
  "plan": {
    "summary": "Investigation complete - no implementation needed",
    "steps": []
  }
}`, userRequest, investigationResults)
}
