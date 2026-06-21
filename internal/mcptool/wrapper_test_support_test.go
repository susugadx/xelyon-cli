package mcptool

import (
	"bytes"
	"context"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/uiprompt"
	"github.com/susugadx/xelyon-cli/internal/uiruntime"
	"io"
	"strings"
)

type testCaller struct{}

func (testCaller) CallTool(context.Context, string, string, map[string]any) (string, error) {
	return "ok", nil
}

type recordingCaller struct {
	calls int
}

func (c *recordingCaller) CallTool(context.Context, string, string, map[string]any) (string, error) {
	c.calls++
	return "called", nil
}

type recordingPrompter struct {
	calls int
}

func (p *recordingPrompter) Prompt(context.Context, uiprompt.PromptRequest) (uiprompt.PromptResponse, error) {
	p.calls++
	return uiprompt.PromptResponse{Action: uiprompt.PromptActionYes}, nil
}

type responsePrompter struct {
	calls int
	resp  uiprompt.PromptResponse
}

func (p *responsePrompter) Prompt(context.Context, uiprompt.PromptRequest) (uiprompt.PromptResponse, error) {
	p.calls++
	return p.resp, nil
}

type argsRecordingCaller struct {
	calls int
	args  map[string]any
}

func (c *argsRecordingCaller) CallTool(_ context.Context, _, _ string, args map[string]any) (string, error) {
	c.calls++
	c.args = args
	return "called", nil
}

type contextWaitingCaller struct {
	calls int
}

func (c *contextWaitingCaller) CallTool(ctx context.Context, _, _ string, _ map[string]any) (string, error) {
	c.calls++
	<-ctx.Done()
	return "", ctx.Err()
}

func newAutoApprovedExecutionContext(ctx context.Context) tools.ExecutionContext {
	runtime := uiruntime.NewRuntime(strings.NewReader(""), io.Discard, io.Discard)
	return tools.ExecutionContext{
		Context:     ctx,
		Stdin:       runtime.Input(),
		Stdout:      runtime.Output(),
		Stderr:      runtime.ErrorOutput(),
		Runtime:     runtime,
		Config:      config.DefaultConfig(),
		AutoApprove: true,
	}
}

func newPromptedExecutionContext(ctx context.Context, prompter uiprompt.Prompter) tools.ExecutionContext {
	var stdout bytes.Buffer
	runtime := uiruntime.NewRuntime(strings.NewReader(""), &stdout, &stdout)
	runtime.SetPrompter(prompter)
	return tools.ExecutionContext{
		Context: ctx,
		Stdin:   runtime.Input(),
		Stdout:  runtime.Output(),
		Stderr:  runtime.ErrorOutput(),
		Runtime: runtime,
		Config:  config.DefaultConfig(),
	}
}
