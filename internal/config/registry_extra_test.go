package config

import (
	"testing"
)

func TestGetStringMapValue(t *testing.T) {
	t.Run("valid map field", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.CommandAliases = map[string]string{
			"c": "config",
			"u": "use",
		}

		got, err := GetStringMapValue(cfg, "command_aliases")
		if err != nil {
			t.Fatalf("GetStringMapValue() error = %v", err)
		}
		if got == nil {
			t.Fatal("GetStringMapValue() returned nil")
		}
		if got["c"] != "config" {
			t.Errorf("got[\"c\"] = %q, want %q", got["c"], "config")
		}
		if got["u"] != "use" {
			t.Errorf("got[\"u\"] = %q, want %q", got["u"], "use")
		}
	})

	t.Run("non-map field", func(t *testing.T) {
		cfg := DefaultConfig()

		_, err := GetStringMapValue(cfg, "default_provider")
		if err == nil {
			t.Error("GetStringMapValue() expected error for non-map field, got nil")
		}
	})

	t.Run("invalid path", func(t *testing.T) {
		cfg := DefaultConfig()

		_, err := GetStringMapValue(cfg, "nonexistent.field")
		if err == nil {
			t.Error("GetStringMapValue() expected error for invalid path, got nil")
		}
	})
}

func TestSetStringMapValue(t *testing.T) {
	t.Run("set map field", func(t *testing.T) {
		cfg := DefaultConfig()
		newAliases := map[string]string{
			"h": "help",
			"q": "quit",
		}

		err := SetStringMapValue(cfg, "command_aliases", newAliases)
		if err != nil {
			t.Fatalf("SetStringMapValue() error = %v", err)
		}

		// 設定された値を確認
		got, err := GetStringMapValue(cfg, "command_aliases")
		if err != nil {
			t.Fatalf("GetStringMapValue() error = %v", err)
		}
		if got["h"] != "help" {
			t.Errorf("got[\"h\"] = %q, want %q", got["h"], "help")
		}
		if got["q"] != "quit" {
			t.Errorf("got[\"q\"] = %q, want %q", got["q"], "quit")
		}
	})
}

func TestGetStringSliceValue(t *testing.T) {
	t.Run("valid slice field", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.Bash.SafeCommands = []string{"ls", "cat", "echo"}

		got, err := GetStringSliceValue(cfg, "bash.safe_commands")
		if err != nil {
			t.Fatalf("GetStringSliceValue() error = %v", err)
		}
		if len(got) != 3 {
			t.Fatalf("GetStringSliceValue() returned %d items, want 3", len(got))
		}
		if got[0] != "ls" {
			t.Errorf("got[0] = %q, want %q", got[0], "ls")
		}
	})

	t.Run("empty slice", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.Bash.SafeCommands = []string{}

		got, err := GetStringSliceValue(cfg, "bash.safe_commands")
		if err != nil {
			t.Fatalf("GetStringSliceValue() error = %v", err)
		}
		if len(got) != 0 {
			t.Errorf("GetStringSliceValue() returned %d items, want 0", len(got))
		}
	})

	t.Run("non-slice field", func(t *testing.T) {
		cfg := DefaultConfig()

		_, err := GetStringSliceValue(cfg, "default_provider")
		if err == nil {
			t.Error("GetStringSliceValue() expected error for non-slice field, got nil")
		}
	})

	t.Run("invalid path", func(t *testing.T) {
		cfg := DefaultConfig()

		_, err := GetStringSliceValue(cfg, "nonexistent.field")
		if err == nil {
			t.Error("GetStringSliceValue() expected error for invalid path, got nil")
		}
	})
}

func TestSetStringSliceValue(t *testing.T) {
	t.Run("set slice field", func(t *testing.T) {
		cfg := DefaultConfig()
		newCommands := []string{"git", "docker", "make"}

		err := SetStringSliceValue(cfg, "bash.safe_commands", newCommands)
		if err != nil {
			t.Fatalf("SetStringSliceValue() error = %v", err)
		}

		// 設定された値を確認
		got, err := GetStringSliceValue(cfg, "bash.safe_commands")
		if err != nil {
			t.Fatalf("GetStringSliceValue() error = %v", err)
		}
		if len(got) != 3 {
			t.Fatalf("got %d items, want 3", len(got))
		}
		if got[0] != "git" || got[1] != "docker" || got[2] != "make" {
			t.Errorf("got = %v, want [git docker make]", got)
		}
	})
}

func TestRegistryEntry_String(t *testing.T) {
	tests := []struct {
		name      string
		fieldType ConfigFieldType
		want      string
	}{
		{"bool", FieldTypeBool, "bool"},
		{"int", FieldTypeInt, "int"},
		{"float", FieldTypeFloat, "float"},
		{"string", FieldTypeString, "string"},
		{"select", FieldTypeSelect, "select"},
		{"string slice", FieldTypeStringSlice, "[]string"},
		{"string map", FieldTypeStringMap, "map[string]string"},
		{"struct map", FieldTypeStructMap, "map[string]struct"},
		{"unknown", ConfigFieldType(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.fieldType.String()
			if got != tt.want {
				t.Errorf("ConfigFieldType(%d).String() = %q, want %q", tt.fieldType, got, tt.want)
			}
		})
	}
}
