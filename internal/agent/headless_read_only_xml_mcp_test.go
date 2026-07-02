package agent

import (
	"context"
	"strings"
	"testing"
)

func TestHeadlessReadOnlyStrictDeniesXMLMCPToolCall(t *testing.T) {
	result := runHeadlessXMLMCPAttempt(t, HeadlessRunOptions{
		FailOnToolError: true,
		ReadOnly:        true,
	})

	if result.Status != HeadlessStatusError {
		t.Fatalf("Status = %q, want error", result.Status)
	}
	if result.Error == nil || result.Error.Type != HeadlessErrorTypeReadOnlyViolation {
		t.Fatalf("Error = %+v, want %s", result.Error, HeadlessErrorTypeReadOnlyViolation)
	}
	if result.FailureReason != HeadlessFailureReasonReadOnlyViolation {
		t.Fatalf("FailureReason = %q, want %q", result.FailureReason, HeadlessFailureReasonReadOnlyViolation)
	}
	if result.Response != "final response after denied XML MCP" {
		t.Fatalf("Response = %q, want preserved final response", result.Response)
	}
	assertDeniedXMLMCPToolCall(t, result)
}

func TestHeadlessReadOnlyNonStrictRecordsXMLMCPDenialAndSucceeds(t *testing.T) {
	result := runHeadlessXMLMCPAttempt(t, HeadlessRunOptions{
		ReadOnly: true,
	})

	if result.Status != HeadlessStatusSuccess {
		t.Fatalf("Status = %q, want success without FailOnToolError: %+v", result.Status, result.Error)
	}
	if result.FailureReason != "" {
		t.Fatalf("FailureReason = %q, want empty", result.FailureReason)
	}
	if result.Response != "final response after denied XML MCP" {
		t.Fatalf("Response = %q, want final response", result.Response)
	}
	assertDeniedXMLMCPToolCall(t, result)
}

func TestHeadlessNormalModeKeepsUnknownXMLMCPAsFinalResponse(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	testSubDir(t)

	xmlAttempt := headlessXMLMCPAttemptResponse()
	provider := &sequenceMockProvider{
		name:      "openai",
		responses: []string{xmlAttempt},
	}
	result := RunHeadlessWithConfigOptions(context.Background(), "mention mcp xml", "gpt-5.4", provider, newProjectMapDisabledConfig(), HeadlessRunOptions{})

	if result.Status != HeadlessStatusSuccess {
		t.Fatalf("Status = %q, want success: %+v", result.Status, result.Error)
	}
	if result.Response != xmlAttempt {
		t.Fatalf("Response = %q, want raw XML response", result.Response)
	}
	if len(result.ToolCalls) != 0 {
		t.Fatalf("ToolCalls = %+v, want no parsed tool calls", result.ToolCalls)
	}
}

func TestHeadlessReadOnlyStrictKeepsXMLMCPExampleInCodeBlockAsFinalResponse(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	testSubDir(t)

	response := "Example:\n```xml\n" + headlessXMLMCPAttemptResponse() + "\n```\n"
	provider := &sequenceMockProvider{
		name:      "openai",
		responses: []string{response},
	}
	result := RunHeadlessWithConfigOptions(context.Background(), "show mcp xml example", "gpt-5.4", provider, newProjectMapDisabledConfig(), HeadlessRunOptions{
		FailOnToolError: true,
		ReadOnly:        true,
	})

	if result.Status != HeadlessStatusSuccess {
		t.Fatalf("Status = %q, want success: %+v", result.Status, result.Error)
	}
	if result.Response != response {
		t.Fatalf("Response = %q, want raw code block response", result.Response)
	}
	if len(result.ToolCalls) != 0 {
		t.Fatalf("ToolCalls = %+v, want no denied call for code block example", result.ToolCalls)
	}
}

func TestHeadlessReadOnlyStrictKeepsUnmatchedXMLMCPAsFinalResponse(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	testSubDir(t)

	response := "<mcp_github_create_issue><title>missing close</title>"
	provider := &sequenceMockProvider{
		name:      "openai",
		responses: []string{response},
	}
	result := RunHeadlessWithConfigOptions(context.Background(), "show incomplete mcp xml", "gpt-5.4", provider, newProjectMapDisabledConfig(), HeadlessRunOptions{
		FailOnToolError: true,
		ReadOnly:        true,
	})

	if result.Status != HeadlessStatusSuccess {
		t.Fatalf("Status = %q, want success: %+v", result.Status, result.Error)
	}
	if result.Response != response {
		t.Fatalf("Response = %q, want unmatched XML response", result.Response)
	}
	if len(result.ToolCalls) != 0 {
		t.Fatalf("ToolCalls = %+v, want no denied call for unmatched XML", result.ToolCalls)
	}
}

func TestHeadlessReadOnlyStrictKeepsViolationWhenLoopLimitReached(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	testSubDir(t)

	cfg := newProjectMapDisabledConfig()
	cfg.General.ToolLoopLimit = 1
	provider := &sequenceMockProvider{
		name:      "openai",
		responses: []string{headlessXMLMCPAttemptResponse()},
	}
	result := RunHeadlessWithConfigOptions(context.Background(), "attempt xml mcp", "gpt-5.4", provider, cfg, HeadlessRunOptions{
		FailOnToolError: true,
		ReadOnly:        true,
	})

	if result.Status != HeadlessStatusError {
		t.Fatalf("Status = %q, want error", result.Status)
	}
	if result.Error == nil || result.Error.Type != HeadlessErrorTypeReadOnlyViolation {
		t.Fatalf("Error = %+v, want %s", result.Error, HeadlessErrorTypeReadOnlyViolation)
	}
	if result.FailureReason != HeadlessFailureReasonReadOnlyViolation {
		t.Fatalf("FailureReason = %q, want %q", result.FailureReason, HeadlessFailureReasonReadOnlyViolation)
	}
	assertDeniedXMLMCPToolCall(t, result)
}

func runHeadlessXMLMCPAttempt(t *testing.T, options HeadlessRunOptions) *HeadlessResult {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	testSubDir(t)

	provider := &sequenceMockProvider{
		name: "openai",
		responses: []string{
			headlessXMLMCPAttemptResponse(),
			"final response after denied XML MCP",
		},
	}
	return RunHeadlessWithConfigOptions(context.Background(), "attempt xml mcp", "gpt-5.4", provider, newProjectMapDisabledConfig(), options)
}

func headlessXMLMCPAttemptResponse() string {
	return "<mcp_github_create_issue><title>should not run</title></mcp_github_create_issue>"
}

func assertDeniedXMLMCPToolCall(t *testing.T, result *HeadlessResult) {
	t.Helper()
	if len(result.ToolCalls) != 1 {
		t.Fatalf("ToolCalls length = %d, want 1", len(result.ToolCalls))
	}
	call := result.ToolCalls[0]
	if call.Tool != "mcp_github_create_issue" || call.Success {
		t.Fatalf("ToolCalls[0] = %+v, want failed XML MCP call", call)
	}
	if len(call.Args) != 0 {
		t.Fatalf("ToolCalls[0].Args = %+v, want empty args for XML MCP denial", call.Args)
	}
	if !strings.Contains(call.Output, "read-only mode denied MCP tool") {
		t.Fatalf("ToolCalls[0].Output = %q, want MCP read-only denial", call.Output)
	}
}
