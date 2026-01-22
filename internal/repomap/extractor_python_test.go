//go:build !norepomap
// +build !norepomap

package repomap

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/testutil"
)

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
