package uiruntime

import "testing"

func TestIsMultilineMarker(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"```", true},
		{"`` `", false},
		{"```\n", false},
		{" ```", false},
		{"test", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := IsMultilineMarker(tt.input)
			if got != tt.want {
				t.Errorf("IsMultilineMarker(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestTrimBracketedPasteMarkers(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "with both markers",
			input: "\x1b[200~hello world\x1b[201~",
			want:  "hello world",
		},
		{
			name:  "with start marker only",
			input: "\x1b[200~hello world",
			want:  "hello world",
		},
		{
			name:  "with end marker only",
			input: "hello world\x1b[201~",
			want:  "hello world",
		},
		{
			name:  "no markers",
			input: "hello world",
			want:  "hello world",
		},
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TrimBracketedPasteMarkers(tt.input)
			if got != tt.want {
				t.Errorf("TrimBracketedPasteMarkers() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestStripAllBracketedPasteMarkers(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "ESC markers",
			input: "\x1b[200~hello\x1b[201~",
			want:  "hello",
		},
		{
			name:  "literal markers",
			input: "^[[200~hello^[[201~",
			want:  "hello",
		},
		{
			name:  "mixed markers",
			input: "\x1b[200~hello^[[201~",
			want:  "hello",
		},
		{
			name:  "multiple markers",
			input: "^[[200~a^[[200~b^[[201~c^[[201~",
			want:  "abc",
		},
		{
			name:  "no markers",
			input: "hello world",
			want:  "hello world",
		},
		{
			name:  "empty",
			input: "",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stripAllBracketedPasteMarkers(tt.input)
			if got != tt.want {
				t.Errorf("stripAllBracketedPasteMarkers() = %q, want %q", got, tt.want)
			}
		})
	}
}
