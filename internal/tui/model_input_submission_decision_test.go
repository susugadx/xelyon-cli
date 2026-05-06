package tui

import (
	"testing"

	"github.com/susugadx/xelyon-cli/internal/tui/commandrouter"
	"github.com/susugadx/xelyon-cli/internal/tui/slash"
)

func TestDecideCommandSubmission(t *testing.T) {
	tests := []struct {
		name              string
		input             string
		hasMouseSelection bool
		wantKind          commandSubmissionDecisionKind
		wantAction        commandrouter.Action
	}{
		{
			name:       "local command parse error",
			input:      `/exit "unterminated`,
			wantKind:   commandSubmissionDecisionLocalSyntaxError,
			wantAction: commandrouter.ActionQuit,
		},
		{
			name:       "unknown command parse error falls back",
			input:      `/note "unterminated`,
			wantKind:   commandSubmissionDecisionFallbackChat,
			wantAction: commandrouter.ActionDispatchAgent,
		},
		{
			name:       "local review action",
			input:      "/review",
			wantKind:   commandSubmissionDecisionLocalAction,
			wantAction: commandrouter.ActionOpenReview,
		},
		{
			name:       "bare provider picker action",
			input:      "/provider",
			wantKind:   commandSubmissionDecisionLocalAction,
			wantAction: commandrouter.ActionOpenProviderPicker,
		},
		{
			name:       "provider with args dispatches",
			input:      "/provider openai",
			wantKind:   commandSubmissionDecisionDispatchAgent,
			wantAction: commandrouter.ActionDispatchAgent,
		},
		{
			name:       "bare model picker action",
			input:      "/model",
			wantKind:   commandSubmissionDecisionLocalAction,
			wantAction: commandrouter.ActionOpenModelPicker,
		},
		{
			name:       "model with args dispatches",
			input:      "/model gpt-5.4",
			wantKind:   commandSubmissionDecisionDispatchAgent,
			wantAction: commandrouter.ActionDispatchAgent,
		},
		{
			name:       "dispatch agent command",
			input:      "/clear",
			wantKind:   commandSubmissionDecisionDispatchAgent,
			wantAction: commandrouter.ActionDispatchAgent,
		},
		{
			name:              "copy without selection dispatches",
			input:             "/copy",
			hasMouseSelection: false,
			wantKind:          commandSubmissionDecisionDispatchAgent,
			wantAction:        commandrouter.ActionDispatchAgent,
		},
		{
			name:              "copy with selection local",
			input:             "/copy",
			hasMouseSelection: true,
			wantKind:          commandSubmissionDecisionLocalAction,
			wantAction:        commandrouter.ActionCopyMouseSelection,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			command := slash.NewCommand(tt.input, tt.input, nil)
			decision := decideCommandSubmission(command, tt.hasMouseSelection)
			if decision.kind != tt.wantKind {
				t.Fatalf("decision.kind = %v, want %v", decision.kind, tt.wantKind)
			}
			if decision.action != tt.wantAction {
				t.Fatalf("decision.action = %v, want %v", decision.action, tt.wantAction)
			}
		})
	}
}
