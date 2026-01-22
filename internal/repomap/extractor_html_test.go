//go:build !norepomap
// +build !norepomap

package repomap

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/testutil"
)

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
