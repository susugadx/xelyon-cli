package agent

import (
	"context"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/history"
)

func TestRequestContext_IncludesCurrentSessionPromptCacheScope(t *testing.T) {
	agent := &Agent{
		agentConversationState: agentConversationState{
			session: &history.Session{ID: " session-1 "},
		},
	}

	got, ok := api.PromptCacheScopeFromContext(agent.requestContext(context.Background()))
	if !ok {
		t.Fatal("PromptCacheScopeFromContext() ok = false, want true")
	}
	if got.SessionID != "session-1" || got.TaskID != "" {
		t.Fatalf("scope = %+v, want current session ID only", got)
	}
}

func TestRequestContext_UsesLoadedSessionPromptCacheScope(t *testing.T) {
	agent := &Agent{
		agentConversationState: agentConversationState{
			session: &history.Session{ID: "old-session"},
		},
	}

	agent.applyLoadedSession(&history.Session{ID: "restored-session"})

	got, ok := api.PromptCacheScopeFromContext(agent.requestContext(context.Background()))
	if !ok {
		t.Fatal("PromptCacheScopeFromContext() ok = false, want true")
	}
	if got.SessionID != "restored-session" {
		t.Fatalf("SessionID = %q, want restored-session", got.SessionID)
	}
}

func TestRequestContext_OmitsPromptCacheScopeWithoutSessionID(t *testing.T) {
	tests := []struct {
		name  string
		agent *Agent
	}{
		{name: "nil session", agent: &Agent{}},
		{name: "empty session ID", agent: &Agent{agentConversationState: agentConversationState{session: &history.Session{ID: "   "}}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := api.PromptCacheScopeFromContext(tt.agent.requestContext(context.Background()))
			if ok {
				t.Fatalf("PromptCacheScopeFromContext() = %+v, true; want empty false", got)
			}
		})
	}
}
