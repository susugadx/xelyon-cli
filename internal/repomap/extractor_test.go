//go:build !norepomap
// +build !norepomap

package repomap

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/testutil"
)

func TestExtractGoFunction(t *testing.T) {
	content := `package main

func Hello(name string) string {
	return "Hello, " + name
}

func (s *Server) Start(port int) error {
	return nil
}
`
	tmpDir := t.TempDir()
	testutil.CreateTempFile(t, tmpDir, "test.go", content)
	testFile := filepath.Join(tmpDir, "test.go")

	fileSymbols, err := ExtractSymbols(testFile)
	if err != nil {
		t.Fatalf("ExtractSymbols failed: %v", err)
	}
	if fileSymbols == nil {
		t.Fatal("Expected fileSymbols, got nil")
	}

	if len(fileSymbols.Symbols) != 2 {
		t.Fatalf("Expected 2 symbols, got %d", len(fileSymbols.Symbols))
	}

	// Hello関数の検証
	hello := fileSymbols.Symbols[0]
	if hello.Name != "Hello" {
		t.Errorf("Expected 'Hello', got '%s'", hello.Name)
	}
	if hello.Kind != "function" {
		t.Errorf("Expected 'function', got '%s'", hello.Kind)
	}
	if !strings.Contains(hello.Signature, "func Hello") {
		t.Errorf("Expected signature to contain 'func Hello', got '%s'", hello.Signature)
	}

	// Startメソッドの検証
	start := fileSymbols.Symbols[1]
	if start.Name != "Start" {
		t.Errorf("Expected 'Start', got '%s'", start.Name)
	}
	if start.Kind != "method" {
		t.Errorf("Expected 'method', got '%s'", start.Kind)
	}
}

func TestExtractGoType(t *testing.T) {
	content := `package main

type User struct {
	Name string
	Age  int
}

type Reader interface {
	Read(p []byte) (n int, err error)
}
`
	tmpDir := t.TempDir()
	testutil.CreateTempFile(t, tmpDir, "types.go", content)
	testFile := filepath.Join(tmpDir, "types.go")

	fileSymbols, err := ExtractSymbols(testFile)
	if err != nil {
		t.Fatalf("ExtractSymbols failed: %v", err)
	}
	if fileSymbols == nil {
		t.Fatal("Expected fileSymbols, got nil")
	}

	if len(fileSymbols.Symbols) != 2 {
		t.Fatalf("Expected 2 symbols, got %d", len(fileSymbols.Symbols))
	}

	// User struct
	user := fileSymbols.Symbols[0]
	if user.Name != "User" {
		t.Errorf("Expected 'User', got '%s'", user.Name)
	}
	if user.Kind != "struct" {
		t.Errorf("Expected 'struct', got '%s'", user.Kind)
	}

	// Reader interface
	reader := fileSymbols.Symbols[1]
	if reader.Name != "Reader" {
		t.Errorf("Expected 'Reader', got '%s'", reader.Name)
	}
	if reader.Kind != "interface" {
		t.Errorf("Expected 'interface', got '%s'", reader.Kind)
	}
}

func TestExtractRustFunction(t *testing.T) {
	content := `fn hello_world(name: &str) -> String {
	format!("Hello, {}!", name)
}

pub fn add(a: i32, b: i32) -> i32 {
	a + b
}
`
	tmpDir := t.TempDir()
	testutil.CreateTempFile(t, tmpDir, "test.rs", content)
	testFile := filepath.Join(tmpDir, "test.rs")

	fileSymbols, err := ExtractSymbols(testFile)
	if err != nil {
		t.Fatalf("ExtractSymbols failed: %v", err)
	}
	if fileSymbols == nil {
		t.Fatal("Expected fileSymbols, got nil")
	}

	if len(fileSymbols.Symbols) < 2 {
		t.Fatalf("Expected at least 2 symbols, got %d", len(fileSymbols.Symbols))
	}

	// hello_world関数
	hello := fileSymbols.Symbols[0]
	if hello.Name != "hello_world" {
		t.Errorf("Expected 'hello_world', got '%s'", hello.Name)
	}
	if hello.Kind != "function" {
		t.Errorf("Expected 'function', got '%s'", hello.Kind)
	}
	if !strings.Contains(hello.Signature, "fn hello_world") {
		t.Errorf("Expected signature to contain 'fn hello_world', got '%s'", hello.Signature)
	}
}

func TestExtractRustStruct(t *testing.T) {
	content := `struct Point {
	x: i32,
	y: i32,
}

enum Color {
	Red,
	Green,
	Blue,
}
`
	tmpDir := t.TempDir()
	testutil.CreateTempFile(t, tmpDir, "types.rs", content)
	testFile := filepath.Join(tmpDir, "types.rs")

	fileSymbols, err := ExtractSymbols(testFile)
	if err != nil {
		t.Fatalf("ExtractSymbols failed: %v", err)
	}
	if fileSymbols == nil {
		t.Fatal("Expected fileSymbols, got nil")
	}

	if len(fileSymbols.Symbols) < 2 {
		t.Fatalf("Expected at least 2 symbols, got %d", len(fileSymbols.Symbols))
	}

	// Point struct
	point := fileSymbols.Symbols[0]
	if point.Name != "Point" {
		t.Errorf("Expected 'Point', got '%s'", point.Name)
	}
	if point.Kind != "struct" {
		t.Errorf("Expected 'struct', got '%s'", point.Kind)
	}

	// Color enum
	color := fileSymbols.Symbols[1]
	if color.Name != "Color" {
		t.Errorf("Expected 'Color', got '%s'", color.Name)
	}
	if color.Kind != "enum" {
		t.Errorf("Expected 'enum', got '%s'", color.Kind)
	}
}

func TestExtractJSFunction(t *testing.T) {
	content := `function fetchUser(userId) {
	return api.get('/users/' + userId);
}

function syncFunc(a, b) {
	return a + b;
}
`
	tmpDir := t.TempDir()
	testutil.CreateTempFile(t, tmpDir, "test.js", content)
	testFile := filepath.Join(tmpDir, "test.js")

	fileSymbols, err := ExtractSymbols(testFile)
	if err != nil {
		t.Fatalf("ExtractSymbols failed: %v", err)
	}
	if fileSymbols == nil {
		t.Fatal("Expected fileSymbols, got nil")
	}

	// 関数シンボルを探す
	var functions []Symbol
	for _, s := range fileSymbols.Symbols {
		if s.Kind == "function" {
			functions = append(functions, s)
		}
	}

	if len(functions) < 2 {
		t.Fatalf("Expected at least 2 function symbols, got %d", len(functions))
	}

	// fetchUser関数
	fetchUser := functions[0]
	if fetchUser.Name != "fetchUser" {
		t.Errorf("Expected 'fetchUser', got '%s'", fetchUser.Name)
	}
	// 引数が含まれているか確認
	if !strings.Contains(fetchUser.Signature, "userId") {
		t.Errorf("Expected signature to contain 'userId', got '%s'", fetchUser.Signature)
	}
}

func TestExtractJSClass(t *testing.T) {
	content := `class User {
	constructor(name) {
		this.name = name;
	}

	greet() {
		return "Hello, " + this.name;
	}
}
`
	tmpDir := t.TempDir()
	testutil.CreateTempFile(t, tmpDir, "user.js", content)
	testFile := filepath.Join(tmpDir, "user.js")

	fileSymbols, err := ExtractSymbols(testFile)
	if err != nil {
		t.Fatalf("ExtractSymbols failed: %v", err)
	}
	if fileSymbols == nil {
		t.Fatal("Expected fileSymbols, got nil")
	}

	// class + constructor + greet = 3 symbols
	if len(fileSymbols.Symbols) < 1 {
		t.Fatalf("Expected at least 1 symbol, got %d", len(fileSymbols.Symbols))
	}

	// User class
	user := fileSymbols.Symbols[0]
	if user.Name != "User" {
		t.Errorf("Expected 'User', got '%s'", user.Name)
	}
	if user.Kind != "class" {
		t.Errorf("Expected 'class', got '%s'", user.Kind)
	}
}

func TestExtractPythonFunction(t *testing.T) {
	content := `def fetch_user(user_id: str, options: dict = None) -> User:
    pass

def simple_add(a, b):
    return a + b
`
	tmpDir := t.TempDir()
	testutil.CreateTempFile(t, tmpDir, "test.py", content)
	testFile := filepath.Join(tmpDir, "test.py")

	fileSymbols, err := ExtractSymbols(testFile)
	if err != nil {
		t.Fatalf("ExtractSymbols failed: %v", err)
	}
	if fileSymbols == nil {
		t.Fatal("Expected fileSymbols, got nil")
	}

	if len(fileSymbols.Symbols) != 2 {
		t.Fatalf("Expected 2 symbols, got %d", len(fileSymbols.Symbols))
	}

	// fetch_user関数
	fetchUser := fileSymbols.Symbols[0]
	if fetchUser.Name != "fetch_user" {
		t.Errorf("Expected 'fetch_user', got '%s'", fetchUser.Name)
	}
	// 引数が含まれているか確認
	if !strings.Contains(fetchUser.Signature, "user_id") {
		t.Errorf("Expected signature to contain 'user_id', got '%s'", fetchUser.Signature)
	}
}

func TestExtractPythonClass(t *testing.T) {
	content := `class User:
    def __init__(self, name):
        self.name = name

    def greet(self):
        return f"Hello, {self.name}"
`
	tmpDir := t.TempDir()
	testutil.CreateTempFile(t, tmpDir, "user.py", content)
	testFile := filepath.Join(tmpDir, "user.py")

	fileSymbols, err := ExtractSymbols(testFile)
	if err != nil {
		t.Fatalf("ExtractSymbols failed: %v", err)
	}
	if fileSymbols == nil {
		t.Fatal("Expected fileSymbols, got nil")
	}

	// class + __init__ + greet = 3 symbols (depends on tree-sitter behavior)
	if len(fileSymbols.Symbols) < 1 {
		t.Fatalf("Expected at least 1 symbol, got %d", len(fileSymbols.Symbols))
	}

	// User class
	user := fileSymbols.Symbols[0]
	if user.Name != "User" {
		t.Errorf("Expected 'User', got '%s'", user.Name)
	}
	if user.Kind != "class" {
		t.Errorf("Expected 'class', got '%s'", user.Kind)
	}
}

func TestExtractJavaClass(t *testing.T) {
	content := `public class User {
    private String name;

    public User(String name) {
        this.name = name;
    }

    public String getName() {
        return name;
    }
}
`
	tmpDir := t.TempDir()
	testutil.CreateTempFile(t, tmpDir, "User.java", content)
	testFile := filepath.Join(tmpDir, "User.java")

	fileSymbols, err := ExtractSymbols(testFile)
	if err != nil {
		t.Fatalf("ExtractSymbols failed: %v", err)
	}
	if fileSymbols == nil {
		t.Fatal("Expected fileSymbols, got nil")
	}

	if len(fileSymbols.Symbols) < 1 {
		t.Fatalf("Expected at least 1 symbol, got %d", len(fileSymbols.Symbols))
	}

	// User class
	user := fileSymbols.Symbols[0]
	if user.Name != "User" {
		t.Errorf("Expected 'User', got '%s'", user.Name)
	}
	if user.Kind != "class" {
		t.Errorf("Expected 'class', got '%s'", user.Kind)
	}
}

func TestExtractRubyClass(t *testing.T) {
	content := `class User
  def initialize(name)
    @name = name
  end

  def greet
    "Hello, #{@name}"
  end
end
`
	tmpDir := t.TempDir()
	testutil.CreateTempFile(t, tmpDir, "user.rb", content)
	testFile := filepath.Join(tmpDir, "user.rb")

	fileSymbols, err := ExtractSymbols(testFile)
	if err != nil {
		t.Fatalf("ExtractSymbols failed: %v", err)
	}
	if fileSymbols == nil {
		t.Fatal("Expected fileSymbols, got nil")
	}

	if len(fileSymbols.Symbols) < 1 {
		t.Fatalf("Expected at least 1 symbol, got %d", len(fileSymbols.Symbols))
	}

	// User class
	user := fileSymbols.Symbols[0]
	if user.Name != "User" {
		t.Errorf("Expected 'User', got '%s'", user.Name)
	}
	if user.Kind != "class" {
		t.Errorf("Expected 'class', got '%s'", user.Kind)
	}
}

func TestExtractUnsupportedLanguage(t *testing.T) {
	content := `Just some text file content`

	tmpDir := t.TempDir()
	testutil.CreateTempFile(t, tmpDir, "readme.txt", content)
	testFile := filepath.Join(tmpDir, "readme.txt")

	fileSymbols, err := ExtractSymbols(testFile)
	if err != nil {
		t.Fatalf("ExtractSymbols failed: %v", err)
	}
	// サポートされていない言語はnilを返す
	if fileSymbols != nil {
		t.Errorf("Expected nil for unsupported language, got %v", fileSymbols)
	}
}

// Issue #58: TypeScript型注釈抽出テスト
func TestExtractTypeScriptFunction(t *testing.T) {
	content := `function fetchUser(userId: string, options?: RequestOptions): Promise<User> {
	return api.get('/users/' + userId);
}

async function createUser(name: string, age: number): Promise<User> {
	return api.post('/users', { name, age });
}
`
	tmpDir := t.TempDir()
	testutil.CreateTempFile(t, tmpDir, "test.ts", content)
	testFile := filepath.Join(tmpDir, "test.ts")

	fileSymbols, err := ExtractSymbols(testFile)
	if err != nil {
		t.Fatalf("ExtractSymbols failed: %v", err)
	}
	if fileSymbols == nil {
		t.Fatal("Expected fileSymbols, got nil")
	}

	// 関数シンボルを探す
	var functions []Symbol
	for _, s := range fileSymbols.Symbols {
		if s.Kind == "function" {
			functions = append(functions, s)
		}
	}

	if len(functions) < 2 {
		t.Fatalf("Expected at least 2 function symbols, got %d", len(functions))
	}

	// fetchUser関数: 型注釈が含まれているか
	fetchUser := functions[0]
	if fetchUser.Name != "fetchUser" {
		t.Errorf("Expected 'fetchUser', got '%s'", fetchUser.Name)
	}
	// パラメータ型注釈
	if !strings.Contains(fetchUser.Signature, "string") {
		t.Errorf("Expected signature to contain type 'string', got '%s'", fetchUser.Signature)
	}
	// 戻り値型注釈
	if !strings.Contains(fetchUser.Signature, "Promise<User>") {
		t.Errorf("Expected signature to contain return type 'Promise<User>', got '%s'", fetchUser.Signature)
	}

	// createUser関数: async + 型注釈
	createUser := functions[1]
	if createUser.Name != "createUser" {
		t.Errorf("Expected 'createUser', got '%s'", createUser.Name)
	}
	// async キーワード
	if !strings.Contains(createUser.Signature, "async") {
		t.Errorf("Expected signature to contain 'async', got '%s'", createUser.Signature)
	}
}

// Issue #58: TypeScriptメソッド型注釈テスト
func TestExtractTypeScriptMethod(t *testing.T) {
	content := `class UserService {
	async getUser(id: string): Promise<User> {
		return this.api.get(id);
	}

	updateUser(id: string, data: UserData): User {
		return this.api.put(id, data);
	}
}
`
	tmpDir := t.TempDir()
	testutil.CreateTempFile(t, tmpDir, "service.ts", content)
	testFile := filepath.Join(tmpDir, "service.ts")

	fileSymbols, err := ExtractSymbols(testFile)
	if err != nil {
		t.Fatalf("ExtractSymbols failed: %v", err)
	}
	if fileSymbols == nil {
		t.Fatal("Expected fileSymbols, got nil")
	}

	// メソッドシンボルを探す
	var methods []Symbol
	for _, s := range fileSymbols.Symbols {
		if s.Kind == "method" {
			methods = append(methods, s)
		}
	}

	if len(methods) < 2 {
		t.Fatalf("Expected at least 2 method symbols, got %d", len(methods))
	}

	// getUser: async + 型注釈
	getUser := methods[0]
	if getUser.Name != "getUser" {
		t.Errorf("Expected 'getUser', got '%s'", getUser.Name)
	}
	if !strings.Contains(getUser.Signature, "async") {
		t.Errorf("Expected signature to contain 'async', got '%s'", getUser.Signature)
	}
	if !strings.Contains(getUser.Signature, "string") {
		t.Errorf("Expected signature to contain 'string', got '%s'", getUser.Signature)
	}
}

// Issue #58: Python async def検出テスト
func TestExtractPythonAsyncFunction(t *testing.T) {
	content := `async def fetch_user(user_id: str) -> User:
    return await api.get(f'/users/{user_id}')

async def create_user(name: str, age: int) -> User:
    return await api.post('/users', {'name': name, 'age': age})

def sync_function(x: int) -> int:
    return x * 2
`
	tmpDir := t.TempDir()
	testutil.CreateTempFile(t, tmpDir, "async_test.py", content)
	testFile := filepath.Join(tmpDir, "async_test.py")

	fileSymbols, err := ExtractSymbols(testFile)
	if err != nil {
		t.Fatalf("ExtractSymbols failed: %v", err)
	}
	if fileSymbols == nil {
		t.Fatal("Expected fileSymbols, got nil")
	}

	if len(fileSymbols.Symbols) < 3 {
		t.Fatalf("Expected at least 3 symbols, got %d", len(fileSymbols.Symbols))
	}

	// fetch_user: async def
	fetchUser := fileSymbols.Symbols[0]
	if fetchUser.Name != "fetch_user" {
		t.Errorf("Expected 'fetch_user', got '%s'", fetchUser.Name)
	}
	if !strings.Contains(fetchUser.Signature, "async def") {
		t.Errorf("Expected signature to contain 'async def', got '%s'", fetchUser.Signature)
	}

	// sync_function: 通常の def
	syncFunc := fileSymbols.Symbols[2]
	if syncFunc.Name != "sync_function" {
		t.Errorf("Expected 'sync_function', got '%s'", syncFunc.Name)
	}
	if strings.Contains(syncFunc.Signature, "async") {
		t.Errorf("Expected signature NOT to contain 'async', got '%s'", syncFunc.Signature)
	}
	if !strings.HasPrefix(syncFunc.Signature, "def ") {
		t.Errorf("Expected signature to start with 'def ', got '%s'", syncFunc.Signature)
	}
}

// Issue #58: Python型注釈抽出テスト
func TestExtractPythonTypeAnnotations(t *testing.T) {
	content := `def process_data(items: list[str], options: dict[str, Any] = None) -> list[Result]:
    return [process(item) for item in items]
`
	tmpDir := t.TempDir()
	testutil.CreateTempFile(t, tmpDir, "types.py", content)
	testFile := filepath.Join(tmpDir, "types.py")

	fileSymbols, err := ExtractSymbols(testFile)
	if err != nil {
		t.Fatalf("ExtractSymbols failed: %v", err)
	}
	if fileSymbols == nil {
		t.Fatal("Expected fileSymbols, got nil")
	}

	if len(fileSymbols.Symbols) < 1 {
		t.Fatalf("Expected at least 1 symbol, got %d", len(fileSymbols.Symbols))
	}

	processData := fileSymbols.Symbols[0]
	// パラメータ型注釈
	if !strings.Contains(processData.Signature, "list[str]") {
		t.Errorf("Expected signature to contain 'list[str]', got '%s'", processData.Signature)
	}
	// 戻り値型注釈
	if !strings.Contains(processData.Signature, "-> list[Result]") {
		t.Errorf("Expected signature to contain '-> list[Result]', got '%s'", processData.Signature)
	}
}

func TestIsSupportedFile(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		{"test.go", true},
		{"test.js", true},
		{"test.ts", true},
		{"test.py", true},
		{"test.rs", true},
		{"test.java", true},
		{"test.rb", true},
		{"test.c", true},
		{"test.cpp", true},
		{"test.kt", true},
		{"test.swift", true},
		{"test.cs", true},
		{"test.scala", true},
		{"test.php", true},
		// Issue #59: CSS/SCSS
		{"test.css", true},
		{"test.scss", true},
		// Issue #60: 設定言語
		{"test.yaml", true},
		{"test.yml", true},
		{"test.toml", true},
		{"test.sql", true},
		{"test.sh", true},
		{"test.bash", true},
		{"test.md", true},
		{"test.txt", false},
		{"test.json", false},
	}

	for _, tt := range tests {
		result := IsSupportedFile(tt.path)
		if result != tt.expected {
			t.Errorf("IsSupportedFile(%s) = %v, expected %v", tt.path, result, tt.expected)
		}
	}
}

// ================== Issue #59: CSS/SCSS Tests ==================

func TestExtractCSSSelectors(t *testing.T) {
	content := `.button {
    color: red;
}

#header {
    background: blue;
}

body {
    margin: 0;
}

:root {
    --primary-color: #007bff;
    --secondary-color: #6c757d;
}
`
	tmpDir := t.TempDir()
	testutil.CreateTempFile(t, tmpDir, "style.css", content)
	testFile := filepath.Join(tmpDir, "style.css")

	fileSymbols, err := ExtractSymbols(testFile)
	if err != nil {
		t.Fatalf("ExtractSymbols failed: %v", err)
	}
	if fileSymbols == nil {
		t.Fatal("Expected fileSymbols, got nil")
	}

	// セレクタの数をチェック（少なくとも3つのセレクタ）
	if len(fileSymbols.Symbols) < 3 {
		t.Fatalf("Expected at least 3 CSS symbols, got %d", len(fileSymbols.Symbols))
	}

	// .button クラス
	found := false
	for _, sym := range fileSymbols.Symbols {
		if strings.Contains(sym.Name, ".button") {
			found = true
			if sym.Kind != "class" {
				t.Errorf("Expected kind 'class' for .button, got '%s'", sym.Kind)
			}
		}
	}
	if !found {
		t.Error("Expected to find .button selector")
	}
}

func TestExtractCSSKeyframes(t *testing.T) {
	content := `@keyframes fadeIn {
    from { opacity: 0; }
    to { opacity: 1; }
}

@keyframes slideUp {
    0% { transform: translateY(100%); }
    100% { transform: translateY(0); }
}
`
	tmpDir := t.TempDir()
	testutil.CreateTempFile(t, tmpDir, "animations.css", content)
	testFile := filepath.Join(tmpDir, "animations.css")

	fileSymbols, err := ExtractSymbols(testFile)
	if err != nil {
		t.Fatalf("ExtractSymbols failed: %v", err)
	}
	if fileSymbols == nil {
		t.Fatal("Expected fileSymbols, got nil")
	}

	// @keyframes が抽出されているか確認
	keyframesCount := 0
	for _, sym := range fileSymbols.Symbols {
		if sym.Kind == "keyframes" {
			keyframesCount++
		}
	}
	if keyframesCount < 2 {
		t.Errorf("Expected at least 2 keyframes, got %d", keyframesCount)
	}
}

func TestExtractCSSMediaQuery(t *testing.T) {
	content := `@media (max-width: 768px) {
    .container {
        width: 100%;
    }
}

@media screen and (min-width: 1200px) {
    .container {
        max-width: 1140px;
    }
}
`
	tmpDir := t.TempDir()
	testutil.CreateTempFile(t, tmpDir, "responsive.css", content)
	testFile := filepath.Join(tmpDir, "responsive.css")

	fileSymbols, err := ExtractSymbols(testFile)
	if err != nil {
		t.Fatalf("ExtractSymbols failed: %v", err)
	}
	if fileSymbols == nil {
		t.Fatal("Expected fileSymbols, got nil")
	}

	// @media クエリが抽出されているか確認
	mediaCount := 0
	for _, sym := range fileSymbols.Symbols {
		if sym.Kind == "media" {
			mediaCount++
		}
	}
	if mediaCount < 2 {
		t.Errorf("Expected at least 2 media queries, got %d", mediaCount)
	}
}

// ================== Issue #60: Configuration Languages Tests ==================

func TestExtractBashFunction(t *testing.T) {
	content := `#!/bin/bash

function hello() {
    echo "Hello, $1"
}

greet() {
    echo "Hi there!"
}

main() {
    hello "World"
    greet
}
`
	tmpDir := t.TempDir()
	testutil.CreateTempFile(t, tmpDir, "script.sh", content)
	testFile := filepath.Join(tmpDir, "script.sh")

	fileSymbols, err := ExtractSymbols(testFile)
	if err != nil {
		t.Fatalf("ExtractSymbols failed: %v", err)
	}
	if fileSymbols == nil {
		t.Fatal("Expected fileSymbols, got nil")
	}

	// 関数が抽出されているか確認
	if len(fileSymbols.Symbols) < 1 {
		t.Fatalf("Expected at least 1 Bash function, got %d", len(fileSymbols.Symbols))
	}

	// hello関数の検証
	found := false
	for _, sym := range fileSymbols.Symbols {
		if sym.Name == "hello" && sym.Kind == "function" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected to find 'hello' function")
	}
}

func TestExtractMakefileTargets(t *testing.T) {
	content := `BINARY=myapp

.PHONY: all clean test

all: build

build:
	go build -o $(BINARY) .

test:
	go test ./...

clean:
	rm -f $(BINARY)

install: build
	cp $(BINARY) /usr/local/bin/
`
	tmpDir := t.TempDir()
	testutil.CreateTempFile(t, tmpDir, "Makefile", content)
	testFile := filepath.Join(tmpDir, "Makefile")

	fileSymbols, err := ExtractSymbols(testFile)
	if err != nil {
		t.Fatalf("ExtractSymbols failed: %v", err)
	}
	if fileSymbols == nil {
		t.Fatal("Expected fileSymbols, got nil")
	}

	// ターゲットが抽出されているか確認（.PHONYは除外）
	if len(fileSymbols.Symbols) < 4 {
		t.Fatalf("Expected at least 4 Makefile targets, got %d", len(fileSymbols.Symbols))
	}

	// build, test, clean, install が含まれているか確認
	targets := make(map[string]bool)
	for _, sym := range fileSymbols.Symbols {
		targets[sym.Name] = true
	}

	expected := []string{"all", "build", "test", "clean", "install"}
	for _, name := range expected {
		if !targets[name] {
			t.Errorf("Expected to find target '%s'", name)
		}
	}
}

func TestExtractMarkdownHeadings(t *testing.T) {
	content := `# Main Title

Some introductory text.

## Getting Started

First steps here.

### Installation

Installation instructions.

## Usage

How to use the tool.

### Basic Commands

Command examples.
`
	tmpDir := t.TempDir()
	testutil.CreateTempFile(t, tmpDir, "README.md", content)
	testFile := filepath.Join(tmpDir, "README.md")

	fileSymbols, err := ExtractSymbols(testFile)
	if err != nil {
		t.Fatalf("ExtractSymbols failed: %v", err)
	}
	if fileSymbols == nil {
		t.Fatal("Expected fileSymbols, got nil")
	}

	// 見出しが抽出されているか確認
	if len(fileSymbols.Symbols) < 3 {
		t.Fatalf("Expected at least 3 Markdown headings, got %d", len(fileSymbols.Symbols))
	}

	// h1, h2, h3 が含まれているか確認
	kindCounts := make(map[string]int)
	for _, sym := range fileSymbols.Symbols {
		kindCounts[sym.Kind]++
	}

	if kindCounts["h1"] < 1 {
		t.Errorf("Expected at least 1 h1 heading, got %d", kindCounts["h1"])
	}
	if kindCounts["h2"] < 1 {
		t.Errorf("Expected at least 1 h2 heading, got %d", kindCounts["h2"])
	}
}

func TestExtractDockerfileInstructions(t *testing.T) {
	content := `FROM golang:1.21-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o main .

FROM alpine:latest

COPY --from=builder /app/main /main

CMD ["/main"]
`
	tmpDir := t.TempDir()
	testutil.CreateTempFile(t, tmpDir, "Dockerfile", content)
	testFile := filepath.Join(tmpDir, "Dockerfile")

	fileSymbols, err := ExtractSymbols(testFile)
	if err != nil {
		t.Fatalf("ExtractSymbols failed: %v", err)
	}
	if fileSymbols == nil {
		t.Fatal("Expected fileSymbols, got nil")
	}

	// FROM, RUN, CMD が抽出されているか確認
	kindCounts := make(map[string]int)
	for _, sym := range fileSymbols.Symbols {
		kindCounts[sym.Kind]++
	}

	if kindCounts["from"] < 2 {
		t.Errorf("Expected at least 2 FROM instructions, got %d", kindCounts["from"])
	}
	if kindCounts["run"] < 2 {
		t.Errorf("Expected at least 2 RUN instructions, got %d", kindCounts["run"])
	}
	if kindCounts["cmd"] < 1 {
		t.Errorf("Expected at least 1 CMD instruction, got %d", kindCounts["cmd"])
	}
}

// ================== Issue #62: Arrow Function Components / Hooks Tests ==================

func TestExtractArrowFunctionComponent(t *testing.T) {
	// Note: JSX構文はTypeScriptパーサーで正しく解析されないため、
	// JSXなしの形式でテスト
	content := `const Header = () => {
    return null;
};

const Button: React.FC<Props> = ({ onClick }) => {
    return null;
};

const App = () => {
    return "Hello";
};
`
	tmpDir := t.TempDir()
	testutil.CreateTempFile(t, tmpDir, "components.ts", content)
	testFile := filepath.Join(tmpDir, "components.ts")

	fileSymbols, err := ExtractSymbols(testFile)
	if err != nil {
		t.Fatalf("ExtractSymbols failed: %v", err)
	}
	if fileSymbols == nil {
		t.Fatal("Expected fileSymbols, got nil")
	}

	// コンポーネントを探す
	components := make(map[string]Symbol)
	for _, sym := range fileSymbols.Symbols {
		if sym.Kind == "component" {
			components[sym.Name] = sym
		}
	}

	// Header コンポーネント
	if _, ok := components["Header"]; !ok {
		t.Error("Expected to find 'Header' component")
	}

	// Button コンポーネント（型注釈付き）
	button, ok := components["Button"]
	if !ok {
		t.Error("Expected to find 'Button' component")
	} else {
		if !strings.Contains(button.Signature, "React.FC<Props>") {
			t.Errorf("Expected Button signature to contain 'React.FC<Props>', got '%s'", button.Signature)
		}
	}

	// App コンポーネント
	if _, ok := components["App"]; !ok {
		t.Error("Expected to find 'App' component")
	}
}

func TestExtractArrowFunctionHook(t *testing.T) {
	content := `const useAuth = () => {
    const [user, setUser] = useState(null);
    return { user, setUser };
};

const useCounter = (initial: number) => {
    const [count, setCount] = useState(initial);
    return { count, increment: () => setCount(c => c + 1) };
};
`
	tmpDir := t.TempDir()
	testutil.CreateTempFile(t, tmpDir, "hooks.tsx", content)
	testFile := filepath.Join(tmpDir, "hooks.tsx")

	fileSymbols, err := ExtractSymbols(testFile)
	if err != nil {
		t.Fatalf("ExtractSymbols failed: %v", err)
	}
	if fileSymbols == nil {
		t.Fatal("Expected fileSymbols, got nil")
	}

	// Hookを探す
	hooks := make(map[string]Symbol)
	for _, sym := range fileSymbols.Symbols {
		if sym.Kind == "hook" {
			hooks[sym.Name] = sym
		}
	}

	// useAuth Hook
	if _, ok := hooks["useAuth"]; !ok {
		t.Error("Expected to find 'useAuth' hook")
	}

	// useCounter Hook
	useCounter, ok := hooks["useCounter"]
	if !ok {
		t.Error("Expected to find 'useCounter' hook")
	} else {
		if !strings.Contains(useCounter.Signature, "(initial: number)") {
			t.Errorf("Expected useCounter signature to contain '(initial: number)', got '%s'", useCounter.Signature)
		}
	}
}

func TestExtractFunctionDeclarationHook(t *testing.T) {
	content := `function useAuth() {
    const [user, setUser] = useState(null);
    return { user, setUser };
}

function useCounter(initial) {
    const [count, setCount] = useState(initial);
    return { count };
}

function App() {
    return <div>Hello</div>;
}
`
	tmpDir := t.TempDir()
	testutil.CreateTempFile(t, tmpDir, "hooks.js", content)
	testFile := filepath.Join(tmpDir, "hooks.js")

	fileSymbols, err := ExtractSymbols(testFile)
	if err != nil {
		t.Fatalf("ExtractSymbols failed: %v", err)
	}
	if fileSymbols == nil {
		t.Fatal("Expected fileSymbols, got nil")
	}

	// シンボルのKindを確認
	kindMap := make(map[string]string)
	for _, sym := range fileSymbols.Symbols {
		kindMap[sym.Name] = sym.Kind
	}

	// useAuth は hook
	if kindMap["useAuth"] != "hook" {
		t.Errorf("Expected useAuth to be 'hook', got '%s'", kindMap["useAuth"])
	}

	// useCounter は hook
	if kindMap["useCounter"] != "hook" {
		t.Errorf("Expected useCounter to be 'hook', got '%s'", kindMap["useCounter"])
	}

	// App は function（PascalCaseだがfunction宣言なのでfunctionのまま）
	if kindMap["App"] != "function" {
		t.Errorf("Expected App to be 'function', got '%s'", kindMap["App"])
	}
}

func TestArrowFunctionSkipsRegularVariables(t *testing.T) {
	content := `const handler = () => {
    console.log("clicked");
};

const callback = (x) => x * 2;

const Header = () => {
    return null;
};
`
	tmpDir := t.TempDir()
	testutil.CreateTempFile(t, tmpDir, "mixed.ts", content)
	testFile := filepath.Join(tmpDir, "mixed.ts")

	fileSymbols, err := ExtractSymbols(testFile)
	if err != nil {
		t.Fatalf("ExtractSymbols failed: %v", err)
	}
	if fileSymbols == nil {
		t.Fatal("Expected fileSymbols, got nil")
	}

	// 名前でシンボルをマップ
	symbolMap := make(map[string]Symbol)
	for _, sym := range fileSymbols.Symbols {
		symbolMap[sym.Name] = sym
	}

	// handler と callback はスキップされるべき（小文字始まり、hookでもない）
	if _, ok := symbolMap["handler"]; ok {
		t.Error("Expected 'handler' to be skipped (not a component or hook)")
	}
	if _, ok := symbolMap["callback"]; ok {
		t.Error("Expected 'callback' to be skipped (not a component or hook)")
	}

	// Header はコンポーネントとして抽出される
	if sym, ok := symbolMap["Header"]; !ok {
		t.Error("Expected to find 'Header' component")
	} else if sym.Kind != "component" {
		t.Errorf("Expected Header kind to be 'component', got '%s'", sym.Kind)
	}
}

// ================== Issue #63: Vue/Svelte SFC Tests ==================

func TestExtractVueSFC(t *testing.T) {
	content := `<template>
  <div class="header">
    <h1>{{ title }}</h1>
  </div>
</template>

<script>
export default {
  name: 'Header',
  data() {
    return {
      title: 'Hello Vue'
    }
  },
  methods: {
    greet() {
      console.log('Hello!')
    }
  }
}
</script>

<style scoped>
.header {
  background: blue;
}
h1 {
  color: white;
}
</style>
`
	tmpDir := t.TempDir()
	testutil.CreateTempFile(t, tmpDir, "Header.vue", content)
	testFile := filepath.Join(tmpDir, "Header.vue")

	fileSymbols, err := ExtractSymbols(testFile)
	if err != nil {
		t.Fatalf("ExtractSymbols failed: %v", err)
	}
	if fileSymbols == nil {
		t.Fatal("Expected fileSymbols, got nil")
	}

	// ファイルパスがVueファイルであることを確認
	if !strings.HasSuffix(fileSymbols.Path, ".vue") {
		t.Errorf("Expected path to end with .vue, got '%s'", fileSymbols.Path)
	}

	// 少なくともいくつかのシンボルが抽出されているか確認
	if len(fileSymbols.Symbols) == 0 {
		t.Error("Expected some symbols to be extracted from Vue SFC")
	}

	// CSSセレクタが抽出されているか確認
	var cssSelectors []Symbol
	for _, sym := range fileSymbols.Symbols {
		if sym.Kind == "class" || sym.Kind == "selector" {
			cssSelectors = append(cssSelectors, sym)
		}
	}

	if len(cssSelectors) < 1 {
		t.Errorf("Expected at least 1 CSS selector, got %d", len(cssSelectors))
	}
}

func TestExtractVueSFCWithTypeScript(t *testing.T) {
	content := `<template>
  <button @click="increment">{{ count }}</button>
</template>

<script lang="ts">
import { ref } from 'vue'

interface Props {
  initialCount: number
}

function useCounter(initial: number) {
  const count = ref(initial)
  const increment = () => count.value++
  return { count, increment }
}

export default {
  setup(props: Props) {
    const { count, increment } = useCounter(props.initialCount)
    return { count, increment }
  }
}
</script>

<style>
button {
  padding: 10px;
}
</style>
`
	tmpDir := t.TempDir()
	testutil.CreateTempFile(t, tmpDir, "Counter.vue", content)
	testFile := filepath.Join(tmpDir, "Counter.vue")

	fileSymbols, err := ExtractSymbols(testFile)
	if err != nil {
		t.Fatalf("ExtractSymbols failed: %v", err)
	}
	if fileSymbols == nil {
		t.Fatal("Expected fileSymbols, got nil")
	}

	// useCounter関数が抽出されているか確認
	var foundUseCounter bool
	for _, sym := range fileSymbols.Symbols {
		if sym.Name == "useCounter" {
			foundUseCounter = true
			// hook として認識されるか
			if sym.Kind != "hook" {
				t.Errorf("Expected useCounter to be 'hook', got '%s'", sym.Kind)
			}
			// 型注釈が含まれているか
			if !strings.Contains(sym.Signature, "number") {
				t.Errorf("Expected signature to contain type annotation, got '%s'", sym.Signature)
			}
			break
		}
	}

	if !foundUseCounter {
		t.Error("Expected to find 'useCounter' function")
	}
}

func TestExtractSvelteSFC(t *testing.T) {
	content := `<script>
  let count = 0;

  function increment() {
    count += 1;
  }

  function decrement() {
    count -= 1;
  }
</script>

<main>
  <h1>Counter: {count}</h1>
  <button on:click={increment}>+</button>
  <button on:click={decrement}>-</button>
</main>

<style>
  main {
    text-align: center;
  }

  button {
    font-size: 2rem;
    margin: 0.5rem;
  }
</style>
`
	tmpDir := t.TempDir()
	testutil.CreateTempFile(t, tmpDir, "Counter.svelte", content)
	testFile := filepath.Join(tmpDir, "Counter.svelte")

	fileSymbols, err := ExtractSymbols(testFile)
	if err != nil {
		t.Fatalf("ExtractSymbols failed: %v", err)
	}
	if fileSymbols == nil {
		t.Fatal("Expected fileSymbols, got nil")
	}

	// 関数が抽出されているか確認
	functionMap := make(map[string]Symbol)
	for _, sym := range fileSymbols.Symbols {
		if sym.Kind == "function" {
			functionMap[sym.Name] = sym
		}
	}

	// increment 関数
	if _, ok := functionMap["increment"]; !ok {
		t.Error("Expected to find 'increment' function")
	}

	// decrement 関数
	if _, ok := functionMap["decrement"]; !ok {
		t.Error("Expected to find 'decrement' function")
	}

	// CSSセレクタ
	var cssCount int
	for _, sym := range fileSymbols.Symbols {
		if sym.Kind == "selector" || sym.Kind == "class" {
			cssCount++
		}
	}
	if cssCount < 1 {
		t.Errorf("Expected at least 1 CSS selector, got %d", cssCount)
	}
}

func TestExtractSvelteSFCWithTypeScript(t *testing.T) {
	content := `<script lang="ts">
  interface User {
    name: string;
    age: number;
  }

  let users: User[] = [];

  function addUser(name: string, age: number): void {
    users = [...users, { name, age }];
  }

  function removeUser(index: number): void {
    users = users.filter((_, i) => i !== index);
  }
</script>

<ul>
  {#each users as user, i}
    <li>{user.name} ({user.age}) <button on:click={() => removeUser(i)}>Remove</button></li>
  {/each}
</ul>
`
	tmpDir := t.TempDir()
	testutil.CreateTempFile(t, tmpDir, "UserList.svelte", content)
	testFile := filepath.Join(tmpDir, "UserList.svelte")

	fileSymbols, err := ExtractSymbols(testFile)
	if err != nil {
		t.Fatalf("ExtractSymbols failed: %v", err)
	}
	if fileSymbols == nil {
		t.Fatal("Expected fileSymbols, got nil")
	}

	// 関数を確認
	functionMap := make(map[string]Symbol)
	for _, sym := range fileSymbols.Symbols {
		if sym.Kind == "function" {
			functionMap[sym.Name] = sym
		}
	}

	// addUser関数の型注釈確認
	if sym, ok := functionMap["addUser"]; !ok {
		t.Error("Expected to find 'addUser' function")
	} else {
		if !strings.Contains(sym.Signature, "string") {
			t.Errorf("Expected addUser signature to contain type, got '%s'", sym.Signature)
		}
	}

	// removeUser関数
	if _, ok := functionMap["removeUser"]; !ok {
		t.Error("Expected to find 'removeUser' function")
	}
}

func TestIsSFCFile(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		{"Header.vue", true},
		{"Counter.svelte", true},
		{"Component.VUE", true}, // 大文字でも対応
		{"App.SVELTE", true},    // 大文字でも対応
		{"main.js", false},
		{"style.css", false},
		{"App.jsx", false},
	}

	for _, tt := range tests {
		result := IsSFCFile(tt.path)
		if result != tt.expected {
			t.Errorf("IsSFCFile(%s) = %v, expected %v", tt.path, result, tt.expected)
		}
	}
}

func TestSFCLineNumberOffset(t *testing.T) {
	// 行番号が正しくオフセットされているか確認
	content := `<template>
  <div>Hello</div>
</template>

<script>
function myFunction() {
  return 42;
}
</script>
`
	tmpDir := t.TempDir()
	testutil.CreateTempFile(t, tmpDir, "LineTest.vue", content)
	testFile := filepath.Join(tmpDir, "LineTest.vue")

	fileSymbols, err := ExtractSymbols(testFile)
	if err != nil {
		t.Fatalf("ExtractSymbols failed: %v", err)
	}
	if fileSymbols == nil {
		t.Fatal("Expected fileSymbols, got nil")
	}

	// myFunction の行番号を確認
	for _, sym := range fileSymbols.Symbols {
		if sym.Name == "myFunction" {
			// <script> タグは5行目、関数は6行目
			if sym.Line < 6 {
				t.Errorf("Expected myFunction line to be >= 6, got %d", sym.Line)
			}
			break
		}
	}
}

// ================== Issue #64: HTML id/class属性抽出 Tests ==================

func TestExtractHTMLIdAttribute(t *testing.T) {
	content := `<!DOCTYPE html>
<html>
<head><title>Test</title></head>
<body>
  <div id="main">
    <h1 id="title">Hello</h1>
  </div>
</body>
</html>
`
	tmpDir := t.TempDir()
	testutil.CreateTempFile(t, tmpDir, "test.html", content)
	testFile := filepath.Join(tmpDir, "test.html")

	fileSymbols, err := ExtractSymbols(testFile)
	if err != nil {
		t.Fatalf("ExtractSymbols failed: %v", err)
	}
	if fileSymbols == nil {
		t.Fatal("Expected fileSymbols, got nil")
	}

	// id属性を探す
	ids := make(map[string]Symbol)
	for _, sym := range fileSymbols.Symbols {
		if sym.Kind == "id" {
			ids[sym.Name] = sym
		}
	}

	// #main
	if _, ok := ids["#main"]; !ok {
		t.Error("Expected to find '#main' id")
	}

	// #title
	if sym, ok := ids["#title"]; !ok {
		t.Error("Expected to find '#title' id")
	} else {
		if !strings.Contains(sym.Signature, "h1") {
			t.Errorf("Expected signature to contain 'h1', got '%s'", sym.Signature)
		}
	}
}

func TestExtractHTMLClassAttribute(t *testing.T) {
	content := `<!DOCTYPE html>
<html>
<body>
  <div class="container">
    <header class="header primary">
      <nav class="nav-bar">Links</nav>
    </header>
  </div>
</body>
</html>
`
	tmpDir := t.TempDir()
	testutil.CreateTempFile(t, tmpDir, "test.html", content)
	testFile := filepath.Join(tmpDir, "test.html")

	fileSymbols, err := ExtractSymbols(testFile)
	if err != nil {
		t.Fatalf("ExtractSymbols failed: %v", err)
	}
	if fileSymbols == nil {
		t.Fatal("Expected fileSymbols, got nil")
	}

	// class属性を探す
	classes := make(map[string]Symbol)
	for _, sym := range fileSymbols.Symbols {
		if sym.Kind == "class" {
			classes[sym.Name] = sym
		}
	}

	// .container
	if _, ok := classes[".container"]; !ok {
		t.Error("Expected to find '.container' class")
	}

	// .header（複数クラスの1つ目）
	if _, ok := classes[".header"]; !ok {
		t.Error("Expected to find '.header' class")
	}

	// .primary（複数クラスの2つ目）
	if _, ok := classes[".primary"]; !ok {
		t.Error("Expected to find '.primary' class (from 'header primary')")
	}

	// .nav-bar
	if _, ok := classes[".nav-bar"]; !ok {
		t.Error("Expected to find '.nav-bar' class")
	}
}

func TestExtractHTMLSemanticElements(t *testing.T) {
	content := `<!DOCTYPE html>
<html>
<body>
  <header>
    <nav>Navigation</nav>
  </header>
  <main>
    <article>Content</article>
  </main>
  <footer>Footer</footer>
</body>
</html>
`
	tmpDir := t.TempDir()
	testutil.CreateTempFile(t, tmpDir, "semantic.html", content)
	testFile := filepath.Join(tmpDir, "semantic.html")

	fileSymbols, err := ExtractSymbols(testFile)
	if err != nil {
		t.Fatalf("ExtractSymbols failed: %v", err)
	}
	if fileSymbols == nil {
		t.Fatal("Expected fileSymbols, got nil")
	}

	// セマンティック要素を探す
	semantics := make(map[string]Symbol)
	for _, sym := range fileSymbols.Symbols {
		if sym.Kind == "semantic" {
			semantics[sym.Name] = sym
		}
	}

	// <header>
	if _, ok := semantics["<header>"]; !ok {
		t.Error("Expected to find '<header>' semantic element")
	}

	// <nav>
	if _, ok := semantics["<nav>"]; !ok {
		t.Error("Expected to find '<nav>' semantic element")
	}

	// <main>
	if _, ok := semantics["<main>"]; !ok {
		t.Error("Expected to find '<main>' semantic element")
	}

	// <article>
	if _, ok := semantics["<article>"]; !ok {
		t.Error("Expected to find '<article>' semantic element")
	}

	// <footer>
	if _, ok := semantics["<footer>"]; !ok {
		t.Error("Expected to find '<footer>' semantic element")
	}
}

func TestExtractHTMLMixedAttributes(t *testing.T) {
	content := `<!DOCTYPE html>
<html>
<body>
  <div id="app" class="container wrapper">
    <section id="content" class="main-section">
      <p>Text</p>
    </section>
  </div>
</body>
</html>
`
	tmpDir := t.TempDir()
	testutil.CreateTempFile(t, tmpDir, "mixed.html", content)
	testFile := filepath.Join(tmpDir, "mixed.html")

	fileSymbols, err := ExtractSymbols(testFile)
	if err != nil {
		t.Fatalf("ExtractSymbols failed: %v", err)
	}
	if fileSymbols == nil {
		t.Fatal("Expected fileSymbols, got nil")
	}

	// シンボルをタイプ別に集計
	ids := make(map[string]bool)
	classes := make(map[string]bool)
	for _, sym := range fileSymbols.Symbols {
		switch sym.Kind {
		case "id":
			ids[sym.Name] = true
		case "class":
			classes[sym.Name] = true
		}
	}

	// id属性
	expectedIds := []string{"#app", "#content"}
	for _, id := range expectedIds {
		if !ids[id] {
			t.Errorf("Expected to find '%s' id", id)
		}
	}

	// class属性（複数クラスが個別に抽出される）
	expectedClasses := []string{".container", ".wrapper", ".main-section"}
	for _, class := range expectedClasses {
		if !classes[class] {
			t.Errorf("Expected to find '%s' class", class)
		}
	}
}

func TestExtractHTMLHtmExtension(t *testing.T) {
	content := `<!DOCTYPE html>
<html>
<body>
  <div id="legacy">Old page</div>
</body>
</html>
`
	tmpDir := t.TempDir()
	testutil.CreateTempFile(t, tmpDir, "page.htm", content)
	testFile := filepath.Join(tmpDir, "page.htm")

	fileSymbols, err := ExtractSymbols(testFile)
	if err != nil {
		t.Fatalf("ExtractSymbols failed: %v", err)
	}
	if fileSymbols == nil {
		t.Fatal("Expected fileSymbols, got nil")
	}

	// .htm 拡張子でもid抽出が動作するか
	found := false
	for _, sym := range fileSymbols.Symbols {
		if sym.Name == "#legacy" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected to find '#legacy' id from .htm file")
	}
}

func TestHTMLIsSupportedFile(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		{"index.html", true},
		{"page.htm", true},
		{"INDEX.HTML", true}, // 大文字でも対応
		{"main.js", true},    // JSはサポート
		{"style.css", true},  // CSSはサポート
		{"test.txt", false},  // txtはサポート外
	}

	for _, tt := range tests {
		result := IsSupportedFile(tt.path)
		if result != tt.expected {
			t.Errorf("IsSupportedFile(%s) = %v, expected %v", tt.path, result, tt.expected)
		}
	}
}

// ================== Issue #65: Tailwind CSS config Tests ==================

func TestIsTailwindConfigFile(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		{"tailwind.config.js", true},
		{"tailwind.config.ts", true},
		{"tailwind.config.mjs", true},
		{"tailwind.config.cjs", true},
		{"tailwind.config.mts", true},
		{"tailwind.css", false},    // 設定ファイルではない
		{"config.js", false},       // tailwind.で始まらない
		{"other.config.js", false}, // tailwindではない
		{"main.js", false},
	}

	for _, tt := range tests {
		result := IsTailwindConfigFile(tt.path)
		if result != tt.expected {
			t.Errorf("IsTailwindConfigFile(%s) = %v, expected %v", tt.path, result, tt.expected)
		}
	}
}

func TestExtractTailwindConfigThemeExtend(t *testing.T) {
	content := `module.exports = {
  content: ['./src/**/*.{js,ts,jsx,tsx}'],
  theme: {
    extend: {
      colors: {
        primary: '#3490dc',
        secondary: '#ffed4a',
        danger: '#e3342f',
      },
      spacing: {
        '72': '18rem',
        '84': '21rem',
      },
    },
  },
  plugins: [],
}
`
	tmpDir := t.TempDir()
	testutil.CreateTempFile(t, tmpDir, "tailwind.config.js", content)
	testFile := filepath.Join(tmpDir, "tailwind.config.js")

	fileSymbols, err := ExtractSymbols(testFile)
	if err != nil {
		t.Fatalf("ExtractSymbols failed: %v", err)
	}
	if fileSymbols == nil {
		t.Fatal("Expected fileSymbols, got nil")
	}

	// シンボルをマップに集約
	symbolMap := make(map[string]Symbol)
	for _, sym := range fileSymbols.Symbols {
		symbolMap[sym.Name] = sym
	}

	// content 設定
	if _, ok := symbolMap["content"]; !ok {
		t.Error("Expected to find 'content' config")
	}

	// theme.extend.colors
	if sym, ok := symbolMap["theme.extend.colors"]; !ok {
		t.Error("Expected to find 'theme.extend.colors'")
	} else {
		if sym.Kind != "theme" {
			t.Errorf("Expected kind 'theme', got '%s'", sym.Kind)
		}
		if !strings.Contains(sym.Signature, "items") {
			t.Errorf("Expected signature to contain item count, got '%s'", sym.Signature)
		}
	}

	// theme.extend.spacing
	if _, ok := symbolMap["theme.extend.spacing"]; !ok {
		t.Error("Expected to find 'theme.extend.spacing'")
	}
}

func TestExtractTailwindConfigPlugins(t *testing.T) {
	content := `module.exports = {
  content: ['./src/**/*.{js,ts}'],
  plugins: [
    require('@tailwindcss/forms'),
    require('@tailwindcss/typography'),
    require('@tailwindcss/aspect-ratio'),
  ],
}
`
	tmpDir := t.TempDir()
	testutil.CreateTempFile(t, tmpDir, "tailwind.config.js", content)
	testFile := filepath.Join(tmpDir, "tailwind.config.js")

	fileSymbols, err := ExtractSymbols(testFile)
	if err != nil {
		t.Fatalf("ExtractSymbols failed: %v", err)
	}
	if fileSymbols == nil {
		t.Fatal("Expected fileSymbols, got nil")
	}

	// プラグインを探す
	plugins := make(map[string]Symbol)
	for _, sym := range fileSymbols.Symbols {
		if sym.Kind == "plugin" {
			plugins[sym.Name] = sym
		}
	}

	// @tailwindcss/forms
	if _, ok := plugins["@tailwindcss/forms"]; !ok {
		t.Error("Expected to find '@tailwindcss/forms' plugin")
	}

	// @tailwindcss/typography
	if _, ok := plugins["@tailwindcss/typography"]; !ok {
		t.Error("Expected to find '@tailwindcss/typography' plugin")
	}

	// @tailwindcss/aspect-ratio
	if _, ok := plugins["@tailwindcss/aspect-ratio"]; !ok {
		t.Error("Expected to find '@tailwindcss/aspect-ratio' plugin")
	}
}

func TestExtractTailwindConfigTypeScript(t *testing.T) {
	content := `import type { Config } from 'tailwindcss'

export default {
  content: ['./src/**/*.{js,ts,jsx,tsx}'],
  theme: {
    extend: {
      colors: {
        brand: {
          light: '#3fbaeb',
          DEFAULT: '#0fa9e6',
          dark: '#0c87b8',
        },
      },
      fontFamily: {
        sans: ['Inter', 'sans-serif'],
        serif: ['Georgia', 'serif'],
      },
    },
  },
  plugins: [],
} satisfies Config
`
	tmpDir := t.TempDir()
	testutil.CreateTempFile(t, tmpDir, "tailwind.config.ts", content)
	testFile := filepath.Join(tmpDir, "tailwind.config.ts")

	fileSymbols, err := ExtractSymbols(testFile)
	if err != nil {
		t.Fatalf("ExtractSymbols failed: %v", err)
	}
	if fileSymbols == nil {
		t.Fatal("Expected fileSymbols, got nil")
	}

	// シンボルをマップに集約
	symbolMap := make(map[string]Symbol)
	for _, sym := range fileSymbols.Symbols {
		symbolMap[sym.Name] = sym
	}

	// theme.extend.colors
	if _, ok := symbolMap["theme.extend.colors"]; !ok {
		t.Error("Expected to find 'theme.extend.colors' in TypeScript config")
	}

	// theme.extend.fontFamily
	if _, ok := symbolMap["theme.extend.fontFamily"]; !ok {
		t.Error("Expected to find 'theme.extend.fontFamily' in TypeScript config")
	}
}

func TestExtractTailwindConfigModernFormat(t *testing.T) {
	// Tailwind v3+ ESModule形式
	content := `export default {
  content: ['./index.html', './src/**/*.{vue,js,ts}'],
  theme: {
    screens: {
      sm: '640px',
      md: '768px',
      lg: '1024px',
    },
    extend: {
      animation: {
        'spin-slow': 'spin 3s linear infinite',
      },
      keyframes: {
        wiggle: {
          '0%, 100%': { transform: 'rotate(-3deg)' },
          '50%': { transform: 'rotate(3deg)' },
        },
      },
    },
  },
}
`
	tmpDir := t.TempDir()
	testutil.CreateTempFile(t, tmpDir, "tailwind.config.mjs", content)
	testFile := filepath.Join(tmpDir, "tailwind.config.mjs")

	fileSymbols, err := ExtractSymbols(testFile)
	if err != nil {
		t.Fatalf("ExtractSymbols failed: %v", err)
	}
	if fileSymbols == nil {
		t.Fatal("Expected fileSymbols, got nil")
	}

	// シンボルをマップに集約
	symbolMap := make(map[string]Symbol)
	for _, sym := range fileSymbols.Symbols {
		symbolMap[sym.Name] = sym
	}

	// theme.screens（extendの外）
	if _, ok := symbolMap["theme.screens"]; !ok {
		t.Error("Expected to find 'theme.screens'")
	}

	// theme.extend.animation
	if _, ok := symbolMap["theme.extend.animation"]; !ok {
		t.Error("Expected to find 'theme.extend.animation'")
	}

	// theme.extend.keyframes
	if _, ok := symbolMap["theme.extend.keyframes"]; !ok {
		t.Error("Expected to find 'theme.extend.keyframes'")
	}
}

func TestExtractTailwindConfigEmptyPlugins(t *testing.T) {
	content := `module.exports = {
  content: ['./src/**/*.html'],
  plugins: [],
}
`
	tmpDir := t.TempDir()
	testutil.CreateTempFile(t, tmpDir, "tailwind.config.js", content)
	testFile := filepath.Join(tmpDir, "tailwind.config.js")

	fileSymbols, err := ExtractSymbols(testFile)
	if err != nil {
		t.Fatalf("ExtractSymbols failed: %v", err)
	}
	if fileSymbols == nil {
		t.Fatal("Expected fileSymbols, got nil")
	}

	// 空のプラグイン配列ではプラグインシンボルは抽出されない
	pluginCount := 0
	for _, sym := range fileSymbols.Symbols {
		if sym.Kind == "plugin" {
			pluginCount++
		}
	}

	if pluginCount != 0 {
		t.Errorf("Expected 0 plugins for empty array, got %d", pluginCount)
	}

	// content は抽出される
	found := false
	for _, sym := range fileSymbols.Symbols {
		if sym.Name == "content" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected to find 'content' config")
	}
}

// ================== Elixir Tests ==================

func TestExtractElixirModule(t *testing.T) {
	content := `defmodule MyApp.Users do
  @moduledoc "User management"

  def get_user(id) do
    Repo.get(User, id)
  end

  defp validate(user) do
    User.changeset(user)
  end
end
`
	tmpDir := t.TempDir()
	testutil.CreateTempFile(t, tmpDir, "users.ex", content)
	testFile := filepath.Join(tmpDir, "users.ex")

	fileSymbols, err := ExtractSymbols(testFile)
	if err != nil {
		t.Fatalf("ExtractSymbols failed: %v", err)
	}
	if fileSymbols == nil {
		t.Fatal("Expected fileSymbols, got nil")
	}

	// シンボルをマップに集約
	symbolMap := make(map[string]Symbol)
	for _, sym := range fileSymbols.Symbols {
		symbolMap[sym.Name] = sym
	}

	// defmodule MyApp.Users
	if sym, ok := symbolMap["MyApp.Users"]; !ok {
		t.Error("Expected to find 'MyApp.Users' module")
	} else {
		if sym.Kind != "module" {
			t.Errorf("Expected kind 'module', got '%s'", sym.Kind)
		}
		if !strings.Contains(sym.Signature, "defmodule") {
			t.Errorf("Expected signature to contain 'defmodule', got '%s'", sym.Signature)
		}
	}

	// def get_user
	if sym, ok := symbolMap["get_user"]; !ok {
		t.Error("Expected to find 'get_user' function")
	} else {
		if sym.Kind != "function" {
			t.Errorf("Expected kind 'function', got '%s'", sym.Kind)
		}
		if !strings.Contains(sym.Signature, "def get_user") {
			t.Errorf("Expected signature to contain 'def get_user', got '%s'", sym.Signature)
		}
	}

	// defp validate（プライベート関数）
	if sym, ok := symbolMap["validate"]; !ok {
		t.Error("Expected to find 'validate' private function")
	} else {
		if sym.Kind != "private_function" {
			t.Errorf("Expected kind 'private_function', got '%s'", sym.Kind)
		}
		if !strings.Contains(sym.Signature, "defp") {
			t.Errorf("Expected signature to contain 'defp', got '%s'", sym.Signature)
		}
	}
}

func TestExtractElixirPhoenixController(t *testing.T) {
	content := `defmodule MyAppWeb.UserController do
  use MyAppWeb, :controller

  def index(conn, _params) do
    users = Users.list_users()
    render(conn, "index.html", users: users)
  end

  def show(conn, %{"id" => id}) do
    user = Users.get_user!(id)
    render(conn, "show.html", user: user)
  end

  def create(conn, %{"user" => user_params}) do
    case Users.create_user(user_params) do
      {:ok, user} ->
        redirect(conn, to: Routes.user_path(conn, :show, user))
      {:error, changeset} ->
        render(conn, "new.html", changeset: changeset)
    end
  end
end
`
	tmpDir := t.TempDir()
	testutil.CreateTempFile(t, tmpDir, "user_controller.ex", content)
	testFile := filepath.Join(tmpDir, "user_controller.ex")

	fileSymbols, err := ExtractSymbols(testFile)
	if err != nil {
		t.Fatalf("ExtractSymbols failed: %v", err)
	}
	if fileSymbols == nil {
		t.Fatal("Expected fileSymbols, got nil")
	}

	// 関数を探す
	functions := make(map[string]Symbol)
	for _, sym := range fileSymbols.Symbols {
		if sym.Kind == "function" {
			functions[sym.Name] = sym
		}
	}

	// index, show, create が抽出されるか
	expectedFuncs := []string{"index", "show", "create"}
	for _, name := range expectedFuncs {
		if _, ok := functions[name]; !ok {
			t.Errorf("Expected to find '%s' function", name)
		}
	}
}

func TestExtractElixirExsScript(t *testing.T) {
	content := `defmodule Mix.Tasks.MyTask do
  use Mix.Task

  def run(args) do
    IO.puts("Running task with args: #{inspect(args)}")
  end
end
`
	tmpDir := t.TempDir()
	testutil.CreateTempFile(t, tmpDir, "my_task.exs", content)
	testFile := filepath.Join(tmpDir, "my_task.exs")

	fileSymbols, err := ExtractSymbols(testFile)
	if err != nil {
		t.Fatalf("ExtractSymbols failed: %v", err)
	}
	if fileSymbols == nil {
		t.Fatal("Expected fileSymbols, got nil")
	}

	// .exs ファイルでもモジュールと関数が抽出されるか
	var foundModule, foundFunc bool
	for _, sym := range fileSymbols.Symbols {
		if sym.Name == "Mix.Tasks.MyTask" && sym.Kind == "module" {
			foundModule = true
		}
		if sym.Name == "run" && sym.Kind == "function" {
			foundFunc = true
		}
	}

	if !foundModule {
		t.Error("Expected to find 'Mix.Tasks.MyTask' module in .exs file")
	}
	if !foundFunc {
		t.Error("Expected to find 'run' function in .exs file")
	}
}

func TestElixirIsSupportedFile(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		{"lib/my_app.ex", true},
		{"test/my_app_test.exs", true},
		{"mix.exs", true},
		{"config/config.exs", true},
		{"main.js", true},   // JSはサポート
		{"test.txt", false}, // txtはサポート外
	}

	for _, tt := range tests {
		result := IsSupportedFile(tt.path)
		if result != tt.expected {
			t.Errorf("IsSupportedFile(%s) = %v, expected %v", tt.path, result, tt.expected)
		}
	}
}

// ================== Lua Tests ==================

func TestExtractLuaLocalFunction(t *testing.T) {
	content := `local function greet(name)
    print("Hello, " .. name)
end

local function add(a, b)
    return a + b
end
`
	tmpDir := t.TempDir()
	testutil.CreateTempFile(t, tmpDir, "utils.lua", content)
	testFile := filepath.Join(tmpDir, "utils.lua")

	fileSymbols, err := ExtractSymbols(testFile)
	if err != nil {
		t.Fatalf("ExtractSymbols failed: %v", err)
	}
	if fileSymbols == nil {
		t.Fatal("Expected fileSymbols, got nil")
	}

	// シンボルをマップに集約
	symbolMap := make(map[string]Symbol)
	for _, sym := range fileSymbols.Symbols {
		symbolMap[sym.Name] = sym
	}

	// local function greet
	if sym, ok := symbolMap["greet"]; !ok {
		t.Error("Expected to find 'greet' function")
	} else {
		if sym.Kind != "function" {
			t.Errorf("Expected kind 'function', got '%s'", sym.Kind)
		}
		if !strings.Contains(sym.Signature, "local") {
			t.Errorf("Expected signature to contain 'local', got '%s'", sym.Signature)
		}
		if !strings.Contains(sym.Signature, "(name)") {
			t.Errorf("Expected signature to contain '(name)', got '%s'", sym.Signature)
		}
	}

	// local function add
	if sym, ok := symbolMap["add"]; !ok {
		t.Error("Expected to find 'add' function")
	} else {
		if !strings.Contains(sym.Signature, "(a, b)") {
			t.Errorf("Expected signature to contain '(a, b)', got '%s'", sym.Signature)
		}
	}
}

func TestExtractLuaGlobalFunction(t *testing.T) {
	content := `function globalFunc()
    return 42
end

function calculate(x, y, z)
    return x + y + z
end
`
	tmpDir := t.TempDir()
	testutil.CreateTempFile(t, tmpDir, "global.lua", content)
	testFile := filepath.Join(tmpDir, "global.lua")

	fileSymbols, err := ExtractSymbols(testFile)
	if err != nil {
		t.Fatalf("ExtractSymbols failed: %v", err)
	}
	if fileSymbols == nil {
		t.Fatal("Expected fileSymbols, got nil")
	}

	// シンボルをマップに集約
	symbolMap := make(map[string]Symbol)
	for _, sym := range fileSymbols.Symbols {
		symbolMap[sym.Name] = sym
	}

	// function globalFunc（localなし）
	if sym, ok := symbolMap["globalFunc"]; !ok {
		t.Error("Expected to find 'globalFunc' function")
	} else {
		// globalなのでlocalが含まれない
		if strings.Contains(sym.Signature, "local") {
			t.Errorf("Expected signature NOT to contain 'local', got '%s'", sym.Signature)
		}
	}

	// function calculate
	if _, ok := symbolMap["calculate"]; !ok {
		t.Error("Expected to find 'calculate' function")
	}
}

func TestExtractLuaModuleFunction(t *testing.T) {
	content := `local M = {}

function M.init()
    M.data = {}
end

function M.add(item)
    table.insert(M.data, item)
end

function M.get(index)
    return M.data[index]
end

return M
`
	tmpDir := t.TempDir()
	testutil.CreateTempFile(t, tmpDir, "module.lua", content)
	testFile := filepath.Join(tmpDir, "module.lua")

	fileSymbols, err := ExtractSymbols(testFile)
	if err != nil {
		t.Fatalf("ExtractSymbols failed: %v", err)
	}
	if fileSymbols == nil {
		t.Fatal("Expected fileSymbols, got nil")
	}

	// モジュール関数を探す
	moduleFuncs := make(map[string]Symbol)
	for _, sym := range fileSymbols.Symbols {
		if strings.HasPrefix(sym.Name, "M.") {
			moduleFuncs[sym.Name] = sym
		}
	}

	// M.init, M.add, M.get が抽出されるか
	expectedFuncs := []string{"M.init", "M.add", "M.get"}
	for _, name := range expectedFuncs {
		if _, ok := moduleFuncs[name]; !ok {
			t.Errorf("Expected to find '%s' module function", name)
		}
	}
}

func TestExtractLuaNeovimConfig(t *testing.T) {
	// Neovim設定ファイルのパターン
	content := `local opts = { noremap = true, silent = true }

local function map(mode, lhs, rhs)
    vim.keymap.set(mode, lhs, rhs, opts)
end

function Setup()
    map('n', '<leader>ff', '<cmd>Telescope find_files<cr>')
    map('n', '<leader>fg', '<cmd>Telescope live_grep<cr>')
end

return {
    setup = Setup,
    map = map,
}
`
	tmpDir := t.TempDir()
	testutil.CreateTempFile(t, tmpDir, "keymaps.lua", content)
	testFile := filepath.Join(tmpDir, "keymaps.lua")

	fileSymbols, err := ExtractSymbols(testFile)
	if err != nil {
		t.Fatalf("ExtractSymbols failed: %v", err)
	}
	if fileSymbols == nil {
		t.Fatal("Expected fileSymbols, got nil")
	}

	// 関数を探す
	functions := make(map[string]Symbol)
	for _, sym := range fileSymbols.Symbols {
		if sym.Kind == "function" {
			functions[sym.Name] = sym
		}
	}

	// map（local）とSetup（global）が抽出されるか
	if _, ok := functions["map"]; !ok {
		t.Error("Expected to find 'map' function")
	}
	if _, ok := functions["Setup"]; !ok {
		t.Error("Expected to find 'Setup' function")
	}
}

func TestLuaIsSupportedFile(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		{"init.lua", true},
		{"config/keymaps.lua", true},
		{"plugin.lua", true},
		{"main.js", true},   // JSはサポート
		{"test.txt", false}, // txtはサポート外
	}

	for _, tt := range tests {
		result := IsSupportedFile(tt.path)
		if result != tt.expected {
			t.Errorf("IsSupportedFile(%s) = %v, expected %v", tt.path, result, tt.expected)
		}
	}
}
