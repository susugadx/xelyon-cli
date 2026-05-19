package agent

import (
	"bytes"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

func TestShowTaskSummary_NoChangesWritesNothing(t *testing.T) {
	var out bytes.Buffer
	agent := &Agent{
		Runtime: &AgentRuntime{
			UI: ui.NewRuntime(strings.NewReader(""), &out, &out),
		},
	}

	agent.showTaskSummary()
	if out.Len() != 0 {
		t.Fatalf("showTaskSummary() output = %q, want empty", out.String())
	}
}

func TestShowTaskSummary_UsesTaskOffsetAndDetails(t *testing.T) {
	var out bytes.Buffer
	agent := &Agent{
		Runtime: &AgentRuntime{
			UI: ui.NewRuntime(strings.NewReader(""), &out, &out),
		},
	}
	agent.changeStack = []tools.FileChange{
		{
			FilePath:     "ignored.go",
			Tool:         "write_file",
			LinesAdded:   1,
			LinesRemoved: 0,
		},
		{
			Tool: "str_replace",
			Details: []tools.FileChangeDetail{
				{FilePath: "internal/agent/agent.go", Action: "modified", LinesAdded: 3, LinesRemoved: 1},
				{FilePath: "internal/ui/view.go", LinesAdded: 5, LinesRemoved: 0},
			},
		},
	}
	agent.taskChangeOffset = 1
	agent.taskTestCommand = "go test ./internal/agent"
	passed := true
	agent.taskTestResult = &passed
	agent.taskPlanVerification = []string{"go test ./internal/agent", "make ci-check"}

	agent.showTaskSummary()
	got := out.String()

	for _, fragment := range []string{
		"Task Completed",
		"agent.go",
		"view.go",
		"modified",
		"go test ./internal/agent",
		"Planned verification",
		"make ci-check",
		"passed",
		"2 file(s)",
		"+8",
		"-1",
	} {
		if !strings.Contains(got, fragment) {
			t.Fatalf("showTaskSummary() output missing %q:\n%s", fragment, got)
		}
	}
	if strings.Contains(got, "ignored.go") {
		t.Fatalf("showTaskSummary() should ignore changes before taskChangeOffset:\n%s", got)
	}
}

func TestBeginTaskTracking_ClearsPlanVerification(t *testing.T) {
	agent := &Agent{}
	agent.taskPlanVerification = []string{"go test ./internal/agent"}

	agent.beginTaskTracking()

	if len(agent.taskPlanVerification) != 0 {
		t.Fatalf("taskPlanVerification = %#v, want cleared", agent.taskPlanVerification)
	}
}
