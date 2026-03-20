package agent

import "testing"

func TestIsDigitOrRange(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{input: "42", want: true},
		{input: "10-20", want: true},
		{input: "", want: false},
		{input: "10-", want: false},
		{input: "-20", want: false},
		{input: "a20", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := isDigitOrRange(tt.input); got != tt.want {
				t.Fatalf("isDigitOrRange(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestReadFileHasExplicitRange(t *testing.T) {
	tests := []struct {
		name string
		args map[string]string
		want bool
	}{
		{
			name: "single path without range",
			args: map[string]string{"path": "main.go"},
			want: false,
		},
		{
			name: "single path with start line",
			args: map[string]string{"path": "main.go", "start_line": "10"},
			want: true,
		},
		{
			name: "batch paths with explicit range",
			args: map[string]string{"paths": `["main.go:10-20","util.go"]`},
			want: true,
		},
		{
			name: "batch paths without range",
			args: map[string]string{"paths": `["main.go","util.go"]`},
			want: false,
		},
		{
			name: "windows path stays broad",
			args: map[string]string{"paths": `["C:\\src\\main.go"]`},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := readFileHasExplicitRange(tt.args); got != tt.want {
				t.Fatalf("readFileHasExplicitRange(%v) = %v, want %v", tt.args, got, tt.want)
			}
		})
	}
}
