package agent

import (
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestResolveCommandAlias_Default(t *testing.T) {
	got := resolveCommandAliasWithConfig("/h", config.DefaultConfig())
	if got != "/help" {
		t.Fatalf("resolveCommandAlias(/h) = %q, want /help", got)
	}

	got = resolveCommandAliasWithConfig("/p", config.DefaultConfig())
	if got != "/p" {
		t.Fatalf("resolveCommandAlias(/p) = %q, want /p", got)
	}
}

func TestResolveCommandAlias_CaseInsensitive(t *testing.T) {
	got := resolveCommandAliasWithConfig("/H", config.DefaultConfig())
	if got != "/help" {
		t.Fatalf("resolveCommandAlias(/H) = %q, want /help", got)
	}
}

func TestResolveCommandAlias_UserOverride(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.CommandAliases = map[string]string{
		"/h": "/history",
	}

	got := resolveCommandAliasWithConfig("/h", cfg)
	if got != "/history" {
		t.Fatalf("resolveCommandAlias(/h) = %q, want /history", got)
	}
}

func TestResolveCommandAlias_UserAdditional(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.CommandAliases = map[string]string{
		"/hh": "/help",
	}

	got := resolveCommandAliasWithConfig("/hh", cfg)
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

	got := resolveCommandAliasWithConfig("/a", cfg)
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

	got := resolveCommandAliasWithConfig("/a", cfg)
	if got != "/a" {
		t.Fatalf("resolveCommandAlias(/a) with cycle = %q, want /a", got)
	}
}

func TestResolveCommandAlias_NoAlias(t *testing.T) {
	got := resolveCommandAliasWithConfig("/unknown", config.DefaultConfig())
	if got != "/unknown" {
		t.Fatalf("resolveCommandAlias(/unknown) = %q, want /unknown", got)
	}
}
