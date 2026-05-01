package config

import (
	"os"
	"strings"
	"testing"
)

func TestApplyEnvironmentOverrides(t *testing.T) {
	tests := []struct {
		name    string
		envVars map[string]string
		checkFn func(*testing.T, *Config)
	}{
		{
			name: "LoopThreshold",
			envVars: map[string]string{
				"XELYON_LOOP_THRESHOLD": "5",
			},
			checkFn: func(t *testing.T, cfg *Config) {
				if cfg.LoopDetection.Threshold != 5 {
					t.Errorf("LoopDetection.Threshold = %d, want 5", cfg.LoopDetection.Threshold)
				}
			},
		},
		{
			name: "Invalid LoopThreshold",
			envVars: map[string]string{
				"XELYON_LOOP_THRESHOLD": "invalid",
			},
			checkFn: func(t *testing.T, cfg *Config) {
				if cfg.LoopDetection.Threshold != 3 {
					t.Errorf("LoopDetection.Threshold should remain default (3), got %d", cfg.LoopDetection.Threshold)
				}
			},
		},
		{
			name: "Multiple env vars",
			envVars: map[string]string{
				"XELYON_DIFF_CONTEXT_LINES": "0",
			},
			checkFn: func(t *testing.T, cfg *Config) {
				if cfg.Diff.ContextLines != 0 {
					t.Errorf("Diff.ContextLines = %d, want 0", cfg.Diff.ContextLines)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for key, value := range tt.envVars {
				t.Setenv(key, value)
			}

			cfg := DefaultConfig()
			cfg.ApplyEnvironmentOverrides()

			tt.checkFn(t, cfg)
		})
	}
}

func TestApplyEnvironmentOverrides_InvalidValues_Warn(t *testing.T) {
	tests := []struct {
		name   string
		envKey string
		envVal string
	}{
		{"non-numeric loop threshold", "XELYON_LOOP_THRESHOLD", "abc"},
		{"negative loop threshold", "XELYON_LOOP_THRESHOLD", "-1"},
		{"zero loop threshold", "XELYON_LOOP_THRESHOLD", "0"},
		{"non-numeric diff lines", "XELYON_DIFF_CONTEXT_LINES", "many"},
		{"negative diff lines", "XELYON_DIFF_CONTEXT_LINES", "-1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(tt.envKey, tt.envVal)

			oldStderr := os.Stderr
			r, w, _ := os.Pipe()
			os.Stderr = w

			cfg := DefaultConfig()
			cfg.ApplyEnvironmentOverrides()

			w.Close()
			os.Stderr = oldStderr

			var buf [1024]byte
			n, _ := r.Read(buf[:])
			r.Close()
			output := string(buf[:n])

			if !strings.Contains(output, "Warning") || !strings.Contains(output, tt.envKey) {
				t.Errorf("Expected warning for %s=%q on stderr, got: %q", tt.envKey, tt.envVal, output)
			}
		})
	}
}

func TestApplyFlagOverrides(t *testing.T) {
	tests := []struct {
		name          string
		loopThreshold *int
		diffLines     *int
		checkFn       func(*testing.T, *Config)
	}{
		{
			name:          "nil pointers",
			loopThreshold: nil,
			diffLines:     nil,
			checkFn: func(t *testing.T, cfg *Config) {
				if cfg.LoopDetection.Threshold != 3 {
					t.Errorf("LoopDetection.Threshold should remain 3, got %d", cfg.LoopDetection.Threshold)
				}
			},
		},
		{
			name:          "valid values",
			loopThreshold: func() *int { v := 5; return &v }(),
			diffLines:     func() *int { v := 20; return &v }(),
			checkFn: func(t *testing.T, cfg *Config) {
				if cfg.LoopDetection.Threshold != 5 {
					t.Errorf("LoopDetection.Threshold = %d, want 5", cfg.LoopDetection.Threshold)
				}
				if cfg.Diff.ContextLines != 20 {
					t.Errorf("Diff.ContextLines = %d, want 20", cfg.Diff.ContextLines)
				}
			},
		},
		{
			name:      "diffLines = 0 is valid",
			diffLines: func() *int { v := 0; return &v }(),
			checkFn: func(t *testing.T, cfg *Config) {
				if cfg.Diff.ContextLines != 0 {
					t.Errorf("Diff.ContextLines should accept 0, got %d", cfg.Diff.ContextLines)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			cfg.ApplyFlagOverrides(tt.loopThreshold, tt.diffLines)

			tt.checkFn(t, cfg)
		})
	}
}
