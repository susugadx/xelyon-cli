//go:build !norepomap
// +build !norepomap

package repomap

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/testutil"
)

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
