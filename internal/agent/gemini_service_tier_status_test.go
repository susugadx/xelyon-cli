package agent

import (
	"bytes"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/commandcatalog"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/uiruntime"
)

func TestHandleStatusCommandForSurface_ShowsGeminiServiceTierPolicy(t *testing.T) {
	var out bytes.Buffer
	cfg := config.DefaultConfig()
	cfg.Gemini.ServiceTier = config.GeminiServiceTierPriority
	runtime := NewAgentRuntimeWithConfig(cfg)
	runtime.UI = uiruntime.NewRuntime(strings.NewReader(""), &out, &out)
	agent := NewAgentWithRuntime("gemini-3.5-flash", &mockProvider{name: "gemini"}, false, runtime)
	t.Cleanup(agent.Cleanup)

	if !handleStatusCommandForSurface(agent, commandcatalog.CommandSurfaceTUI) {
		t.Fatal("handleStatusCommandForSurface() = false, want true")
	}

	requireGeminiServiceTierStatusFragments(t, out.String(),
		"Service Tier",
		"configured=priority",
		"request_body=priority",
		"pricing_family=gemini_priority",
	)
}

func TestBuildLastRequestTable_ShowsGeminiServiceTierBillingPolicy(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Gemini.ServiceTier = config.GeminiServiceTierPriority
	usage := &api.Usage{
		InputTokens:        17,
		OutputTokens:       5,
		BillingServiceTier: config.GeminiServiceTierStandard,
	}

	table := buildLastRequestTable(cfg, "gemini", "gemini-3.1-pro-preview-customtools", usage, nil)
	if table == nil {
		t.Fatal("buildLastRequestTable() = nil, want table")
	}

	requireGeminiServiceTierStatusFragments(t, table.RenderCompact(),
		"Service Tier",
		"configured=priority",
		"request_body=priority",
		"pricing_family=gemini_priority",
		"billing=standard",
		"billing_pricing_family=gemini",
	)
}

func requireGeminiServiceTierStatusFragments(t *testing.T, output string, fragments ...string) {
	t.Helper()
	for _, fragment := range fragments {
		if !strings.Contains(output, fragment) {
			t.Fatalf("status output missing %q:\n%s", fragment, output)
		}
	}
}
