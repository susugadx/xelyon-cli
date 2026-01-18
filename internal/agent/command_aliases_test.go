package agent

import (
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestResolveCommandAlias_Default(t *testing.T) {
	config.SetGlobalConfig(config.DefaultConfig())

	got := resolveCommandAlias("/h")
	if got != "/help" {
		t.Fatalf("resolveCommandAlias(/h) = %q, want /help", got)
	}

	got = resolveCommandAlias("/p")
	if got != "/paste" {
		t.Fatalf("resolveCommandAlias(/p) = %q, want /paste", got)
	}
}

func TestResolveCommandAlias_CaseInsensitive(t *testing.T) {
	config.SetGlobalConfig(config.DefaultConfig())

	got := resolveCommandAlias("/H")
	if got != "/help" {
		t.Fatalf("resolveCommandAlias(/H) = %q, want /help", got)
	}
}

func TestResolveCommandAlias_UserOverride(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.CommandAliases = map[string]string{
		"/h": "/history",
	}
	config.SetGlobalConfig(cfg)

	got := resolveCommandAlias("/h")
	if got != "/history" {
		t.Fatalf("resolveCommandAlias(/h) = %q, want /history", got)
	}
}

func TestResolveCommandAlias_UserAdditional(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.CommandAliases = map[string]string{
		"/hh": "/help",
	}
	config.SetGlobalConfig(cfg)

	got := resolveCommandAlias("/hh")
	if got != "/help" {
		t.Fatalf("resolveCommandAlias(/hh) = %q, want /help", got)
	}
}

func TestResolveCommandAlias_Chain(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.CommandAliases = map[string]string{
		"/a": "/b",
		"/b": "/help",
	}
	config.SetGlobalConfig(cfg)

	got := resolveCommandAlias("/a")
	if got != "/help" {
		t.Fatalf("resolveCommandAlias(/a) = %q, want /help", got)
	}
}

func TestResolveCommandAlias_Cycle(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.CommandAliases = map[string]string{
		"/a": "/b",
		"/b": "/a",
	}
	config.SetGlobalConfig(cfg)

	got := resolveCommandAlias("/a")
	if got != "/a" {
		t.Fatalf("resolveCommandAlias(/a) with cycle = %q, want /a", got)
	}
}

func TestResolveCommandAlias_NoAlias(t *testing.T) {
	config.SetGlobalConfig(config.DefaultConfig())

	got := resolveCommandAlias("/unknown")
	if got != "/unknown" {
		t.Fatalf("resolveCommandAlias(/unknown) = %q, want /unknown", got)
	}
}
