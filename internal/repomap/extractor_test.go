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
