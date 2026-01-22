//go:build !norepomap
// +build !norepomap

package repomap

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/testutil"
)

// ================== Vue SFC Tests ==================

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

// ================== Svelte SFC Tests ==================

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

// ================== SFC Helper Function Tests ==================

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
