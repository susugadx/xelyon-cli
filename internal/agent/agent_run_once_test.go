package agent

import (
	"bytes"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/stdio"
)

func TestRunOnceWithConfig_PrintsHeaderModeAndAuditBanner(t *testing.T) {
	disableColors(t)
	withTempWorkdir(t)
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XELYON_AUDIT_LOG", "1")

	var out bytes.Buffer
	stdio.SetDefaults(strings.NewReader(""), &out, &out)
	t.Cleanup(func() { stdio.SetDefaults(nil, nil, nil) })

	provider := &scriptedChatProvider{name: "openai", functionCalling: true}
	if err := RunOnceWithConfig("hello", "test-model", provider, newProjectMapDisabledConfig(), true, false); err != nil {
		t.Fatalf("RunOnceWithConfig() error = %v", err)
	}
	if provider.callCount != 1 {
		t.Fatalf("provider.callCount = %d, want 1", provider.callCount)
	}

	got := out.String()
	for _, fragment := range []string{
		"Provider: openai | Model: test-model",
		"Mode: Auto-approve",
		"Audit logging enabled",
	} {
		if !strings.Contains(got, fragment) {
			t.Fatalf("RunOnceWithConfig() output missing %q:\n%s", fragment, got)
		}
	}
}

func TestRunOnceWithConfig_QuietSkipsStatusOutput(t *testing.T) {
	disableColors(t)
	withTempWorkdir(t)
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XELYON_AUDIT_LOG", "1")

	var out bytes.Buffer
	stdio.SetDefaults(strings.NewReader(""), &out, &out)
	t.Cleanup(func() { stdio.SetDefaults(nil, nil, nil) })

	provider := &scriptedChatProvider{name: "openai", functionCalling: true}
	if err := RunOnceWithConfig("hello", "test-model", provider, newProjectMapDisabledConfig(), true, true); err != nil {
		t.Fatalf("RunOnceWithConfig() error = %v", err)
	}
	if provider.callCount != 1 {
		t.Fatalf("provider.callCount = %d, want 1", provider.callCount)
	}

	got := out.String()
	for _, fragment := range []string{
		"Provider: openai | Model: test-model",
		"Mode: Auto-approve",
		"Audit logging enabled",
	} {
		if strings.Contains(got, fragment) {
			t.Fatalf("RunOnceWithConfig() quiet output should not contain %q:\n%s", fragment, got)
		}
	}
}
