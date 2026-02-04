package plan

import "fmt"

// BuildInvestigationQueryPrompt は Supervisor が調査クエリを生成するためのプロンプト
func BuildInvestigationQueryPrompt(userRequest string, maxWorkers int) string {
	return fmt.Sprintf(`USER REQUEST: %s

You are the Supervisor in Plan Mode. Generate 1-%d investigation queries for parallel Worker execution.

Each query should:
- Focus on a SPECIFIC aspect (file structure, implementation, dependencies)
- Be independent of other queries (no dependencies)
- Be clear and specific

Return JSON:
{
  "queries": [
    "Query 1: ...",
    "Query 2: ..."
  ]
}

If trivial (greeting, simple question), return: {"queries": []}`, userRequest, maxWorkers)
}

func BuildPlanGenerationPrompt(userRequest, investigationResults string) string {
	return fmt.Sprintf(`USER REQUEST: %s

INVESTIGATION RESULTS:
%s

Based on the investigation, create an implementation plan.

### Step Rules
- Each step = ONE focused task, not multiple edits combined
- Include WHAT (action), WHERE (file path), HOW (approach) in each description
  - GOOD: "Add validateEmail function to internal/utils/validation.go using regex"
  - BAD: "Add validation"
- Use depends_on when steps modify/read the same file
- Steps with no dependencies execute IN PARALLEL

Use the create_plan tool to save the plan.
If no implementation is needed, use create_plan with empty steps and summary explaining why.
Do NOT output JSON directly.`, userRequest, investigationResults)
}
