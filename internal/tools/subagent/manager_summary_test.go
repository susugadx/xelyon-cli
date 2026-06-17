package subagent

import (
	"context"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
)

// TestManagerGetSummary は集計結果に completed / running / コストが反映されることを確認します。
func TestManagerGetSummary(t *testing.T) {
	cfg := config.DefaultConfig()
	provider := &managerTestProvider{name: "openai"}
	release := make(chan struct{})

	manager := NewManagerWithOptions(ManagerOptions{
		RunHeadless: func(_ context.Context, message, model string, _ api.Provider, _ *config.Config) *RunResult {
			if message == "running task" {
				<-release
				return &RunResult{Status: "completed", Model: model}
			}
			return &RunResult{
				Status:             "completed",
				Model:              model,
				Response:           "done",
				InputTokens:        120,
				CachedTokens:       40,
				OutputTokens:       30,
				ThinkingTokens:     10,
				Cost:               0.0123,
				PricingUnavailable: true,
				ToolExecutions:     2,
				DurationMs:         250,
			}
		},
		ProviderFactory: func(providerName string) (api.Provider, error) {
			return &managerTestProvider{name: providerName}, nil
		},
	})

	completedID, err := manager.Spawn(context.Background(), "completed task", TaskTypeEdit, "", "", provider, cfg)
	if err != nil {
		t.Fatalf("Spawn(completed task) error = %v", err)
	}
	if response := manager.Wait([]string{completedID}, 0); response.Status != "completed" {
		t.Fatalf("Wait(completed task).Status = %q, want completed", response.Status)
	}

	runningID, err := manager.Spawn(context.Background(), "running task", TaskTypeVerify, "", "", provider, cfg)
	if err != nil {
		t.Fatalf("Spawn(running task) error = %v", err)
	}

	summary := manager.GetSummary()
	if summary.TotalSpawned != 2 {
		t.Fatalf("summary.TotalSpawned = %d, want 2", summary.TotalSpawned)
	}
	if summary.TotalCompleted != 1 {
		t.Fatalf("summary.TotalCompleted = %d, want 1", summary.TotalCompleted)
	}
	if summary.TotalRunning != 1 {
		t.Fatalf("summary.TotalRunning = %d, want 1", summary.TotalRunning)
	}
	if summary.TotalInput != 120 {
		t.Fatalf("summary.TotalInput = %d, want 120", summary.TotalInput)
	}
	if summary.TotalCached != 40 {
		t.Fatalf("summary.TotalCached = %d, want 40", summary.TotalCached)
	}
	if summary.TotalOutput != 30 {
		t.Fatalf("summary.TotalOutput = %d, want 30", summary.TotalOutput)
	}
	if summary.TotalThinking != 10 {
		t.Fatalf("summary.TotalThinking = %d, want 10", summary.TotalThinking)
	}
	if summary.TotalCost != 0.0123 {
		t.Fatalf("summary.TotalCost = %f, want 0.0123", summary.TotalCost)
	}
	if !summary.PricingUnavailable {
		t.Fatal("summary.PricingUnavailable = false, want true")
	}
	if summary.TotalTools != 2 {
		t.Fatalf("summary.TotalTools = %d, want 2", summary.TotalTools)
	}
	if len(summary.Agents) != 2 {
		t.Fatalf("len(summary.Agents) = %d, want 2", len(summary.Agents))
	}
	if summary.Agents[0].ID != completedID {
		t.Fatalf("summary.Agents[0].ID = %q, want %q", summary.Agents[0].ID, completedID)
	}
	if summary.Agents[0].TaskType != TaskTypeEdit {
		t.Fatalf("summary.Agents[0].TaskType = %q, want %q", summary.Agents[0].TaskType, TaskTypeEdit)
	}
	if !summary.Agents[0].PricingUnavailable {
		t.Fatal("summary.Agents[0].PricingUnavailable = false, want true")
	}
	if summary.Agents[1].ID != runningID {
		t.Fatalf("summary.Agents[1].ID = %q, want %q", summary.Agents[1].ID, runningID)
	}
	if summary.Agents[1].Status != "running" {
		t.Fatalf("summary.Agents[1].Status = %q, want running", summary.Agents[1].Status)
	}
	if summary.Agents[1].Model != "gpt-5.4-mini" {
		t.Fatalf("summary.Agents[1].Model = %q, want gpt-5.4-mini", summary.Agents[1].Model)
	}
	if summary.Agents[1].TaskType != TaskTypeVerify {
		t.Fatalf("summary.Agents[1].TaskType = %q, want %q", summary.Agents[1].TaskType, TaskTypeVerify)
	}

	close(release)
	_ = manager.Wait([]string{runningID}, 0)
}

// TestManagerGetSummary_ErrorMessage はエラー詳細がサマリーへ反映されることを確認します。
func TestManagerGetSummary_ErrorMessage(t *testing.T) {
	cfg := config.DefaultConfig()
	provider := &managerTestProvider{name: "openai"}
	manager := NewManagerWithOptions(ManagerOptions{
		RunHeadless: func(_ context.Context, _ string, model string, _ api.Provider, _ *config.Config) *RunResult {
			return &RunResult{
				Status:       "error",
				Model:        model,
				ErrorMessage: "provider timeout",
			}
		},
		ProviderFactory: func(providerName string) (api.Provider, error) {
			return &managerTestProvider{name: providerName}, nil
		},
	})

	id, err := manager.Spawn(context.Background(), "failing task", "", "", "", provider, cfg)
	if err != nil {
		t.Fatalf("Spawn() error = %v", err)
	}
	_ = manager.Wait([]string{id}, 0)

	summary := manager.GetSummary()
	if len(summary.Agents) != 1 {
		t.Fatalf("len(summary.Agents) = %d, want 1", len(summary.Agents))
	}
	if summary.Agents[0].ErrorMessage != "provider timeout" {
		t.Fatalf("summary.Agents[0].ErrorMessage = %q, want %q", summary.Agents[0].ErrorMessage, "provider timeout")
	}
}
