package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/susugadx/xelyon-cli/internal/api"
	openaisubscription "github.com/susugadx/xelyon-cli/internal/api/providers/openai_subscription"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

func TestCompressWithCompactAPI_UsesCompressionModel(t *testing.T) {
	provider := &compressionTestProvider{name: "openai", supportsCompact: true}
	cfg := config.DefaultConfig()
	cfg.Compression.Model = ""

	agent, _ := newCompressionTestAgent(t, provider, "gpt-5.4", cfg)
	agent.History = []api.Message{{Role: "user", Content: "hello"}}

	if err := agent.CompressWithCompactAPI(context.Background()); err != nil {
		t.Fatalf("CompressWithCompactAPI() error = %v", err)
	}
	if provider.capturedCompactModel != "gpt-5.4-mini" {
		t.Fatalf("CompressWithCompactAPI() model = %q, want %q", provider.capturedCompactModel, "gpt-5.4-mini")
	}
}

func TestCompressWithCompactAPI_SuppressesAssistantUpdates(t *testing.T) {
	provider := &compressionTestProvider{name: "openai", supportsCompact: true}
	cfg := config.DefaultConfig()
	cfg.Output.AssistantUpdates = api.AssistantUpdatesVerbose

	agent, _ := newCompressionTestAgent(t, provider, "gpt-5.4", cfg)
	agent.History = []api.Message{{Role: "user", Content: "hello"}}

	if err := agent.CompressWithCompactAPI(context.Background()); err != nil {
		t.Fatalf("CompressWithCompactAPI() error = %v", err)
	}
	if provider.capturedCompactUpdateMode != api.AssistantUpdatesOff {
		t.Fatalf("compact request assistant update mode = %q, want %q", provider.capturedCompactUpdateMode, api.AssistantUpdatesOff)
	}
}

func TestCompressWithCompactAPI_UsesOpenAISubscriptionCompactRuntime(t *testing.T) {
	t.Setenv("XELYON_OPENAI_SUBSCRIPTION_AUTH_DIR", t.TempDir())
	t.Setenv("OPENAI_API_KEY", "platform-key-must-not-be-used")

	var raw map[string]any
	var authorization string
	var accountID string
	var originator string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authorization = r.Header.Get("Authorization")
		accountID = r.Header.Get("ChatGPT-Account-Id")
		originator = r.Header.Get("originator")
		if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
			t.Fatalf("decode compact request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]any{
			"model": "gpt-5.4-mini",
			"output": []map[string]any{{
				"type": "compacted",
				"data": "subscription-compact-data",
			}},
			"usage": map[string]any{
				"input_tokens":  10,
				"output_tokens": 4,
				"total_tokens":  14,
			},
		}); err != nil {
			t.Fatalf("encode compact response: %v", err)
		}
	}))
	defer server.Close()
	t.Setenv("XELYON_OPENAI_SUBSCRIPTION_COMPACT_ENDPOINT", server.URL)
	if err := openaisubscription.SaveSubscriptionCredential(openaisubscription.DefaultSubscriptionAuthConfig(), openaisubscription.SubscriptionCredential{
		AccessToken:  "oauth-access-token",
		RefreshToken: "oauth-refresh-token",
		AccountID:    "acct_1234abcd",
		ExpiresAt:    time.Now().Add(time.Hour),
	}); err != nil {
		t.Fatalf("SaveSubscriptionCredential() error = %v", err)
	}

	provider := openaisubscription.NewSubscription()
	provider.HTTPClient = server.Client()
	cfg := config.DefaultConfig()
	cfg.Compression.Model = ""
	var out bytes.Buffer
	runtime := NewAgentRuntimeWithConfig(cfg)
	runtime.UI = ui.NewRuntime(strings.NewReader(""), &out, &out)
	agent := NewAgentWithRuntime("gpt-5.5", provider, false, runtime)
	t.Cleanup(agent.Cleanup)
	agent.History = []api.Message{{Role: "user", Content: "please compact this"}}

	if err := agent.CompressWithCompactAPI(context.Background()); err != nil {
		t.Fatalf("CompressWithCompactAPI() error = %v\noutput:\n%s", err, out.String())
	}

	if authorization != "Bearer oauth-access-token" || strings.Contains(authorization, "platform-key") {
		t.Fatalf("Authorization = %q, want OAuth bearer and no OPENAI_API_KEY", authorization)
	}
	if accountID != "acct_1234abcd" {
		t.Fatalf("ChatGPT-Account-Id = %q, want account id", accountID)
	}
	if originator != "xelyon" {
		t.Fatalf("originator = %q, want xelyon", originator)
	}
	if raw["model"] != "gpt-5.4-mini" {
		t.Fatalf("compact model = %#v, want subscription compression model", raw["model"])
	}
	if len(agent.History) != 0 {
		t.Fatalf("History length = %d, want cleared after compact", len(agent.History))
	}
	if len(agent.compactedItems) != 1 || agent.compactedItems[0].Data != "subscription-compact-data" {
		t.Fatalf("compactedItems = %#v, want subscription compact output", agent.compactedItems)
	}
	compacted := api.CompactedInputItemsFromContext(agent.requestContext(context.Background()))
	if len(compacted) != 1 || compacted[0].Data != "subscription-compact-data" {
		t.Fatalf("request compacted input = %#v, want compacted state in request context", compacted)
	}
}
