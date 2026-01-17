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
		{"test.txt", false},
		{"test.md", false},
		{"test.json", false},
	}

	for _, tt := range tests {
		result := IsSupportedFile(tt.path)
		if result != tt.expected {
			t.Errorf("IsSupportedFile(%s) = %v, expected %v", tt.path, result, tt.expected)
		}
	}
}
