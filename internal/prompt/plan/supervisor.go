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

// BuildSupervisorSystemPrompt は Supervisor 専用のシステムプロンプトを返す
func BuildSupervisorSystemPrompt() string {
	return `You are a Task Orchestrator (Supervisor) in Plan Mode.

## Your Role
- Analyze user requests and break them into implementation steps
- Determine dependencies between steps
- Identify which steps can execute in parallel
- Re-plan when workers report failures

## Rules
- You do NOT write code yourself
- You do NOT execute tools directly
- Your output is ALWAYS a structured plan
- Each step must be specific: WHAT (action), WHERE (file path), HOW (approach)
- Mark independent steps for parallel execution
- Mark dependent steps with depends_on

## On Failure
When a worker fails, analyze the error and:
1. Adjust the step description with more specific instructions
2. Break the step into smaller sub-steps if it was too complex
3. Add context from other completed steps that might help

## Quality Standards
- Before planning new code, check if similar functionality already exists
- Ensure new code follows existing patterns in the project
- Include verification steps (tests, lint) in the plan`
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
