package plan

import (
	"fmt"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/agent/plan"
)

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

// BuildStepExecutionPrompt は Worker がステップを実行するためのプロンプト
func BuildStepExecutionPrompt(step *plan.PlanStep, context string) string {
	toolsList := "any appropriate tools"
	if len(step.Tools) > 0 {
		toolsList = strings.Join(step.Tools, ", ")
	}

	return fmt.Sprintf(`Execute step %d:
%s

Previous context from completed steps:
%s

Suggested tools: %s

Execute this step autonomously without asking questions.
Focus on completing this specific step only.
Report success or failure clearly.`, step.ID, step.Description, context, toolsList)
}

// BuildInvestigationExecutionPrompt は Worker が調査を実行するためのプロンプト
func BuildInvestigationExecutionPrompt(query string) string {
	return fmt.Sprintf(`INVESTIGATION QUERY:
%s

Execute this investigation using read-only tools:
- read_file: Read file contents
- search_code: Search for code patterns
- search_file: Find files by name
- list_dir: List directory contents
- git_status: Check git status
- git_diff: View git diff

Gather relevant information and report your findings clearly.
Do NOT make any modifications to the codebase.
Focus on answering the query comprehensively.`, query)
}
