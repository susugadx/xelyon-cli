package subagent

import (
	"context"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestManagerSpawn_EditTaskUsesResolvedEditToolPrompt(t *testing.T) {
	tests := []struct {
		name            string
		providerName    string
		model           string
		wantContains    string
		wantNotContains string
	}{
		{
			name:            "openai uses apply_patch prompt",
			providerName:    "openai",
			model:           "gpt-5.4-mini",
			wantContains:    "apply_patch",
			wantNotContains: "str_replace",
		},
		{
			name:            "claude uses legacy prompt",
			providerName:    "claude",
			model:           "claude-haiku-4-5-20251001",
			wantContains:    "str_replace",
			wantNotContains: "apply_patch",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.DefaultConfig()
			provider := &managerTestProvider{name: tt.providerName}
			createdProvider := &managerTestProvider{name: tt.providerName}

			var gotCfg *config.Config
			manager := NewManagerWithOptions(ManagerOptions{
				RunHeadless: func(_ context.Context, _ string, _ string, _ api.Provider, cfg *config.Config) *RunResult {
					gotCfg = cfg
					return &RunResult{Status: "completed", Response: "ok"}
				},
				ProviderFactory: func(providerName string) (api.Provider, error) {
					return createdProvider, nil
				},
			})

			id, err := manager.Spawn(context.Background(), "edit files", TaskTypeEdit, tt.model, "", provider, cfg)
			if err != nil {
				t.Fatalf("Spawn() error = %v", err)
			}
			response := manager.Wait([]string{id}, 0)
			if response.Status != "completed" {
				t.Fatalf("Wait().Status = %q, want completed", response.Status)
			}
			if gotCfg == nil {
				t.Fatal("expected cloned config to be passed to runner")
			}
			if !strings.Contains(gotCfg.SubAgentPrompt, tt.wantContains) {
				t.Fatalf("SubAgentPrompt = %q, want substring %q", gotCfg.SubAgentPrompt, tt.wantContains)
			}
			if tt.wantNotContains != "" && strings.Contains(gotCfg.SubAgentPrompt, tt.wantNotContains) {
				t.Fatalf("SubAgentPrompt = %q, must not contain %q", gotCfg.SubAgentPrompt, tt.wantNotContains)
			}
		})
	}
}
