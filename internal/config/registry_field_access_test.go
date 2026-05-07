package config

import "testing"

func TestGetFieldValue(t *testing.T) {
	cfg := DefaultConfig()

	tests := []struct {
		name     string
		path     string
		wantType string
	}{
		{"default_provider", "default_provider", "string"},
		{"default_model", "default_model", "string"},
		{"compression.enabled", "compression.enabled", "bool"},
		{"compression.keep_recent", "compression.keep_recent", "int"},
		{"thinking.enabled", "thinking.enabled", "bool"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, err := GetFieldValue(cfg, tt.path)
			if err != nil {
				t.Errorf("GetFieldValue(%s) error = %v", tt.path, err)
				return
			}
			if val == nil {
				t.Errorf("GetFieldValue(%s) returned nil", tt.path)
			}
		})
	}
}

func TestSetFieldValue(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		value   interface{}
		wantErr bool
	}{
		{"set string", "default_model", "new-model", false},
		{"set bool", "thinking.enabled", true, false},
		{"set int", "compression.keep_recent", 5, false},
		{"invalid path", "nonexistent.field", "value", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			err := SetFieldValue(cfg, tt.path, tt.value)

			if (err != nil) != tt.wantErr {
				t.Errorf("SetFieldValue(%s) error = %v, wantErr %v", tt.path, err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				gotVal, _ := GetFieldValue(cfg, tt.path)
				if gotVal != tt.value {
					t.Errorf("SetFieldValue(%s) value = %v, want %v", tt.path, gotVal, tt.value)
				}
			}
		})
	}
}

func TestGetFieldValue_CommandAliasesMapKeyPath(t *testing.T) {
	cfg := DefaultConfig()
	cfg.CommandAliases = map[string]string{}
	cfg.CommandAliases["custom"] = "config"

	val, err := GetFieldValue(cfg, "command_aliases.custom")
	if err != nil {
		t.Fatalf("GetFieldValue(command_aliases.custom) error = %v", err)
	}

	got, ok := val.(string)
	if !ok {
		t.Fatalf("GetFieldValue(command_aliases.custom) returned %T, want string", val)
	}
	if got != "config" {
		t.Fatalf("GetFieldValue(command_aliases.custom) = %q, want %q", got, "config")
	}
}

func TestSetFieldValue_CommandAliasesMapKeyPath(t *testing.T) {
	cfg := DefaultConfig()

	if err := SetFieldValue(cfg, "command_aliases.alias-c", "config"); err != nil {
		t.Fatalf("SetFieldValue(command_aliases.alias-c) error = %v", err)
	}

	if got := cfg.CommandAliases["alias-c"]; got != "config" {
		t.Fatalf("cfg.CommandAliases[alias-c] = %q, want %q", got, "config")
	}
}

func TestSetFieldValue_MapKeyPathInitializesNilMap(t *testing.T) {
	cfg := DefaultConfig()
	cfg.CommandAliases = nil

	if err := SetFieldValue(cfg, "command_aliases.alias-c", "config"); err != nil {
		t.Fatalf("SetFieldValue(command_aliases.alias-c) error = %v", err)
	}

	if got := cfg.CommandAliases["alias-c"]; got != "config" {
		t.Fatalf("cfg.CommandAliases[alias-c] = %q, want %q", got, "config")
	}
}

func TestSetFieldValue_IntFieldAcceptsConvertibleAliasType(t *testing.T) {
	cfg := DefaultConfig()

	if err := SetFieldValue(cfg, "compression.keep_recent", int8(7)); err != nil {
		t.Fatalf("SetFieldValue(compression.keep_recent, int8) error = %v", err)
	}

	gotVal, err := GetFieldValue(cfg, "compression.keep_recent")
	if err != nil {
		t.Fatalf("GetFieldValue(compression.keep_recent) error = %v", err)
	}
	got, ok := gotVal.(int)
	if !ok {
		t.Fatalf("GetFieldValue(compression.keep_recent) returned %T, want int", gotVal)
	}
	if got != 7 {
		t.Fatalf("GetFieldValue(compression.keep_recent) = %d, want %d", got, 7)
	}
}

func TestSetFieldValue_IntFieldRejectsTypeMismatchWithoutMutation(t *testing.T) {
	cfg := DefaultConfig()
	before := cfg.Compression.KeepRecent

	if err := SetFieldValue(cfg, "compression.keep_recent", "7"); err == nil {
		t.Fatal("SetFieldValue(compression.keep_recent, string) expected error, got nil")
	}

	if got := cfg.Compression.KeepRecent; got != before {
		t.Fatalf("cfg.Compression.KeepRecent = %d, want unchanged %d", got, before)
	}
}
