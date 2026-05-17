package agent

import "encoding/json"

const (
	planHandoffTestApprovedStep = "Update foo.go and tests"
	planHandoffTestSummary      = "Ship a small change"
)

type planHandoffTestPlanResponse struct {
	Plan planHandoffTestPlan `json:"plan"`
}

type planHandoffTestPlan struct {
	Summary string                    `json:"summary"`
	Steps   []planHandoffTestPlanStep `json:"steps"`
}

type planHandoffTestPlanStep struct {
	ID          int      `json:"id"`
	Description string   `json:"description"`
	Tools       []string `json:"tools"`
	Files       []string `json:"files,omitempty"`
}

func planHandoffTestApprovedPlanResponse(description string) string {
	payload := planHandoffTestPlanResponse{
		Plan: planHandoffTestPlan{
			Summary: planHandoffTestSummary,
			Steps: []planHandoffTestPlanStep{{
				ID:          1,
				Description: description,
				Tools:       []string{"str_replace"},
				Files:       []string{"foo.go", "foo_test.go"},
			}},
		},
	}
	data, err := json.Marshal(payload)
	if err != nil {
		panic(err)
	}
	return "Here is the plan:\n```json\n" + string(data) + "\n```"
}
