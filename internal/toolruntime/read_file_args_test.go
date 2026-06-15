package toolruntime

import "testing"

func TestReadFilePathsFromArgsParsesPathsAndFallbackPath(t *testing.T) {
	got := ReadFilePathsFromArgs(map[string]string{"paths": `[" a.go ","","b.go"]`})
	if len(got) != 2 || got[0] != "a.go" || got[1] != "b.go" {
		t.Fatalf("ReadFilePathsFromArgs(paths) = %#v, want trimmed non-empty paths", got)
	}

	got = ReadFilePathsFromArgs(map[string]string{"path": " single.go "})
	if len(got) != 1 || got[0] != "single.go" {
		t.Fatalf("ReadFilePathsFromArgs(path) = %#v, want single.go", got)
	}

	if got := ReadFilePathsFromArgs(map[string]string{"paths": `{"bad":true}`}); got != nil {
		t.Fatalf("ReadFilePathsFromArgs(invalid paths) = %#v, want nil", got)
	}
}

func TestReadFileHasExplicitRangeDetectsLineArgsAndPathSuffix(t *testing.T) {
	tests := []struct {
		name string
		args map[string]string
		want bool
	}{
		{name: "start line", args: map[string]string{"path": "a.go", "start_line": "10"}, want: true},
		{name: "path range", args: map[string]string{"path": "a.go:10-20"}, want: true},
		{name: "path line", args: map[string]string{"path": "a.go:12"}, want: true},
		{name: "non numeric suffix", args: map[string]string{"path": "a.go:main"}, want: false},
		{name: "no range", args: map[string]string{"path": "a.go"}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ReadFileHasExplicitRange(tt.args); got != tt.want {
				t.Fatalf("ReadFileHasExplicitRange() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsDigitOrRange(t *testing.T) {
	tests := []struct {
		value string
		want  bool
	}{
		{value: "12", want: true},
		{value: "12-20", want: true},
		{value: "-20", want: false},
		{value: "12-", want: false},
		{value: "12-20-30", want: false},
		{value: "", want: false},
	}

	for _, tt := range tests {
		if got := IsDigitOrRange(tt.value); got != tt.want {
			t.Fatalf("IsDigitOrRange(%q) = %v, want %v", tt.value, got, tt.want)
		}
	}
}
