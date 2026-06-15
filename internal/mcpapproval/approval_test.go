package mcpapproval

import "testing"

func TestNormalize(t *testing.T) {
	tests := []struct {
		name      string
		raw       string
		wantMode  Mode
		wantValid bool
	}{
		{name: "missing defaults to confirm", raw: "", wantMode: ModeConfirm, wantValid: true},
		{name: "confirm", raw: "confirm", wantMode: ModeConfirm, wantValid: true},
		{name: "auto", raw: "auto", wantMode: ModeAuto, wantValid: true},
		{name: "deny", raw: "deny", wantMode: ModeDeny, wantValid: true},
		{name: "trimmed", raw: " confirm ", wantMode: ModeConfirm, wantValid: true},
		{name: "case sensitive", raw: "Confirm", wantMode: ModeConfirm, wantValid: false},
		{name: "invalid", raw: "prompt", wantMode: ModeConfirm, wantValid: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotMode, gotValid := Normalize(tt.raw)
			if gotMode != tt.wantMode || gotValid != tt.wantValid {
				t.Fatalf("Normalize(%q) = (%q, %v), want (%q, %v)", tt.raw, gotMode, gotValid, tt.wantMode, tt.wantValid)
			}
		})
	}
}

func TestEffective(t *testing.T) {
	tests := []struct {
		name string
		mode Mode
		want Mode
	}{
		{name: "zero value defaults to confirm", mode: "", want: ModeConfirm},
		{name: "valid auto", mode: ModeAuto, want: ModeAuto},
		{name: "invalid defaults to confirm", mode: Mode("invalid"), want: ModeConfirm},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Effective(tt.mode); got != tt.want {
				t.Fatalf("Effective(%q) = %q, want %q", tt.mode, got, tt.want)
			}
		})
	}
}
