package agent

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/tools"
)

type approvedPlanWriteTool struct{}

func (*approvedPlanWriteTool) Name() string { return "approved_plan_test_write" }

func (*approvedPlanWriteTool) Description() string {
	return "records a synthetic file change for approved-plan tests"
}

func (*approvedPlanWriteTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"path": map[string]interface{}{"type": "string"},
		},
	}
}

func (*approvedPlanWriteTool) Run(_ tools.ExecutionContext, args map[string]string) (string, *tools.FileChange, error) {
	path := args["path"]
	if path == "" {
		path = "internal/agent/approved_plan_target.go"
	}
	return "recorded synthetic change", &tools.FileChange{
		FilePath: path,
		Tool:     "approved_plan_test_write",
		Details: []tools.FileChangeDetail{{
			FilePath: path,
			Action:   "modified",
		}},
	}, nil
}

func registerApprovedPlanWriteTool(agent *Agent) {
	agent.registry().Register(&approvedPlanWriteTool{})
}

func approvedPlanWriteCall(path string) string {
	if path == "" {
		path = "internal/agent/approved_plan_target.go"
	}
	return fmt.Sprintf("{\"tool\": \"approved_plan_test_write\", \"args\": {\"path\": %q}}", path)
}

type approvedPlanSilentWriteTool struct{}

func (*approvedPlanSilentWriteTool) Name() string { return "approved_plan_silent_write" }

func (*approvedPlanSilentWriteTool) Description() string {
	return "writes a file without returning FileChange, simulating bash-style edits"
}

func (*approvedPlanSilentWriteTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"path":    map[string]interface{}{"type": "string"},
			"content": map[string]interface{}{"type": "string"},
		},
	}
}

func (*approvedPlanSilentWriteTool) Run(_ tools.ExecutionContext, args map[string]string) (string, *tools.FileChange, error) {
	path := args["path"]
	if path == "" {
		path = "internal/agent/approved_plan_silent_target.go"
	}
	content := args["content"]
	if content == "" {
		content = "package main\n\nfunc main() { println(\"fixed\") }\n"
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return "", nil, err
	}
	return "wrote file without reported change", nil, nil
}

func approvedPlanSilentWriteCall(path, content string) string {
	if path == "" {
		path = "internal/agent/approved_plan_silent_target.go"
	}
	return fmt.Sprintf("{\"tool\": \"approved_plan_silent_write\", \"args\": {\"path\": %q, \"content\": %q}}", path, content)
}

func TestExecuteChatRequest_ApprovedPlanHandoffIsInjectedOnce(t *testing.T) {
	disableColors(t)

	var out bytes.Buffer
	var capturedHistories [][]api.Message
	provider := &scriptedChatProvider{
		name:            "test",
		functionCalling: true,
		chatWithToolsFn: func(call int, ctx context.Context, systemPrompt string, history []api.Message, model string) (string, error) {
			snapshot := append([]api.Message(nil), history...)
			capturedHistories = append(capturedHistories, snapshot)
			switch call {
			case 0:
				return approvedPlanWriteCall("internal/agent/approved_plan_target.go"), nil
			default:
				return "The requested changes are done.", nil
			}
		},
	}
	agent := newChatRequestTestAgent(t, provider, &out)
	registerApprovedPlanWriteTool(agent)
	agent.PendingApprovedPlan = "Implementation Plan\n1. Update the target file"

	req := &chatRequest{input: "implement it"}
	if err := agent.executeChatRequest(context.Background(), req); err != nil {
		t.Fatalf("executeChatRequest() error = %v", err)
	}

	if req.approvedPlanHandoff == "" {
		t.Fatal("approvedPlanHandoff should be captured for the request")
	}
	if agent.PendingApprovedPlan != "" {
		t.Fatalf("PendingApprovedPlan = %q, want empty after successful completion", agent.PendingApprovedPlan)
	}
	if provider.callCount != 2 {
		t.Fatalf("provider.callCount = %d, want 2", provider.callCount)
	}
	firstTurn := capturedHistories[0]
	if got := firstTurn[len(firstTurn)-1].Content; !strings.Contains(got, "The user approved this plan in the previous /plan turn.") {
		t.Fatalf("expected approved plan guidance in provider history, got %q", got)
	}
	if got := firstTurn[len(firstTurn)-1].Content; !strings.Contains(got, "Implementation Plan") {
		t.Fatalf("expected approved plan text in provider history, got %q", got)
	}
	if got := firstTurn[len(firstTurn)-1].Content; !strings.Contains(got, "[NORMAL MODE]") {
		t.Fatalf("expected normal mode prompt in provider history, got %q", got)
	}
	for _, msg := range agent.History {
		if strings.Contains(msg.Content, "[APPROVED PLAN HANDOFF]") {
			t.Fatalf("approved plan handoff should not persist in history, got %#v", agent.History)
		}
	}

	req2 := &chatRequest{input: "continue"}
	if err := agent.executeChatRequest(context.Background(), req2); err != nil {
		t.Fatalf("second executeChatRequest() error = %v", err)
	}

	if req2.approvedPlanHandoff != "" {
		t.Fatalf("second request approvedPlanHandoff = %q, want empty", req2.approvedPlanHandoff)
	}
	if provider.callCount != 3 {
		t.Fatalf("provider.callCount = %d, want 3", provider.callCount)
	}
	for _, msg := range capturedHistories[2] {
		if strings.Contains(msg.Content, "[APPROVED PLAN HANDOFF]") {
			t.Fatalf("approved plan handoff should be injected only once, got %#v", capturedHistories[2])
		}
	}
}

func TestExecuteChatRequest_ApprovedPlanHandoffPersistsAfterCompletionWithoutTaskChanges(t *testing.T) {
	disableColors(t)

	var out bytes.Buffer
	var capturedHistories [][]api.Message
	provider := &scriptedChatProvider{
		name:            "test",
		functionCalling: true,
		chatWithToolsFn: func(call int, ctx context.Context, systemPrompt string, history []api.Message, model string) (string, error) {
			snapshot := append([]api.Message(nil), history...)
			capturedHistories = append(capturedHistories, snapshot)
			switch call {
			case 0:
				return "Done.", nil
			case 1:
				return approvedPlanWriteCall("internal/agent/approved_plan_followup.go"), nil
			default:
				return "The requested changes are done.", nil
			}
		},
	}
	agent := newChatRequestTestAgent(t, provider, &out)
	registerApprovedPlanWriteTool(agent)
	agent.PendingApprovedPlan = "Implementation Plan\n1. Update the target file"

	req1 := &chatRequest{input: "implement it"}
	if err := agent.executeChatRequest(context.Background(), req1); err != nil {
		t.Fatalf("first executeChatRequest() error = %v", err)
	}
	if agent.PendingApprovedPlan != "" {
		t.Fatalf("PendingApprovedPlan = %q, want empty after first successful handoff turn", agent.PendingApprovedPlan)
	}
	if req1.approvedPlanHandoff == "" {
		t.Fatal("approvedPlanHandoff should be captured for the first request")
	}

	req2 := &chatRequest{input: "try again"}
	if err := agent.executeChatRequest(context.Background(), req2); err != nil {
		t.Fatalf("second executeChatRequest() error = %v", err)
	}
	if req2.approvedPlanHandoff != "" {
		t.Fatalf("approvedPlanHandoff = %q, want empty after one-shot handoff is consumed", req2.approvedPlanHandoff)
	}
	if got := capturedHistories[1][len(capturedHistories[1])-1].Content; strings.Contains(got, "[APPROVED PLAN HANDOFF]") {
		t.Fatalf("approved plan handoff should not be reinjected after the first successful turn, got %q", got)
	}
}

func TestChatCore_ApprovedPlanHandoffAndVerificationKeepHandoffForSilentWriteToolInGitRepo(t *testing.T) {
	disableColors(t)

	repoDir, filePath := newCommittedGitRepo(t)
	chdirForTest(t, repoDir)

	var out bytes.Buffer
	cfg := newChatRequestTestConfig()
	cfg.FinalChecks.Commands = []string{fmt.Sprintf("grep -q fixed %q", filePath)}
	cfg.FinalChecks.Timeout = 10

	provider := &scriptedChatProvider{
		name:            "test",
		functionCalling: true,
		chatWithToolsFn: func(call int, ctx context.Context, systemPrompt string, history []api.Message, model string) (string, error) {
			switch call {
			case 0:
				return approvedPlanSilentWriteCall(filePath, "package main\n\nfunc main() { println(\"fixed\") }\n"), nil
			default:
				return "The requested changes are done.", nil
			}
		},
	}

	t.Setenv("HOME", t.TempDir())
	agent := newTurnRunnerTestAgent(provider, cfg, "", &out, &approvedPlanSilentWriteTool{})
	agent.PendingApprovedPlan = "Implementation Plan\n1. Update main.go"

	if err := agent.chatCore("implement approved plan", nil, false); err != nil {
		t.Fatalf("chatCore() error = %v", err)
	}
	if agent.PendingApprovedPlan != "" {
		t.Fatalf("PendingApprovedPlan = %q, want empty after one-shot handoff turn", agent.PendingApprovedPlan)
	}
	if agent.taskTestResult != nil {
		t.Fatalf("taskTestResult = %v, want nil because final checks should be skipped", agent.taskTestResult)
	}
	if strings.Contains(out.String(), "Final check passed") {
		t.Fatalf("final checks should be skipped without recorded changes, got %q", out.String())
	}
}

func TestChatCore_ApprovedPlanHandoffAndVerificationKeepHandoffForSilentWriteToolOutsideGit(t *testing.T) {
	disableColors(t)

	workDir := t.TempDir()
	chdirForTest(t, workDir)
	filePath := workDir + "/main.go"

	var out bytes.Buffer
	cfg := newChatRequestTestConfig()
	cfg.FinalChecks.Commands = []string{fmt.Sprintf("grep -q fixed %q", filePath)}
	cfg.FinalChecks.Timeout = 10

	provider := &scriptedChatProvider{
		name:            "test",
		functionCalling: true,
		chatWithToolsFn: func(call int, ctx context.Context, systemPrompt string, history []api.Message, model string) (string, error) {
			switch call {
			case 0:
				return approvedPlanSilentWriteCall(filePath, "package main\n\nfunc main() { println(\"fixed\") }\n"), nil
			default:
				return "The requested changes are done.", nil
			}
		},
	}

	t.Setenv("HOME", t.TempDir())
	agent := newTurnRunnerTestAgent(provider, cfg, "", &out, &approvedPlanSilentWriteTool{})
	agent.PendingApprovedPlan = "Implementation Plan\n1. Update main.go"

	if err := agent.chatCore("implement approved plan", nil, false); err != nil {
		t.Fatalf("chatCore() error = %v", err)
	}
	if agent.PendingApprovedPlan != "" {
		t.Fatalf("PendingApprovedPlan = %q, want empty after one-shot handoff turn", agent.PendingApprovedPlan)
	}
	if agent.taskTestResult != nil {
		t.Fatalf("taskTestResult = %v, want nil because final checks should be skipped", agent.taskTestResult)
	}
	if strings.Contains(out.String(), "Final check passed") {
		t.Fatalf("final checks should be skipped without recorded changes, got %q", out.String())
	}
}

func TestExecuteChatRequest_ApprovedPlanHandoffPersistsAfterFailureAndReinjectsOnRetry(t *testing.T) {
	disableColors(t)

	var out bytes.Buffer
	var capturedHistories [][]api.Message
	provider := &scriptedChatProvider{
		name:            "test",
		functionCalling: true,
		chatWithToolsFn: func(call int, ctx context.Context, systemPrompt string, history []api.Message, model string) (string, error) {
			snapshot := append([]api.Message(nil), history...)
			capturedHistories = append(capturedHistories, snapshot)
			if call == 0 {
				return "", errors.New("temporary api error")
			}
			if call == 1 {
				return approvedPlanWriteCall("internal/agent/approved_plan_retry.go"), nil
			}
			return "The requested changes are done.", nil
		},
	}
	agent := newChatRequestTestAgent(t, provider, &out)
	registerApprovedPlanWriteTool(agent)
	agent.PendingApprovedPlan = "Implementation Plan\n1. Update the target file"

	req1 := &chatRequest{input: "implement it"}
	err := agent.executeChatRequest(context.Background(), req1)
	if err == nil || !strings.Contains(err.Error(), "temporary api error") {
		t.Fatalf("first executeChatRequest() error = %v, want temporary api error", err)
	}
	if agent.PendingApprovedPlan == "" {
		t.Fatal("PendingApprovedPlan should remain after failed request")
	}
	if req1.approvedPlanHandoff == "" {
		t.Fatal("approvedPlanHandoff should be captured even on failure")
	}
	if got := capturedHistories[0][len(capturedHistories[0])-1].Content; !strings.Contains(got, "[APPROVED PLAN HANDOFF]") {
		t.Fatalf("expected handoff guidance in failed request history, got %q", got)
	}

	req2 := &chatRequest{input: "implement it again"}
	if err := agent.executeChatRequest(context.Background(), req2); err != nil {
		t.Fatalf("second executeChatRequest() error = %v", err)
	}
	if req2.approvedPlanHandoff == "" {
		t.Fatal("approvedPlanHandoff should be captured on retry")
	}
	if agent.PendingApprovedPlan != "" {
		t.Fatalf("PendingApprovedPlan = %q, want empty after retry succeeds", agent.PendingApprovedPlan)
	}
	if provider.callCount != 3 {
		t.Fatalf("provider.callCount = %d, want 3", provider.callCount)
	}
	if got := capturedHistories[1][len(capturedHistories[1])-1].Content; !strings.Contains(got, "[APPROVED PLAN HANDOFF]") {
		t.Fatalf("expected handoff guidance in retry history, got %q", got)
	}
}

func TestExecuteChatRequest_ApprovedPlanHandoffPersistsAfterNonCompletionResponse(t *testing.T) {
	disableColors(t)

	var out bytes.Buffer
	var capturedHistories [][]api.Message
	provider := &scriptedChatProvider{
		name:            "test",
		functionCalling: true,
		chatWithToolsFn: func(call int, ctx context.Context, systemPrompt string, history []api.Message, model string) (string, error) {
			snapshot := append([]api.Message(nil), history...)
			capturedHistories = append(capturedHistories, snapshot)
			if call == 0 {
				return "Would you like me to proceed with the implementation?", nil
			}
			if call == 1 {
				return approvedPlanWriteCall("internal/agent/approved_plan_non_completion.go"), nil
			}
			return "The requested changes are done.", nil
		},
	}
	agent := newChatRequestTestAgent(t, provider, &out)
	registerApprovedPlanWriteTool(agent)
	agent.PendingApprovedPlan = "Implementation Plan\n1. Update the target file"

	req1 := &chatRequest{input: "implement it"}
	if err := agent.executeChatRequest(context.Background(), req1); err != nil {
		t.Fatalf("first executeChatRequest() error = %v", err)
	}
	if agent.PendingApprovedPlan != "" {
		t.Fatalf("PendingApprovedPlan = %q, want empty after first successful handoff turn", agent.PendingApprovedPlan)
	}
	if req1.approvedPlanHandoff == "" {
		t.Fatal("approvedPlanHandoff should be captured for non-completion response")
	}
	if got := capturedHistories[0][len(capturedHistories[0])-1].Content; !strings.Contains(got, "[APPROVED PLAN HANDOFF]") {
		t.Fatalf("expected handoff guidance in first request history, got %q", got)
	}

	req2 := &chatRequest{input: "continue"}
	if err := agent.executeChatRequest(context.Background(), req2); err != nil {
		t.Fatalf("second executeChatRequest() error = %v", err)
	}
	if req2.approvedPlanHandoff != "" {
		t.Fatalf("approvedPlanHandoff = %q, want empty after one-shot handoff is consumed", req2.approvedPlanHandoff)
	}
	if got := capturedHistories[1][len(capturedHistories[1])-1].Content; strings.Contains(got, "[APPROVED PLAN HANDOFF]") {
		t.Fatalf("approved plan handoff should not be reinjected after the first successful turn, got %q", got)
	}
}

func TestExecuteChatRequest_ApprovedPlanHandoffPersistsAfterClarificationWithoutQuestionMark(t *testing.T) {
	disableColors(t)

	var out bytes.Buffer
	var capturedHistories [][]api.Message
	provider := &scriptedChatProvider{
		name:            "test",
		functionCalling: true,
		chatWithToolsFn: func(call int, ctx context.Context, systemPrompt string, history []api.Message, model string) (string, error) {
			snapshot := append([]api.Message(nil), history...)
			capturedHistories = append(capturedHistories, snapshot)
			if call == 0 {
				return "Please provide the file path.", nil
			}
			if call == 1 {
				return approvedPlanWriteCall("internal/agent/approved_plan_clarification.go"), nil
			}
			return "The requested changes are done.", nil
		},
	}
	agent := newChatRequestTestAgent(t, provider, &out)
	registerApprovedPlanWriteTool(agent)
	agent.PendingApprovedPlan = "Implementation Plan\n1. Update the target file"

	req1 := &chatRequest{input: "implement it"}
	if err := agent.executeChatRequest(context.Background(), req1); err != nil {
		t.Fatalf("first executeChatRequest() error = %v", err)
	}
	if agent.PendingApprovedPlan != "" {
		t.Fatalf("PendingApprovedPlan = %q, want empty after first successful handoff turn", agent.PendingApprovedPlan)
	}
	if req1.approvedPlanHandoff == "" {
		t.Fatal("approvedPlanHandoff should be captured for clarification-only response")
	}

	req2 := &chatRequest{input: "the path is internal/agent/plan_request.go"}
	if err := agent.executeChatRequest(context.Background(), req2); err != nil {
		t.Fatalf("second executeChatRequest() error = %v", err)
	}
	if req2.approvedPlanHandoff != "" {
		t.Fatalf("approvedPlanHandoff = %q, want empty after one-shot handoff is consumed", req2.approvedPlanHandoff)
	}
	if got := capturedHistories[1][len(capturedHistories[1])-1].Content; strings.Contains(got, "[APPROVED PLAN HANDOFF]") {
		t.Fatalf("approved plan handoff should not be reinjected after the first successful turn, got %q", got)
	}
}

func TestExecuteChatRequest_ApprovedPlanHandoffClearsAfterFollowUpCompletionWithoutNewEdits(t *testing.T) {
	disableColors(t)

	var out bytes.Buffer
	cfg := newChatRequestTestConfig()
	cfg.FinalChecks.Commands = []string{`test "$XELYON_CHANGED_FILES" = "internal/agent/approved_plan_followup_closure.go"`}
	cfg.FinalChecks.Timeout = 10
	var capturedHistories [][]api.Message
	provider := &scriptedChatProvider{
		name:            "test",
		functionCalling: true,
		chatWithToolsFn: func(call int, ctx context.Context, systemPrompt string, history []api.Message, model string) (string, error) {
			snapshot := append([]api.Message(nil), history...)
			capturedHistories = append(capturedHistories, snapshot)
			switch call {
			case 0:
				return approvedPlanWriteCall("internal/agent/approved_plan_followup_closure.go"), nil
			case 1:
				return "Please provide the exact module path.", nil
			default:
				return "Done. The requested changes are complete.", nil
			}
		},
	}
	agent := newTurnRunnerTestAgent(provider, cfg, "", &out, &approvedPlanWriteTool{})
	agent.PendingApprovedPlan = "Implementation Plan\n1. Update the target file"

	req1 := &chatRequest{input: "implement it"}
	if err := agent.executeChatRequest(context.Background(), req1); err != nil {
		t.Fatalf("first executeChatRequest() error = %v", err)
	}
	if agent.PendingApprovedPlan != "" {
		t.Fatalf("PendingApprovedPlan = %q, want empty after first successful handoff turn", agent.PendingApprovedPlan)
	}
	if agent.PendingApprovedPlanHasChanges {
		t.Fatal("PendingApprovedPlanHasChanges = true, want false after one-shot handoff is consumed")
	}
	if req1.approvedPlanHandoff == "" {
		t.Fatal("approvedPlanHandoff should be captured for clarification turn")
	}

	req2 := &chatRequest{input: "the module path is foo/bar"}
	if err := agent.executeChatRequest(context.Background(), req2); err != nil {
		t.Fatalf("second executeChatRequest() error = %v", err)
	}
	if req2.approvedPlanHandoff != "" {
		t.Fatalf("approvedPlanHandoff = %q, want empty after one-shot handoff is consumed", req2.approvedPlanHandoff)
	}
	if len(capturedHistories) != 3 {
		t.Fatalf("capturedHistories = %d, want 3", len(capturedHistories))
	}
	if got := capturedHistories[2][len(capturedHistories[2])-1].Content; strings.Contains(got, "[APPROVED PLAN HANDOFF]") {
		t.Fatalf("approved plan handoff should not be reinjected on the follow-up turn, got %q", got)
	}
}

func TestExecuteChatRequest_ApprovedPlanHandoffPersistsAfterShortTextPlanResponse(t *testing.T) {
	disableColors(t)

	var out bytes.Buffer
	var capturedHistories [][]api.Message
	provider := &scriptedChatProvider{
		name:            "test",
		functionCalling: true,
		chatWithToolsFn: func(call int, ctx context.Context, systemPrompt string, history []api.Message, model string) (string, error) {
			snapshot := append([]api.Message(nil), history...)
			capturedHistories = append(capturedHistories, snapshot)
			if call == 0 {
				return "1. Update foo.go\n2. Add tests\n3. Run go test", nil
			}
			if call == 1 {
				return approvedPlanWriteCall("internal/agent/approved_plan_short_text.go"), nil
			}
			return "The requested changes are done.", nil
		},
	}
	agent := newChatRequestTestAgent(t, provider, &out)
	registerApprovedPlanWriteTool(agent)
	agent.PendingApprovedPlan = "Implementation Plan\n1. Update the target file"

	req1 := &chatRequest{input: "implement it"}
	if err := agent.executeChatRequest(context.Background(), req1); err != nil {
		t.Fatalf("first executeChatRequest() error = %v", err)
	}
	if agent.PendingApprovedPlan != "" {
		t.Fatalf("PendingApprovedPlan = %q, want empty after first successful handoff turn", agent.PendingApprovedPlan)
	}
	if req1.approvedPlanHandoff == "" {
		t.Fatal("approvedPlanHandoff should be captured for short text-plan response")
	}

	req2 := &chatRequest{input: "go ahead"}
	if err := agent.executeChatRequest(context.Background(), req2); err != nil {
		t.Fatalf("second executeChatRequest() error = %v", err)
	}
	if req2.approvedPlanHandoff != "" {
		t.Fatalf("approvedPlanHandoff = %q, want empty after one-shot handoff is consumed", req2.approvedPlanHandoff)
	}
	if got := capturedHistories[1][len(capturedHistories[1])-1].Content; strings.Contains(got, "[APPROVED PLAN HANDOFF]") {
		t.Fatalf("approved plan handoff should not be reinjected after the first successful turn, got %q", got)
	}
}

func TestExecuteChatRequest_ApprovedPlanHandoffPersistsAfterFinalSummaryWithoutCompletionKeyword(t *testing.T) {
	disableColors(t)

	var out bytes.Buffer
	var capturedHistories [][]api.Message
	provider := &scriptedChatProvider{
		name:            "test",
		functionCalling: true,
		chatWithToolsFn: func(call int, ctx context.Context, systemPrompt string, history []api.Message, model string) (string, error) {
			snapshot := append([]api.Message(nil), history...)
			capturedHistories = append(capturedHistories, snapshot)
			if call == 0 {
				return "Updated foo.go and added tests.", nil
			}
			if call == 1 {
				return approvedPlanWriteCall("internal/agent/approved_plan_summary.go"), nil
			}
			return "The changes have been completed.", nil
		},
	}
	agent := newChatRequestTestAgent(t, provider, &out)
	registerApprovedPlanWriteTool(agent)
	agent.PendingApprovedPlan = "Implementation Plan\n1. Update the target file"

	req1 := &chatRequest{input: "implement it"}
	if err := agent.executeChatRequest(context.Background(), req1); err != nil {
		t.Fatalf("first executeChatRequest() error = %v", err)
	}
	if agent.PendingApprovedPlan != "" {
		t.Fatalf("PendingApprovedPlan = %q, want empty after first successful handoff turn", agent.PendingApprovedPlan)
	}
	if req1.approvedPlanHandoff == "" {
		t.Fatal("approvedPlanHandoff should be captured for final summary response")
	}

	req2 := &chatRequest{input: "unrelated follow-up"}
	if err := agent.executeChatRequest(context.Background(), req2); err != nil {
		t.Fatalf("second executeChatRequest() error = %v", err)
	}
	if req2.approvedPlanHandoff != "" {
		t.Fatalf("approvedPlanHandoff = %q, want empty after one-shot handoff is consumed", req2.approvedPlanHandoff)
	}
	if len(capturedHistories) != 3 {
		t.Fatalf("capturedHistories = %d, want 3", len(capturedHistories))
	}
	if got := capturedHistories[1][len(capturedHistories[1])-1].Content; strings.Contains(got, "[APPROVED PLAN HANDOFF]") {
		t.Fatalf("approved plan handoff should not be reinjected after the first successful turn, got %q", got)
	}
}

func TestExecuteChatRequest_ApprovedPlanHandoffPersistsAfterPartialProgressCompletionPhrase(t *testing.T) {
	disableColors(t)

	var out bytes.Buffer
	var capturedHistories [][]api.Message
	provider := &scriptedChatProvider{
		name:            "test",
		functionCalling: true,
		chatWithToolsFn: func(call int, ctx context.Context, systemPrompt string, history []api.Message, model string) (string, error) {
			snapshot := append([]api.Message(nil), history...)
			capturedHistories = append(capturedHistories, snapshot)
			switch call {
			case 0:
				return approvedPlanWriteCall("internal/agent/approved_plan_partial_progress.go"), nil
			case 1:
				return "Completed updating foo.go. Next I'll update tests.", nil
			case 2:
				return approvedPlanWriteCall("internal/agent/approved_plan_partial_progress_followup.go"), nil
			default:
				return "The requested changes are done.", nil
			}
		},
	}
	agent := newChatRequestTestAgent(t, provider, &out)
	registerApprovedPlanWriteTool(agent)
	agent.PendingApprovedPlan = "Implementation Plan\n1. Update the target file"

	req1 := &chatRequest{input: "implement it"}
	if err := agent.executeChatRequest(context.Background(), req1); err != nil {
		t.Fatalf("first executeChatRequest() error = %v", err)
	}
	if agent.PendingApprovedPlan != "" {
		t.Fatalf("PendingApprovedPlan = %q, want empty after first successful handoff turn", agent.PendingApprovedPlan)
	}
	if req1.approvedPlanHandoff == "" {
		t.Fatal("approvedPlanHandoff should be captured for partial-progress response")
	}

	req2 := &chatRequest{input: "continue implementation"}
	if err := agent.executeChatRequest(context.Background(), req2); err != nil {
		t.Fatalf("second executeChatRequest() error = %v", err)
	}
	if req2.approvedPlanHandoff != "" {
		t.Fatalf("approvedPlanHandoff = %q, want empty after one-shot handoff is consumed", req2.approvedPlanHandoff)
	}
	if len(capturedHistories) != 4 {
		t.Fatalf("capturedHistories = %d, want 4", len(capturedHistories))
	}
	if got := capturedHistories[2][len(capturedHistories[2])-1].Content; strings.Contains(got, "[APPROVED PLAN HANDOFF]") {
		t.Fatalf("approved plan handoff should not be reinjected after the first successful turn, got %q", got)
	}
}

func TestExecuteChatRequest_ApprovedPlanHandoffPersistsAfterReadOnlyToolTurn(t *testing.T) {
	disableColors(t)

	var out bytes.Buffer
	var capturedHistories [][]api.Message
	tempFile := t.TempDir() + "/target.txt"
	if err := os.WriteFile(tempFile, []byte("hello"), 0600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	provider := &scriptedChatProvider{
		name:            "test",
		functionCalling: true,
		chatWithToolsFn: func(call int, ctx context.Context, systemPrompt string, history []api.Message, model string) (string, error) {
			snapshot := append([]api.Message(nil), history...)
			capturedHistories = append(capturedHistories, snapshot)
			switch call {
			case 0:
				return "{\"tool\": \"read_file\", \"args\": {\"path\": \"" + tempFile + "\"}}", nil
			case 1:
				return "I inspected the related file and next I'll update foo.go.", nil
			case 2:
				return approvedPlanWriteCall("internal/agent/approved_plan_after_read_only.go"), nil
			default:
				return "The requested changes are done.", nil
			}
		},
	}
	agent := newChatRequestTestAgent(t, provider, &out)
	registerApprovedPlanWriteTool(agent)
	agent.PendingApprovedPlan = "Implementation Plan\n1. Update the target file"

	req1 := &chatRequest{input: "implement it"}
	if err := agent.executeChatRequest(context.Background(), req1); err != nil {
		t.Fatalf("first executeChatRequest() error = %v", err)
	}
	if agent.PendingApprovedPlan != "" {
		t.Fatalf("PendingApprovedPlan = %q, want empty after first successful handoff turn", agent.PendingApprovedPlan)
	}
	if req1.approvedPlanHandoff == "" {
		t.Fatal("approvedPlanHandoff should be captured for read-only tool turn")
	}

	req2 := &chatRequest{input: "continue implementation"}
	if err := agent.executeChatRequest(context.Background(), req2); err != nil {
		t.Fatalf("second executeChatRequest() error = %v", err)
	}
	if req2.approvedPlanHandoff != "" {
		t.Fatalf("approvedPlanHandoff = %q, want empty after one-shot handoff is consumed", req2.approvedPlanHandoff)
	}
	if got := capturedHistories[2][len(capturedHistories[2])-1].Content; strings.Contains(got, "[APPROVED PLAN HANDOFF]") {
		t.Fatalf("approved plan handoff should not be reinjected after the first successful turn, got %q", got)
	}
}

func TestExecuteChatRequest_ApprovedPlanHandoffClearsAfterCompletionWithSocialQuestion(t *testing.T) {
	disableColors(t)

	var out bytes.Buffer
	var capturedHistories [][]api.Message
	provider := &scriptedChatProvider{
		name:            "test",
		functionCalling: true,
		chatWithToolsFn: func(call int, ctx context.Context, systemPrompt string, history []api.Message, model string) (string, error) {
			snapshot := append([]api.Message(nil), history...)
			capturedHistories = append(capturedHistories, snapshot)
			switch call {
			case 0:
				return approvedPlanWriteCall("internal/agent/approved_plan_social.go"), nil
			case 1:
				return "Done. Anything else?", nil
			default:
				return "unrelated follow-up", nil
			}
		},
	}
	agent := newChatRequestTestAgent(t, provider, &out)
	registerApprovedPlanWriteTool(agent)
	agent.PendingApprovedPlan = "Implementation Plan\n1. Update the target file"

	req1 := &chatRequest{input: "implement it"}
	if err := agent.executeChatRequest(context.Background(), req1); err != nil {
		t.Fatalf("first executeChatRequest() error = %v", err)
	}
	if agent.PendingApprovedPlan != "" {
		t.Fatalf("PendingApprovedPlan = %q, want empty after completion response with social closing question", agent.PendingApprovedPlan)
	}
	if req1.approvedPlanHandoff == "" {
		t.Fatal("approvedPlanHandoff should be captured for the completion response")
	}

	req2 := &chatRequest{input: "unrelated follow-up"}
	if err := agent.executeChatRequest(context.Background(), req2); err != nil {
		t.Fatalf("second executeChatRequest() error = %v", err)
	}
	if req2.approvedPlanHandoff != "" {
		t.Fatalf("approvedPlanHandoff = %q, want empty after social closing consumed the handoff", req2.approvedPlanHandoff)
	}
	if len(capturedHistories) != 3 {
		t.Fatalf("capturedHistories = %d, want 3", len(capturedHistories))
	}
	for _, msg := range capturedHistories[2] {
		if strings.Contains(msg.Content, "[APPROVED PLAN HANDOFF]") {
			t.Fatalf("approved plan handoff should not be reinjected after completion with social closing question, got %#v", capturedHistories[2])
		}
	}
}

func TestExecuteChatRequest_ApprovedPlanHandoffClearsAfterCompletionWithNumberedSummary(t *testing.T) {
	disableColors(t)

	var out bytes.Buffer
	var capturedHistories [][]api.Message
	provider := &scriptedChatProvider{
		name:            "test",
		functionCalling: true,
		chatWithToolsFn: func(call int, ctx context.Context, systemPrompt string, history []api.Message, model string) (string, error) {
			snapshot := append([]api.Message(nil), history...)
			capturedHistories = append(capturedHistories, snapshot)
			switch call {
			case 0:
				return approvedPlanWriteCall("internal/agent/approved_plan_numbered_summary.go"), nil
			case 1:
				return "Done.\n1. Updated foo.go\n2. Added tests\n3. Ran go test", nil
			default:
				return "unrelated follow-up", nil
			}
		},
	}
	agent := newChatRequestTestAgent(t, provider, &out)
	registerApprovedPlanWriteTool(agent)
	agent.PendingApprovedPlan = "Implementation Plan\n1. Update the target file"

	req1 := &chatRequest{input: "implement it"}
	if err := agent.executeChatRequest(context.Background(), req1); err != nil {
		t.Fatalf("first executeChatRequest() error = %v", err)
	}
	if agent.PendingApprovedPlan != "" {
		t.Fatalf("PendingApprovedPlan = %q, want empty after completion response with numbered summary", agent.PendingApprovedPlan)
	}
	if req1.approvedPlanHandoff == "" {
		t.Fatal("approvedPlanHandoff should be captured for the completion response")
	}

	req2 := &chatRequest{input: "unrelated follow-up"}
	if err := agent.executeChatRequest(context.Background(), req2); err != nil {
		t.Fatalf("second executeChatRequest() error = %v", err)
	}
	if req2.approvedPlanHandoff != "" {
		t.Fatalf("approvedPlanHandoff = %q, want empty after completion with numbered summary consumed the handoff", req2.approvedPlanHandoff)
	}
	if len(capturedHistories) != 3 {
		t.Fatalf("capturedHistories = %d, want 3", len(capturedHistories))
	}
	for _, msg := range capturedHistories[2] {
		if strings.Contains(msg.Content, "[APPROVED PLAN HANDOFF]") {
			t.Fatalf("approved plan handoff should not be reinjected after completion with numbered summary, got %#v", capturedHistories[2])
		}
	}
}

func TestHandleClearCommand_ClearsPendingApprovedPlanAndPreventsReinjection(t *testing.T) {
	disableColors(t)

	var out bytes.Buffer
	var capturedHistories [][]api.Message
	provider := &scriptedChatProvider{
		name:            "test",
		functionCalling: true,
		chatWithToolsFn: func(call int, ctx context.Context, systemPrompt string, history []api.Message, model string) (string, error) {
			snapshot := append([]api.Message(nil), history...)
			capturedHistories = append(capturedHistories, snapshot)
			return "done", nil
		},
	}
	agent := newChatRequestTestAgent(t, provider, &out)
	agent.PendingApprovedPlan = "Implementation Plan\n1. Update the target file"
	agent.History = []api.Message{{Role: "user", Content: "old task"}}
	agent.lastOutputs = []string{"old output"}
	if agent.session == nil {
		t.Fatal("agent.session = nil")
	}
	agent.session.AddMessage("user", "old task", agent.CurrentModel)
	agent.session.PendingApprovedPlan = agent.PendingApprovedPlan
	agent.persistSession()

	if !handleClearCommand(agent, nil) {
		t.Fatal("handleClearCommand() = false, want true")
	}
	if agent.PendingApprovedPlan != "" {
		t.Fatalf("PendingApprovedPlan = %q, want empty after /clear", agent.PendingApprovedPlan)
	}
	if len(agent.History) != 0 {
		t.Fatalf("len(agent.History) = %d, want 0 after /clear", len(agent.History))
	}
	if len(agent.lastOutputs) != 0 {
		t.Fatalf("len(agent.lastOutputs) = %d, want 0 after /clear", len(agent.lastOutputs))
	}
	loaded, err := agent.storage.Load(agent.session.ID)
	if err != nil {
		t.Fatalf("storage.Load() error = %v", err)
	}
	if len(loaded.ToAPIMessages()) != 0 {
		t.Fatalf("len(loaded.ToAPIMessages()) = %d, want 0 after /clear", len(loaded.ToAPIMessages()))
	}
	if loaded.PendingApprovedPlan != "" {
		t.Fatalf("loaded.PendingApprovedPlan = %q, want empty after /clear", loaded.PendingApprovedPlan)
	}

	req := &chatRequest{input: "new task"}
	if err := agent.executeChatRequest(context.Background(), req); err != nil {
		t.Fatalf("executeChatRequest() error = %v", err)
	}
	if req.approvedPlanHandoff != "" {
		t.Fatalf("approvedPlanHandoff = %q, want empty after /clear", req.approvedPlanHandoff)
	}
	for _, msg := range capturedHistories[0] {
		if strings.Contains(msg.Content, "[APPROVED PLAN HANDOFF]") {
			t.Fatalf("approved plan handoff should not be injected after /clear, got %#v", capturedHistories[0])
		}
	}
}
