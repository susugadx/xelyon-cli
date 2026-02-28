package common

import (
	"testing"
)

func TestExtractBlockName(t *testing.T) {
	tests := []struct {
		line     string
		expected string
	}{
		// Go
		{"func hello() {}", "func hello"},
		{"func (s *Server) Start() {", "func Start"},
		{"func (s *Server) handleRequest(w http.ResponseWriter) {", "func handleRequest"},
		{"type Config struct {", "type Config"},
		{"type Handler interface {", "type Handler"},
		// Python
		{"def process(data):", "def process"},
		{"class MyClass:", "class MyClass"},
		// JavaScript
		{"function doSomething(arg1, arg2) {", "function doSomething"},
		// Rust
		{"fn main() {", "fn main"},
		{"struct Point {", "struct Point"},
		{"enum Color {", "enum Color"},
		{"trait Display {", "trait Display"},
		// その他
		{"interface Runnable {", "interface Runnable"},
		{"method run() {", "method run"},
		// 非宣言行 → 空文字列
		{"x := 42", ""},
		{"if cond {", ""},
		{"fmt.Println(hello)", ""},
		{"// func fake() {}", ""},
		{"", ""},
	}
	for _, tt := range tests {
		got := ExtractBlockName(tt.line)
		if got != tt.expected {
			t.Errorf("ExtractBlockName(%q) = %q, want %q", tt.line, got, tt.expected)
		}
	}
}

func TestExtractBlockName_Modifiers(t *testing.T) {
	tests := []struct {
		line     string
		expected string
	}{
		{"async def process():", "def process"},
		{"export default function doSomething() {", "function doSomething"},
		{"go func() {", "func(anonymous)"},
		{"defer func() {", "func(anonymous)"},
		{"public class MyClass {", "class MyClass"},
		{"export async function fetchData() {", "function fetchData"},
		{"async def async_function():", "def async_function"},
	}
	for _, tt := range tests {
		got := ExtractBlockName(tt.line)
		if got != tt.expected {
			t.Errorf("ExtractBlockName(%q) = %q, want %q", tt.line, got, tt.expected)
		}
	}
}

func TestExtractBlockName_VarAssignment(t *testing.T) {
	tests := []struct {
		line     string
		expected string
	}{
		{"const config =", "const config"},
		{"var count =", "var count"},
		{"let isReady =", "let isReady"},
		{"export const MyComponent = () => {", "const MyComponent"},
		{"const handler = (req, res) => {", "const handler"},
		{"var processor = func(data string) {", "var processor"},
	}
	for _, tt := range tests {
		got := ExtractBlockName(tt.line)
		if got != tt.expected {
			t.Errorf("ExtractBlockName(%q) = %q, want %q", tt.line, got, tt.expected)
		}
	}
}

func TestExtractBlockName_AnonymousFunc(t *testing.T) {
	tests := []struct {
		line     string
		expected string
	}{
		{"func() {", "func(anonymous)"},
		{"go func() {", "func(anonymous)"},
		{"defer func() {", "func(anonymous)"},
	}
	for _, tt := range tests {
		got := ExtractBlockName(tt.line)
		if got != tt.expected {
			t.Errorf("ExtractBlockName(%q) = %q, want %q", tt.line, got, tt.expected)
		}
	}
}

func TestExtractBlockName_Generics(t *testing.T) {
	tests := []struct {
		line     string
		expected string
	}{
		{"func Process[T any](data T) {", "func Process"},
		{"func Convert<T>() {", "func Convert"},
		{"type Cache[K comparable, V any] struct {", "type Cache"},
		{"fn process<T: Clone>(item: T) {", "fn process"},
		{"class Container<T> {", "class Container"},
		{"function process<T>(value: T) {", "function process"},
	}
	for _, tt := range tests {
		got := ExtractBlockName(tt.line)
		if got != tt.expected {
			t.Errorf("ExtractBlockName(%q) = %q, want %q", tt.line, got, tt.expected)
		}
	}
}

func TestExtractBlockName_ArrowFunction(t *testing.T) {
	tests := []struct {
		line     string
		expected string
	}{
		{"const x = () => {", "const x"},
		{"let multiply = (x, y) => x * y", "let multiply"},
		{"export const handleClick = () => {", "const handleClick"},
	}
	for _, tt := range tests {
		got := ExtractBlockName(tt.line)
		if got != tt.expected {
			t.Errorf("ExtractBlockName(%q) = %q, want %q", tt.line, got, tt.expected)
		}
	}
}

func TestStripModifiers(t *testing.T) {
	tests := []struct {
		line     string
		expected string
	}{
		{"async export default public private protected static abstract override final virtual inline go defer func()", "func()"},
		{"async const x = 1", "const x = 1"},
		{"async def process():", "def process():"},
		{"export function App() {", "function App() {"},
		{"public static void main() {", "void main() {"},
		{"", ""},
	}
	for _, tt := range tests {
		got := StripModifiers(tt.line)
		if got != tt.expected {
			t.Errorf("StripModifiers(%q) = %q, want %q", tt.line, got, tt.expected)
		}
	}
}

func TestIsCommentLine(t *testing.T) {
	tests := []struct {
		line     string
		expected bool
	}{
		{"// Go comment", true},
		{"/* block comment */", true},
		{"# Python comment", true},
		{"-- SQL comment", true},
		{"; Lisp comment", true},
		{`"""docstring"""`, true},
		{"'''docstring'''", true},
		{"func foo() {}", false},
		{"x := 1 // inline comment", false},
		{"", false},
	}
	for _, tt := range tests {
		got := IsCommentLine(tt.line)
		if got != tt.expected {
			t.Errorf("IsCommentLine(%q) = %v, want %v", tt.line, got, tt.expected)
		}
	}
}

func TestCountIndent(t *testing.T) {
	tests := []struct {
		line     string
		expected int
	}{
		{"no indent", 0},
		{"  two spaces", 2},
		{"    four spaces", 4},
		{"\tone tab", 4},
		{"\t\ttwo tabs", 8},
		{"  \tmixed", 6}, // 2 spaces + 1 tab(4)
		{"", 0},
	}
	for _, tt := range tests {
		got := CountIndent(tt.line)
		if got != tt.expected {
			t.Errorf("CountIndent(%q) = %d, want %d", tt.line, got, tt.expected)
		}
	}
}

func TestIsBraceLanguage(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		{"main.go", true},
		{"app.js", true},
		{"index.ts", true},
		{"Main.java", true},
		{"lib.rs", true},
		{"main.c", true},
		{"script.py", false},
		{"script.pyw", false},
		{"config.yaml", false},
		{"config.yml", false},
		{"app.coffee", false},
	}
	for _, tt := range tests {
		got := IsBraceLanguage(tt.path)
		if got != tt.expected {
			t.Errorf("IsBraceLanguage(%q) = %v, want %v", tt.path, got, tt.expected)
		}
	}
}

func TestStripQuoted(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		// ダブルクォート
		{`x = "hello"`, `x = `},
		{`fmt.Println("hello=world")`, `fmt.Println()`},
		// シングルクォート
		{`x = 'hello'`, `x = `},
		// エスケープ
		{`x = "he\"llo"`, `x = `},
		{`x = 'he\'llo'`, `x = `},
		// 混在
		{`x = "it's"`, `x = `},
		{`x = 'say "hi"'`, `x = `},
		// クォートなし
		{"x := 42", "x := 42"},
		{"", ""},
		// URL 内の = がクォート内にある
		{`url := "https://example.com?key=value"`, `url := `},
		// バッククォート（Go struct tag 等）
		{"Name string `json:\"name=value\"`", "Name string "},
		{"`entire backtick`", ""},
		{"before `inside` after", "before  after"},
	}
	for _, tt := range tests {
		got := StripQuoted(tt.input)
		if got != tt.expected {
			t.Errorf("StripQuoted(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestBuildBlockMap_BraceLanguage(t *testing.T) {
	content := `package main

func foo() {
	x := 1
}

func bar() {
	if cond {
		y := 2
	}
}

type Server struct {
	addr string
}

func (s *Server) Start() {
	listen()
}
`
	ranges := BuildBlockMap(content, true)

	// ブロック名を収集
	names := make(map[string]bool)
	for _, r := range ranges {
		names[r.Name] = true
	}

	// func foo, func bar, type Server, func Start が検出されること
	expectedNames := []string{"func foo", "func bar", "type Server", "func Start"}
	for _, name := range expectedNames {
		if !names[name] {
			t.Errorf("Expected block '%s' to be detected, got blocks: %v", name, ranges)
		}
	}

	// 複数行コメント内の func 宣言はブロックとして検出されないこと
	contentWithComment := `package main

func real() {
	x := 1
}

/*
func fake() {
	y := 2
}
*/

func another() {
	z := 3
}
`
	rangesC := BuildBlockMap(contentWithComment, true)
	for _, r := range rangesC {
		if r.Name == "func fake" {
			t.Error("func fake inside /* */ block comment should NOT be detected as a block")
		}
	}

	// real と another は検出されること
	namesC := make(map[string]bool)
	for _, r := range rangesC {
		namesC[r.Name] = true
	}
	if !namesC["func real"] {
		t.Error("Expected 'func real' to be detected")
	}
	if !namesC["func another"] {
		t.Error("Expected 'func another' to be detected")
	}

	// 同一行 /* ... */ コメントの後の func が正しく検出されること
	contentInline := `package main

func before() {
	x := 1
}

/* inline comment */ func after() {
	y := 2
}

func last() {
	z := 3
}
`
	rangesI := BuildBlockMap(contentInline, true)
	namesI := make(map[string]bool)
	for _, r := range rangesI {
		namesI[r.Name] = true
	}
	if !namesI["func before"] {
		t.Error("Expected 'func before' to be detected")
	}
	if !namesI["func last"] {
		t.Error("Expected 'func last' to be detected after inline /* */ comment line")
	}
}

func TestBuildBlockMap_IndentLanguage(t *testing.T) {
	content := `class MyClass:
    def process(self):
        x = 1
        y = 2
    def cleanup(self):
        pass

def standalone():
    z = 3
`
	ranges := BuildBlockMap(content, false)

	// ブロック名を収集
	names := make(map[string]bool)
	for _, r := range ranges {
		names[r.Name] = true
	}

	// class MyClass, def process, def cleanup, def standalone が検出されること
	expectedNames := []string{"class MyClass", "def process", "def cleanup", "def standalone"}
	for _, name := range expectedNames {
		if !names[name] {
			t.Errorf("Expected block '%s' to be detected, got blocks: %v", name, ranges)
		}
	}
}
