//go:build !norepomap
// +build !norepomap

package repomap

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/testutil"
)

// ================== CSS Tests ==================

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

// ================== Tailwind CSS Config Tests ==================

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
