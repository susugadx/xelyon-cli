package planning

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/uiprompt"
	"github.com/susugadx/xelyon-cli/internal/uiruntime"
)

type askUserQuestionResponse struct {
	Question     string   `json:"question"`
	QuestionType string   `json:"question_type"`
	Answer       string   `json:"answer"`
	Answers      []string `json:"answers"`
}

func newTestExecCtx(input string, stdout io.Writer) tools.ExecutionContext {
	return tools.ExecutionContext{
		Stdin:  strings.NewReader(input),
		Stdout: stdout,
		Stderr: io.Discard,
	}
}

type askQuestionPrompter struct {
	reqs []uiprompt.PromptRequest
	resp uiprompt.PromptResponse
}

func (p *askQuestionPrompter) Prompt(_ context.Context, req uiprompt.PromptRequest) (uiprompt.PromptResponse, error) {
	p.reqs = append(p.reqs, req)
	return p.resp, nil
}

func TestAskUserQuestionTool_Run_SingleChoice(t *testing.T) {
	tool := &AskUserQuestionTool{}
	result, _, err := tool.Run(newTestExecCtx("2\n", io.Discard), map[string]string{
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
	tool := &AskUserQuestionTool{}
	result, _, err := tool.Run(newTestExecCtx("1,3\n", io.Discard), map[string]string{
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
	_, _, err := tool.Run(tools.ExecutionContext{}, map[string]string{
		"question":      "Choose",
		"question_type": "single_choice",
	})
	if err == nil {
		t.Error("Expected error for missing options")
	}
}

func TestAskUserQuestionTool_Run_OptionsParseFallback(t *testing.T) {
	tool := &AskUserQuestionTool{}
	result, _, err := tool.Run(newTestExecCtx("\n", io.Discard), map[string]string{
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

func TestAskUserQuestionTool_Run_DiscardDoesNotLeakProcessStdout(t *testing.T) {
	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() error = %v", err)
	}
	os.Stdout = w
	t.Cleanup(func() {
		os.Stdout = oldStdout
	})

	tool := &AskUserQuestionTool{}
	result, _, err := tool.Run(newTestExecCtx("2\n", io.Discard), map[string]string{
		"question":      "Choose",
		"question_type": "single_choice",
		"options":       `["Yes","No"]`,
	})
	_ = w.Close()
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	var stdout bytes.Buffer
	_, _ = io.Copy(&stdout, r)
	if stdout.Len() != 0 {
		t.Fatalf("expected no process stdout leak, got %q", stdout.String())
	}

	var response askUserQuestionResponse
	if err := json.Unmarshal([]byte(result), &response); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if response.Answer != "No" {
		t.Fatalf("Answer = %s, want No", response.Answer)
	}
}

func TestAskUserQuestionTool_Run_UsesPrompter(t *testing.T) {
	tool := &AskUserQuestionTool{}
	prompter := &askQuestionPrompter{resp: uiprompt.PromptResponse{Value: "No"}}
	runtime := uiruntime.NewRuntime(strings.NewReader("Yes\n"), io.Discard, io.Discard)
	runtime.SetPrompter(prompter)
	execCtx := tools.ExecutionContext{
		Runtime: runtime,
		Stdin:   runtime.Input(),
		Stdout:  io.Discard,
		Stderr:  io.Discard,
	}

	result, _, err := tool.Run(execCtx, map[string]string{
		"question":      "Choose",
		"question_type": "single_choice",
		"options":       `["Yes","No"]`,
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if len(prompter.reqs) != 1 {
		t.Fatalf("prompt calls = %d, want 1", len(prompter.reqs))
	}
	if prompter.reqs[0].Kind != uiprompt.PromptKindSingleChoice {
		t.Fatalf("Kind = %q, want single_choice", prompter.reqs[0].Kind)
	}

	var response askUserQuestionResponse
	if err := json.Unmarshal([]byte(result), &response); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if response.Answer != "No" {
		t.Fatalf("Answer = %s, want No", response.Answer)
	}
}

func TestAskUserQuestionTool_Run_MultiChoicePrompter(t *testing.T) {
	tool := &AskUserQuestionTool{}
	prompter := &askQuestionPrompter{resp: uiprompt.PromptResponse{Values: []string{"A", "C"}}}
	runtime := uiruntime.NewRuntime(strings.NewReader(""), io.Discard, io.Discard)
	runtime.SetPrompter(prompter)

	result, _, err := tool.Run(tools.ExecutionContext{
		Runtime: runtime,
		Stdin:   runtime.Input(),
		Stdout:  io.Discard,
		Stderr:  io.Discard,
	}, map[string]string{
		"question":      "Pick",
		"question_type": "multi_choice",
		"options":       `["A","B","C"]`,
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if len(prompter.reqs) != 1 || prompter.reqs[0].Kind != uiprompt.PromptKindMultiChoice {
		t.Fatalf("prompt requests = %#v, want multi_choice", prompter.reqs)
	}

	var response askUserQuestionResponse
	if err := json.Unmarshal([]byte(result), &response); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if len(response.Answers) != 2 || response.Answers[0] != "A" || response.Answers[1] != "C" {
		t.Fatalf("Answers = %v, want [A C]", response.Answers)
	}
}

func TestAskUserQuestionTool_Run_FreeTextPrompter(t *testing.T) {
	tool := &AskUserQuestionTool{}
	prompter := &askQuestionPrompter{resp: uiprompt.PromptResponse{Text: "details"}}
	runtime := uiruntime.NewRuntime(strings.NewReader(""), io.Discard, io.Discard)
	runtime.SetPrompter(prompter)

	result, _, err := tool.Run(tools.ExecutionContext{
		Runtime: runtime,
		Stdin:   runtime.Input(),
		Stdout:  io.Discard,
		Stderr:  io.Discard,
	}, map[string]string{
		"question":      "Explain",
		"question_type": "free_text",
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if len(prompter.reqs) != 1 || prompter.reqs[0].Kind != uiprompt.PromptKindText {
		t.Fatalf("prompt requests = %#v, want text", prompter.reqs)
	}

	var response askUserQuestionResponse
	if err := json.Unmarshal([]byte(result), &response); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if response.Answer != "details" {
		t.Fatalf("Answer = %s, want details", response.Answer)
	}
}
