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
