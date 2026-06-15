package mcp

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/susugadx/xelyon-cli/internal/mcpapproval"
)

func TestManagerStatusSnapshot_ClassifiesServersAndSanitizesConfig(t *testing.T) {
	manager := NewManager()
	lastHealthy := time.Date(2026, 6, 15, 10, 0, 0, 0, time.UTC)
	manager.config = &Config{MCPServers: map[string]ServerConfig{
		"connected": {
			Command:               "node",
			Args:                  []string{"--token=RAW_ARG_SECRET"},
			Env:                   map[string]string{"API_TOKEN": "ENV_SECRET"},
			Approval:              "auto",
			StartupTimeoutSeconds: 30,
			ToolTimeoutSeconds:    45,
		},
		"blocked": {
			Command: "sh",
		},
		"disabled": {
			Command:  "python3",
			Disabled: true,
			Approval: "not-a-real-mode SECRET_APPROVAL",
		},
		"idle": {
			Command: "uvx",
		},
	}}
	manager.sessions["connected"] = nil
	manager.healthCheck["connected"] = lastHealthy
	manager.tools = []MCPTool{
		{ServerName: "connected", Name: "zeta", Approval: mcpapproval.ModeAuto},
		{ServerName: "connected", Name: "alpha", Approval: mcpapproval.ModeConfirm},
	}

	snapshot := manager.StatusSnapshot()
	if !snapshot.ConfigLoaded {
		t.Fatal("ConfigLoaded = false, want true")
	}
	if snapshot.ServerCount != 4 || snapshot.ConnectedServerCount != 1 || snapshot.DisabledServerCount != 1 || snapshot.BlockedServerCount != 1 || snapshot.NotConnectedServerCount != 1 {
		t.Fatalf("server counts = %#v", snapshot)
	}

	gotStates := map[string]ServerStatusState{}
	for _, server := range snapshot.Servers {
		gotStates[server.Name] = server.State
	}
	wantStates := map[string]ServerStatusState{
		"blocked":   ServerStatusBlocked,
		"connected": ServerStatusConnected,
		"disabled":  ServerStatusDisabled,
		"idle":      ServerStatusNotConnected,
	}
	for name, want := range wantStates {
		if gotStates[name] != want {
			t.Fatalf("server %s state = %q, want %q (snapshot=%#v)", name, gotStates[name], want, snapshot.Servers)
		}
	}

	if snapshot.Servers[0].Name != "blocked" || snapshot.Servers[1].Name != "connected" {
		t.Fatalf("servers should be sorted by name: %#v", snapshot.Servers)
	}
	connected := requireStatusServer(t, snapshot, "connected")
	if connected.RegisteredToolCount != 2 {
		t.Fatalf("connected.RegisteredToolCount = %d, want 2", connected.RegisteredToolCount)
	}
	if connected.StartupTimeoutSeconds != 30 || connected.ToolTimeoutSeconds != 45 {
		t.Fatalf("connected timeouts = %ds/%ds, want 30s/45s", connected.StartupTimeoutSeconds, connected.ToolTimeoutSeconds)
	}
	if !connected.LastHealthySet || !connected.LastHealthy.Equal(lastHealthy) {
		t.Fatalf("connected.LastHealthy = %v set=%v, want %v", connected.LastHealthy, connected.LastHealthySet, lastHealthy)
	}
	disabled := requireStatusServer(t, snapshot, "disabled")
	if disabled.Approval != "confirm" || disabled.ApprovalValid {
		t.Fatalf("disabled approval = %q valid=%v, want confirm invalid", disabled.Approval, disabled.ApprovalValid)
	}

	if len(snapshot.Tools) != 2 {
		t.Fatalf("Tools len = %d, want 2", len(snapshot.Tools))
	}
	if snapshot.Tools[0].ExportedName != "mcp_connected_alpha" || snapshot.Tools[1].ExportedName != "mcp_connected_zeta" {
		t.Fatalf("tools should be sorted and exported: %#v", snapshot.Tools)
	}

	rendered := fmt.Sprintf("%#v", snapshot)
	for _, secret := range []string{"RAW_ARG_SECRET", "ENV_SECRET", "SECRET_APPROVAL"} {
		if strings.Contains(rendered, secret) {
			t.Fatalf("StatusSnapshot leaked secret %q:\n%s", secret, rendered)
		}
	}
}

func TestManagerStatusSnapshot_NoConfigLoaded(t *testing.T) {
	snapshot := NewManager().StatusSnapshot()
	if snapshot.ConfigLoaded {
		t.Fatal("ConfigLoaded = true, want false")
	}
	if snapshot.ServerCount != 0 || len(snapshot.Servers) != 0 || len(snapshot.Tools) != 0 {
		t.Fatalf("empty manager snapshot = %#v, want empty counts", snapshot)
	}
}

func requireStatusServer(t *testing.T, snapshot StatusSnapshot, name string) ServerStatusSnapshot {
	t.Helper()
	for _, server := range snapshot.Servers {
		if server.Name == name {
			return server
		}
	}
	t.Fatalf("server %q not found in %#v", name, snapshot.Servers)
	return ServerStatusSnapshot{}
}
