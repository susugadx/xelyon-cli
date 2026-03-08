package planning

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

type askUserQuestionResponse struct {
	Question     string   `json:"question"`
	QuestionType string   `json:"question_type"`
	Answer       string   `json:"answer"`
	Answers      []string `json:"answers"`
}

func setTestReader(t *testing.T, input string) {
	t.Helper()
	reader := ui.NewMultilineReader(strings.NewReader(input))
	ui.SetGlobalReader(reader)
	t.Cleanup(func() {
		ui.SetGlobalReader(nil)
	})
}

func TestAskUserQuestionTool_Run_SingleChoice(t *testing.T) {
	setTestReader(t, "2\n")

	tool := &AskUserQuestionTool{}
	result, _, err := tool.Run(tools.DefaultExecutionContext(), map[string]string{
		"question":      "Choose",
		"question_type": "single_choice",
		"options":       `["Yes","No"]`,
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	var response askUserQuestionResponse
	if err := json.Unmarshal([]byte(result), &response); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if response.QuestionType != "single_choice" {
		t.Errorf("QuestionType = %s, want %s", response.QuestionType, "single_choice")
	}
	if response.Answer != "No" {
		t.Errorf("Answer = %s, want %s", response.Answer, "No")
	}
}

func TestAskUserQuestionTool_Run_MultiChoice(t *testing.T) {
	setTestReader(t, "1,3\n")

	tool := &AskUserQuestionTool{}
	result, _, err := tool.Run(tools.DefaultExecutionContext(), map[string]string{
		"question":      "Pick",
		"question_type": "multi_choice",
		"options":       `["A","B","C"]`,
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	var response askUserQuestionResponse
	if err := json.Unmarshal([]byte(result), &response); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if len(response.Answers) != 2 || response.Answers[0] != "A" || response.Answers[1] != "C" {
		t.Errorf("Answers = %v, want [A C]", response.Answers)
	}
}

func TestAskUserQuestionTool_Run_OptionsValidation(t *testing.T) {
	tool := &AskUserQuestionTool{}
	_, _, err := tool.Run(tools.DefaultExecutionContext(), map[string]string{
		"question":      "Choose",
		"question_type": "single_choice",
	})
	if err == nil {
		t.Error("Expected error for missing options")
	}
}

func TestAskUserQuestionTool_Run_OptionsParseFallback(t *testing.T) {
	setTestReader(t, "\n")

	tool := &AskUserQuestionTool{}
	result, _, err := tool.Run(tools.DefaultExecutionContext(), map[string]string{
		"question":      "Choose",
		"question_type": "single_choice",
		"options":       "Yes,No",
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	var response askUserQuestionResponse
	if err := json.Unmarshal([]byte(result), &response); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if response.Answer != "Yes,No" {
		t.Errorf("Answer = %s, want %s (fallback)", response.Answer, "Yes,No")
	}
}
