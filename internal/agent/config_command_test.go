package agent

import (
	"testing"
)

func TestConfigCommandRecognition(t *testing.T) {
	// splitCommand のテスト
	parts := splitCommand("/config")
	if len(parts) != 1 {
		t.Errorf("Expected 1 part, got %d", len(parts))
	}
	if parts[0] != "/config" {
		t.Errorf("Expected '/config', got '%s'", parts[0])
	}

	// /config show のテスト
	parts = splitCommand("/config show")
	if len(parts) != 2 {
		t.Errorf("Expected 2 parts, got %d", len(parts))
	}
	if parts[0] != "/config" {
		t.Errorf("Expected '/config', got '%s'", parts[0])
	}
	if parts[1] != "show" {
		t.Errorf("Expected 'show', got '%s'", parts[1])
	}
}

func TestResolveCommandAlias(t *testing.T) {
	// /config がそのまま返されるか
	resolved := resolveCommandAlias("/config")
	if resolved != "/config" {
		t.Errorf("Expected '/config', got '%s'", resolved)
	}
}

func TestConfigCommandSwitch(t *testing.T) {
	// handleSpecialCommand のswitch内で/configが正しくマッチするか確認
	parts := splitCommand("/config")
	cmd := resolveCommandAlias(parts[0])

	// switch 文の条件と同じ
	switch cmd {
	case "/config":
		// OK
	default:
		t.Errorf("'/config' was not matched in switch, got cmd='%s'", cmd)
	}
}
