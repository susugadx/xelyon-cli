package repomap

import "testing"

func TestPatterns_Go(t *testing.T) {
	if !matchesSymbolPattern("main.go", "func Build() error {") {
		t.Fatal("Go func pattern did not match")
	}
	if !matchesSymbolPattern("main.go", "type Config struct {") {
		t.Fatal("Go type pattern did not match")
	}
	if matchesSymbolPattern("main.go", "fmt.Println(\"hello\")") {
		t.Fatal("non-definition Go line should not match")
	}
}

func TestPatterns_TypeScript(t *testing.T) {
	if !matchesSymbolPattern("app.ts", "export function buildMap() {") {
		t.Fatal("TypeScript export function pattern did not match")
	}
	if !matchesSymbolPattern("app.ts", "const buildArrow = () => {") {
		t.Fatal("TypeScript arrow function pattern did not match")
	}
	if !matchesSymbolPattern("app.ts", "const buildAsyncArrow = async () => {") {
		t.Fatal("TypeScript async arrow function pattern did not match")
	}
	if !matchesSymbolPattern("app.ts", "interface Config {") {
		t.Fatal("TypeScript interface pattern did not match")
	}
	if matchesSymbolPattern("app.ts", "console.log(value)") {
		t.Fatal("non-definition TypeScript line should not match")
	}
}

func TestPatterns_Python(t *testing.T) {
	if !matchesSymbolPattern("tasks.py", "def build_map():") {
		t.Fatal("Python def pattern did not match")
	}
	if !matchesSymbolPattern("tasks.py", "class Builder:") {
		t.Fatal("Python class pattern did not match")
	}
	if matchesSymbolPattern("tasks.py", "print(value)") {
		t.Fatal("non-definition Python line should not match")
	}
}

func TestPatterns_Rust(t *testing.T) {
	if !matchesSymbolPattern("lib.rs", "pub async fn build_map() {") {
		t.Fatal("Rust fn pattern did not match")
	}
	if !matchesSymbolPattern("lib.rs", "pub struct Builder {") {
		t.Fatal("Rust struct pattern did not match")
	}
	if matchesSymbolPattern("lib.rs", "println!(\"hello\");") {
		t.Fatal("non-definition Rust line should not match")
	}
}

func TestPatterns_CommentExclusion(t *testing.T) {
	if matchesSymbolPattern("main.go", "// func Build() error") {
		t.Fatal("commented Go definition should not match")
	}
	if matchesSymbolPattern("tasks.py", "# def build_map():") {
		t.Fatal("commented Python definition should not match")
	}
	if matchesSymbolPattern("build.sh", "# function build_map") {
		t.Fatal("commented Shell definition should not match")
	}
}

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

func TestPatterns_MultiLanguageRegression(t *testing.T) {
	positive := []struct {
		name string
		path string
		line string
	}{
		{name: "Java interface", path: "src/Runner.java", line: "public interface Runner {"},
		{name: "Kotlin data class", path: "src/Runner.kt", line: "data class Runner(val id: String)"},
		{name: "Ruby module", path: "lib/runner.rb", line: "  module Runner"},
		{name: "PHP function", path: "web/runner.php", line: "protected static function buildMap() {"},
		{name: "Swift protocol", path: "ios/Runner.swift", line: "public protocol Runner {"},
		{name: "Scala case class", path: "src/Runner.scala", line: "case class Runner(id: String)"},
		{name: "Shell function", path: "scripts/build.sh", line: "function build_map"},
	}
	for _, tt := range positive {
		t.Run(tt.name, func(t *testing.T) {
			if !matchesSymbolPattern(tt.path, tt.line) {
				t.Fatalf("matchesSymbolPattern(%q, %q) = false, want true", tt.path, tt.line)
			}
		})
	}

	negative := []struct {
		name string
		path string
		line string
	}{
		{name: "Java statement", path: "src/Runner.java", line: "if (ready) {"},
		{name: "Kotlin assignment", path: "src/Runner.kt", line: "val runner = Runner()"},
		{name: "Ruby call", path: "lib/runner.rb", line: "puts runner"},
		{name: "PHP echo", path: "web/runner.php", line: "echo $runner;"},
		{name: "Swift call", path: "ios/Runner.swift", line: "print(runner)"},
		{name: "Scala call", path: "src/Runner.scala", line: "println(runner)"},
		{name: "Shell command", path: "scripts/build.sh", line: "echo build"},
	}
	for _, tt := range negative {
		t.Run(tt.name, func(t *testing.T) {
			if matchesSymbolPattern(tt.path, tt.line) {
				t.Fatalf("matchesSymbolPattern(%q, %q) = true, want false", tt.path, tt.line)
			}
		})
	}
}

func TestPatterns_CppStructAndDefine(t *testing.T) {
	if !matchesSymbolPattern("lib.h", "struct MyConfig {") {
		t.Fatal("C struct pattern did not match")
	}
	if !matchesSymbolPattern("lib.h", "#define MAX_SIZE 1024") {
		t.Fatal("C #define pattern did not match")
	}
	if !matchesSymbolPattern("lib.cpp", "namespace utils {") {
		t.Fatal("C++ namespace pattern did not match")
	}
}

func TestPatterns_CppNoControlFlow(t *testing.T) {
	for _, line := range []string{
		"if (condition) {",
		"while (true) {",
		"for (int i = 0; i < n; i++) {",
		"switch (value) {",
		"return (result);",
	} {
		if matchesSymbolPattern("main.c", line) {
			t.Fatalf("control flow should not match: %s", line)
		}
	}
}

func TestExtractSignatureMetadata(t *testing.T) {
	tests := []struct {
		name     string
		sig      string
		wantName string
		wantKind string
	}{
		{
			name:     "Go method",
			sig:      "func (a *Agent) maybeAutoCompress() bool",
			wantName: "maybeAutoCompress",
			wantKind: "method",
		},
		{
			name:     "Python async def",
			sig:      "async def build_map():",
			wantName: "build_map",
			wantKind: "function",
		},
		{
			name:     "TypeScript interface",
			sig:      "export interface Config",
			wantName: "Config",
			wantKind: "interface",
		},
		{
			name:     "TypeScript async arrow function",
			sig:      "const buildMap = async (): Promise<Map<string, string>> => {",
			wantName: "buildMap",
			wantKind: "function",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotName, gotKind, ok := extractSignatureMetadata(tt.sig)
			if !ok {
				t.Fatalf("extractSignatureMetadata(%q) returned ok=false", tt.sig)
			}
			if gotName != tt.wantName {
				t.Fatalf("name = %q, want %q", gotName, tt.wantName)
			}
			if gotKind != tt.wantKind {
				t.Fatalf("kind = %q, want %q", gotKind, tt.wantKind)
			}
		})
	}
}

func TestSignatureMetadataForPath_TypeScriptArrowFunction(t *testing.T) {
	gotName, gotKind, exported := signatureMetadataForPath("app.ts", "const buildMap = async (): Promise<Map<string, string>> =>")
	if gotName != "buildMap" {
		t.Fatalf("name = %q, want %q", gotName, "buildMap")
	}
	if gotKind != "function" {
		t.Fatalf("kind = %q, want %q", gotKind, "function")
	}
	if exported {
		t.Fatal("exported = true, want false")
	}
}

func TestExtractJSArrowFunctionMetadata(t *testing.T) {
	tests := []struct {
		name     string
		sig      string
		wantName string
		wantKind string
		wantOK   bool
	}{
		{
			name:     "const arrow",
			sig:      "const buildMap = () => {",
			wantName: "buildMap",
			wantKind: "function",
			wantOK:   true,
		},
		{
			name:     "const async arrow with return type",
			sig:      "const buildMap = async (): Promise<Map<string, string>> => {",
			wantName: "buildMap",
			wantKind: "function",
			wantOK:   true,
		},
		{
			name:     "export const async arrow",
			sig:      "export const buildMap = async () => {",
			wantName: "buildMap",
			wantKind: "function",
			wantOK:   true,
		},
		{
			name:   "plain const value",
			sig:    "const buildMap = new Map()",
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotName, gotKind, ok := extractJSArrowFunctionMetadata(tt.sig)
			if ok != tt.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tt.wantOK)
			}
			if !tt.wantOK {
				return
			}
			if gotName != tt.wantName {
				t.Fatalf("name = %q, want %q", gotName, tt.wantName)
			}
			if gotKind != tt.wantKind {
				t.Fatalf("kind = %q, want %q", gotKind, tt.wantKind)
			}
		})
	}
}
