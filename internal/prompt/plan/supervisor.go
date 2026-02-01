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

IMPORTANT: Include "depends_on" for each step to specify dependencies:
- Step that modifies file A depends on any step that also modifies file A
- Step that reads file A depends on any step that writes to file A
- Steps with no dependencies can be executed IN PARALLEL

Use the create_plan tool to save the plan.

Steps with the SAME depends_on values can be executed in parallel.

If no implementation is needed, use create_plan with an empty steps array and summary explaining why.

Do NOT output JSON directly. Always use the create_plan tool.`, userRequest, investigationResults)
}
