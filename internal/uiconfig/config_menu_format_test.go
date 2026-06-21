package uiconfig

import "testing"

func TestFormatValue(t *testing.T) {
	tests := []struct {
		name string
		in   interface{}
		want string
	}{
		{name: "nil", in: nil, want: "null"},
		{name: "bool true", in: true, want: "true"},
		{name: "empty string", in: "", want: "(empty)"},
		{name: "string slice", in: []string{"a", "b"}, want: "[2 items]"},
		{name: "string map", in: map[string]string{"A": "B"}, want: "{1 entries}"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatValue(tt.in); got != tt.want {
				t.Fatalf("formatValue(%#v) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
