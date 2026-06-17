package authcli

import (
	"bytes"
	"strings"
	"testing"

	openaisubscription "github.com/susugadx/xelyon-cli/internal/api/providers/openai_subscription"
)

func TestRenderOpenAISubscriptionAuthStatusRedactsEndpointAndMessage(t *testing.T) {
	status := openaisubscription.SubscriptionAuthStatus{
		State:           openaisubscription.SubscriptionAuthStateLoginRequired,
		TokenState:      "missing",
		AuthFilePath:    "/tmp/auth.json",
		Permission:      "0600",
		Endpoint:        "https://example.test/backend-api/codex/responses?token=secret",
		Originator:      "xelyon",
		Message:         "refresh_token=secret",
		Suggestion:      "Run login",
		AccountIDMasked: "acct_***",
	}

	var out bytes.Buffer
	RenderOpenAISubscriptionAuthStatus(&out, status)
	rendered := out.String()
	if strings.Contains(rendered, "token=secret") || strings.Contains(rendered, "refresh_token=secret") {
		t.Fatalf("rendered auth status leaked secret:\n%s", rendered)
	}
	for _, want := range []string{"OpenAI Subscription auth", "Status: login_required", "Run login"} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("rendered auth status missing %q:\n%s", want, rendered)
		}
	}
}
