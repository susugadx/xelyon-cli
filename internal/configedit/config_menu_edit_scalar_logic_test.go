package configedit

import (
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestParseBoolInput(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		current bool
		value   bool
		changed bool
		valid   bool
	}{
		{name: "keep on empty", input: "", current: true, value: true, changed: false, valid: true},
		{name: "yes", input: "yes", current: false, value: true, changed: true, valid: true},
		{name: "off", input: "off", current: true, value: false, changed: true, valid: true},
		{name: "invalid", input: "maybe", current: true, value: true, changed: false, valid: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value, changed, valid := parseBoolInput(tt.input, tt.current)
			if value != tt.value || changed != tt.changed || valid != tt.valid {
				t.Fatalf("parseBoolInput() = (%v,%v,%v), want (%v,%v,%v)", value, changed, valid, tt.value, tt.changed, tt.valid)
			}
		})
	}
}

func TestExtractIntCurrentValue(t *testing.T) {
	tests := []struct {
		name  string
		value interface{}
		want  int
	}{
		{name: "int", value: 3, want: 3},
		{name: "int64", value: int64(4), want: 4},
		{name: "float64", value: 5.9, want: 5},
		{name: "other", value: "x", want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := extractIntCurrentValue(tt.value); got != tt.want {
				t.Fatalf("extractIntCurrentValue() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestParseIntInput(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		current int
		value   int
		changed bool
		valid   bool
	}{
		{name: "keep", input: "", current: 3, value: 3, changed: false, valid: true},
		{name: "valid", input: "7", current: 3, value: 7, changed: true, valid: true},
		{name: "invalid", input: "abc", current: 3, value: 3, changed: false, valid: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value, changed, valid := parseIntInput(tt.input, tt.current)
			if value != tt.value || changed != tt.changed || valid != tt.valid {
				t.Fatalf("parseIntInput() = (%v,%v,%v), want (%v,%v,%v)", value, changed, valid, tt.value, tt.changed, tt.valid)
			}
		})
	}
}

func TestExtractFloatCurrentValue(t *testing.T) {
	tests := []struct {
		name  string
		value interface{}
		want  float64
	}{
		{name: "float64", value: 0.25, want: 0.25},
		{name: "float32", value: float32(0.5), want: 0.5},
		{name: "int", value: 2, want: 2},
		{name: "other", value: nil, want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := extractFloatCurrentValue(tt.value); got != tt.want {
				t.Fatalf("extractFloatCurrentValue() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseFloatInput(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		current float64
		path    string
		value   float64
		changed bool
		status  floatInputStatus
	}{
		{name: "keep", input: "", current: 0.2, path: "x", value: 0.2, changed: false, status: floatInputKeep},
		{name: "valid generic", input: "0.7", current: 0.2, path: "x", value: 0.7, changed: true, status: floatInputValid},
		{name: "invalid nan", input: "NaN", current: 0.2, path: "x", value: 0.2, changed: false, status: floatInputInvalid},
		{
			name:    "context ratio out of range",
			input:   "0.5",
			current: config.ProjectMapContextRatioDefault,
			path:    "project_map.context_ratio",
			value:   config.ProjectMapContextRatioDefault,
			changed: false,
			status:  floatInputOutOfRange,
		},
		{
			name:    "context ratio valid",
			input:   "0.05",
			current: config.ProjectMapContextRatioDefault,
			path:    "project_map.context_ratio",
			value:   0.05,
			changed: true,
			status:  floatInputValid,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value, changed, status := parseFloatInput(tt.input, tt.current, tt.path)
			if value != tt.value || changed != tt.changed || status != tt.status {
				t.Fatalf("parseFloatInput() = (%v,%v,%v), want (%v,%v,%v)", value, changed, status, tt.value, tt.changed, tt.status)
			}
		})
	}
}

func TestParseStringInput(t *testing.T) {
	value, changed := parseStringInput("  hello  ", "old")
	if value != "hello" || !changed {
		t.Fatalf("parseStringInput() = (%q,%v), want (%q,true)", value, changed, "hello")
	}

	value, changed = parseStringInput("   ", "old")
	if value != "old" || changed {
		t.Fatalf("parseStringInput() keep = (%q,%v), want (%q,false)", value, changed, "old")
	}
}

func TestParseSelectInput(t *testing.T) {
	options := []string{"a", "b", "c"}

	value, changed, valid := parseSelectInput("", "b", options)
	if value != "b" || changed || !valid {
		t.Fatalf("parseSelectInput() keep = (%q,%v,%v), want (%q,false,true)", value, changed, valid, "b")
	}

	value, changed, valid = parseSelectInput("3", "b", options)
	if value != "c" || !changed || !valid {
		t.Fatalf("parseSelectInput() valid = (%q,%v,%v), want (%q,true,true)", value, changed, valid, "c")
	}

	value, changed, valid = parseSelectInput("9", "b", options)
	if value != "b" || changed || valid {
		t.Fatalf("parseSelectInput() invalid = (%q,%v,%v), want (%q,false,false)", value, changed, valid, "b")
	}
}
