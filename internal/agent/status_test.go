package agent

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/tools/subagent"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

func TestAgentState_Constants(t *testing.T) {
	// AgentState定数が正しく定義されていることを確認
	tests := []struct {
		state    AgentState
		expected string
	}{
		{StateRunning, "running"},
		{StateWaitingInput, "waiting_input"},
		{StateWaitingApproval, "waiting_approval"},
		{StateAborted, "aborted"},
		{StateCompleted, "completed"},
	}

	for _, tt := range tests {
		t.Run(string(tt.state), func(t *testing.T) {
			if string(tt.state) != tt.expected {
				t.Errorf("AgentState = %q, want %q", string(tt.state), tt.expected)
			}
		})
	}
}

func TestStatusHolder_SetAndGet(t *testing.T) {
	var holder statusHolder

	// 初期状態はゼロ値
	status := holder.getStatus()
	if status.State != "" {
		t.Errorf("Initial state = %q, want empty", status.State)
	}

	// ステータスを設定
	newStatus := AgentStatus{
		State:    StateRunning,
		ReasonEN: "Processing",
		ReasonJP: "処理中",
		NextEN:   "Please wait",
		NextJP:   "お待ちください",
	}
	holder.setStatus(newStatus)

	// 取得して確認
	got := holder.getStatus()
	if got.State != StateRunning {
		t.Errorf("State = %q, want %q", got.State, StateRunning)
	}
	if got.ReasonEN != "Processing" {
		t.Errorf("ReasonEN = %q, want 'Processing'", got.ReasonEN)
	}
	if got.ReasonJP != "処理中" {
		t.Errorf("ReasonJP = %q, want '処理中'", got.ReasonJP)
	}
	if got.NextEN != "Please wait" {
		t.Errorf("NextEN = %q, want 'Please wait'", got.NextEN)
	}
	if got.NextJP != "お待ちください" {
		t.Errorf("NextJP = %q, want 'お待ちください'", got.NextJP)
	}
}

func TestDefaultStatus(t *testing.T) {
	status := defaultStatus()

	if status.State != StateWaitingInput {
		t.Errorf("defaultStatus().State = %q, want %q", status.State, StateWaitingInput)
	}
	if status.ReasonEN != "Ready for input" {
		t.Errorf("defaultStatus().ReasonEN = %q, want 'Ready for input'", status.ReasonEN)
	}
	if status.ReasonJP != "入力待ち" {
		t.Errorf("defaultStatus().ReasonJP = %q, want '入力待ち'", status.ReasonJP)
	}
}

func TestStatusRef_InitializesDefault(t *testing.T) {
	var agent Agent

	status := agent.statusRef().getStatus()
	if status.State != StateWaitingInput {
		t.Errorf("statusRef().State = %q, want %q", status.State, StateWaitingInput)
	}
}

func TestAgent_SetStatus(t *testing.T) {
	provider := &mockProvider{name: "test"}
	agent := NewAgent("test-model", provider, false)

	agent.SetStatus(
		StateRunning,
		"Executing tool",
		"ツール実行中",
		"Wait for completion",
		"完了をお待ちください",
	)

	status := agent.statusRef().getStatus()
	if status.State != StateRunning {
		t.Errorf("State = %q, want %q", status.State, StateRunning)
	}
	if status.ReasonEN != "Executing tool" {
		t.Errorf("ReasonEN = %q, want 'Executing tool'", status.ReasonEN)
	}
	if status.ReasonJP != "ツール実行中" {
		t.Errorf("ReasonJP = %q, want 'ツール実行中'", status.ReasonJP)
	}
}

func TestAgentStatus_AllStates(t *testing.T) {
	provider := &mockProvider{name: "test"}
	agent := NewAgent("test-model", provider, false)

	testCases := []struct {
		state    AgentState
		reasonEN string
		reasonJP string
	}{
		{StateRunning, "Running", "実行中"},
		{StateWaitingInput, "Waiting input", "入力待ち"},
		{StateWaitingApproval, "Waiting approval", "承認待ち"},
		{StateAborted, "Aborted", "中断"},
		{StateCompleted, "Completed", "完了"},
	}

	for _, tc := range testCases {
		t.Run(string(tc.state), func(t *testing.T) {
			agent.SetStatus(tc.state, tc.reasonEN, tc.reasonJP, "", "")
			status := agent.statusRef().getStatus()
			if status.State != tc.state {
				t.Errorf("State = %q, want %q", status.State, tc.state)
			}
			if status.ReasonEN != tc.reasonEN {
				t.Errorf("ReasonEN = %q, want %q", status.ReasonEN, tc.reasonEN)
			}
		})
	}
}

func TestStatusHolder_Concurrent(t *testing.T) {
	var holder statusHolder
	done := make(chan bool)

	// 並行でsetStatusとgetStatusを呼び出し（競合状態がないことを確認）
	go func() {
		for i := 0; i < 100; i++ {
			holder.setStatus(AgentStatus{State: StateRunning, ReasonEN: "test"})
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 100; i++ {
			_ = holder.getStatus()
		}
		done <- true
	}()

	<-done
	<-done
	// パニックせずに完了すればOK
}

func TestPrintStatusFooter_UsesRuntimeOutput(t *testing.T) {
	var outA bytes.Buffer
	var outB bytes.Buffer

	agentA := &Agent{
		CurrentModel: "model-a",
		ProviderName: "openai",
		Stats:        NewSessionStats("openai", "model-a"),
		Runtime: &AgentRuntime{
			UI: ui.NewRuntime(strings.NewReader(""), &outA, &outA),
		},
	}
	agentB := &Agent{
		CurrentModel: "model-b",
		ProviderName: "openai",
		Stats:        NewSessionStats("openai", "model-b"),
		Runtime: &AgentRuntime{
			UI: ui.NewRuntime(strings.NewReader(""), &outB, &outB),
		},
	}

	agentA.PrintStatusFooter()
	agentB.PrintStatusFooter()

	if strings.Contains(outA.String(), "model-b") {
		t.Fatalf("agent A output should not contain model-b: %q", outA.String())
	}
	if !strings.Contains(outA.String(), "model-a") {
		t.Fatalf("agent A output should contain model-a: %q", outA.String())
	}
	if strings.Contains(outB.String(), "model-a") {
		t.Fatalf("agent B output should not contain model-a: %q", outB.String())
	}
	if !strings.Contains(outB.String(), "model-b") {
		t.Fatalf("agent B output should contain model-b: %q", outB.String())
	}
}

func TestPrintStatusFooter_IncludesSubAgentCost(t *testing.T) {
	var out bytes.Buffer

	// サブエージェント完了時にコスト 0.050 を返すマネージャー
	manager := subagent.NewManagerWithOptions(subagent.ManagerOptions{
		RunHeadless: func(_ context.Context, _, model string, _ api.Provider, _ *config.Config) *subagent.RunResult {
			return &subagent.RunResult{
				Status: "completed",
				Model:  model,
				Cost:   0.050,
			}
		},
		ProviderFactory: func(providerName string) (api.Provider, error) {
			return &mockProvider{name: providerName}, nil
		},
	})
	cfg := config.DefaultConfig()
	provider := &mockProvider{name: "openai"}
	id, err := manager.Spawn(context.Background(), "task", "", "gpt-5.4", "", provider, cfg)
	if err != nil {
		t.Fatalf("Spawn() error = %v", err)
	}
	_ = manager.Wait([]string{id}, 0)

	agent := &Agent{
		CurrentModel: "gpt-5.4",
		ProviderName: "openai",
		Stats:        NewSessionStats("openai", "gpt-5.4"),
		Runtime: &AgentRuntime{
			UI:              ui.NewRuntime(strings.NewReader(""), &out, &out),
			SubAgentManager: manager,
		},
	}
	agent.Stats.AccumulatedCost = 0.100

	agent.PrintStatusFooter()

	output := out.String()
	// 親 0.100 + サブ 0.050 = 0.150
	if !strings.Contains(output, "~$0.150") {
		t.Fatalf("PrintStatusFooter() should include sub-agent cost, got:\n%s", output)
	}
}

func TestPrintStatusFooter_UnknownPricingUsesNA(t *testing.T) {
	var out bytes.Buffer
	agent := &Agent{
		CurrentModel: "amazon.nova-pro-v1:0",
		ProviderName: "bedrock",
		Stats:        NewSessionStats("bedrock", "amazon.nova-pro-v1:0"),
		Runtime: &AgentRuntime{
			UI: ui.NewRuntime(strings.NewReader(""), &out, &out),
		},
	}
	agent.Stats.AddTokens(1000, 200)

	agent.PrintStatusFooter()

	if !strings.Contains(out.String(), "cost N/A") {
		t.Fatalf("PrintStatusFooter() should show cost N/A for unknown pricing, got:\n%s", out.String())
	}
}
