//go:build !norepomap
// +build !norepomap

package repomap

import (
	"path/filepath"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/testutil"
)

// ================== Java Tests ==================

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

// ================== Ruby Tests ==================

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
