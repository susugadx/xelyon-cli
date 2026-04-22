package repomap

import "testing"

func TestSupportsSymbols_LanguageCoverage(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{path: "main.go", want: true},
		{path: "app.tsx", want: true},
		{path: "tasks.py", want: true},
		{path: "lib.rs", want: true},
		{path: "src/Main.java", want: true},
		{path: "src/Main.kt", want: true},
		{path: "src/Main.kts", want: true},
		{path: "lib/runner.rb", want: true},
		{path: "web/index.php", want: true},
		{path: "main.c", want: true},
		{path: "main.hpp", want: true},
		{path: "App.swift", want: true},
		{path: "App.scala", want: true},
		{path: "build.sh", want: true},
		{path: "build.bash", want: true},
		{path: "build.zsh", want: true},
		{path: "README.md", want: false},
		{path: "Dockerfile", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			if got := supportsSymbols(tt.path); got != tt.want {
				t.Fatalf("supportsSymbols(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}
