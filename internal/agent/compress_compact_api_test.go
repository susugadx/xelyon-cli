package agent

import (
	"context"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
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
