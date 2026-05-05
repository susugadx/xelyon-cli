// Package e2e は実際のLLM APIを使用するE2Eテスト。
// XELYON_E2E=1 環境変数がセットされているときのみ実行される。
package e2e

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

// e2eModel はE2Eテストで使用するモデル（最安）
const e2eModel = "gpt-5.4-nano"

func TestMain(m *testing.M) {
	if os.Getenv("XELYON_E2E") != "1" {
		fmt.Println("Skipping E2E tests (set XELYON_E2E=1 to run)")
		os.Exit(0)
	}

	// テスト実行ディレクトリは e2e/ になるため、
	// ツールがプロジェクトルートのファイルを参照できるよう親ディレクトリに移動する。
	if err := os.Chdir(".."); err != nil {
		fmt.Fprintf(os.Stderr, "failed to chdir to project root: %v\n", err)
		os.Exit(1)
	}

	os.Exit(m.Run())
}

// TestE2E_SearchCode は gather_context 経由のコード検索が正常に実行されることを確認する
func TestE2E_SearchCode(t *testing.T) {
	t.Parallel()
	result := runHeadless(t,
		`Use the gather_context tool to search for the pattern "func main" and report what files contain it. Do not use any other tools.`)

	assertSuccess(t, result)
	assertToolUsed(t, result, "gather_context")

	if !hasToolOutputContaining(result, "gather_context", "main.go") {
		t.Errorf("expected any gather_context output to contain main.go, calls: %s", formatToolCalls(result, "gather_context"))
	}
}

// TestE2E_ReadFile はread_fileツールが正常に実行されることを確認する
func TestE2E_ReadFile(t *testing.T) {
	t.Parallel()
	result := runHeadless(t,
		`Use the read_file tool to read the file "go.mod" and report the module name. Do not use any other tools.`)

	assertSuccess(t, result)
	assertToolUsed(t, result, "read_file")

	output, ok := findToolOutput(result, "read_file")
	if !ok {
		t.Fatal("read_file output not found")
	}
	if !strings.Contains(output, "github.com/susugadx/xelyon-cli") {
		t.Errorf("expected read_file output to contain module name, got: %s", truncate(output, 200))
	}
}

// TestE2E_ReadFileBatch はread_fileの複数ファイル読み込みが正常に動作することを確認する
func TestE2E_ReadFileBatch(t *testing.T) {
	t.Parallel()
	result := runHeadless(t,
		`Use the read_file tool to read both "go.mod" and "Makefile" in a single call using the paths parameter. Report what you found. Do not use any other tools.`)

	assertSuccess(t, result)
	assertToolUsed(t, result, "read_file")

	output, ok := findToolOutput(result, "read_file")
	if !ok {
		t.Fatal("read_file output not found")
	}
	// go.mod の内容が含まれることを確認
	if !strings.Contains(output, "github.com/susugadx/xelyon-cli") {
		t.Errorf("expected read_file output to contain go.mod content, got: %s", truncate(output, 200))
	}
	// Makefile の内容が含まれることを確認
	if !strings.Contains(output, "Makefile") || !strings.Contains(output, "build") {
		t.Errorf("expected read_file output to contain Makefile content, got: %s", truncate(output, 200))
	}
}

// TestE2E_SearchCodeSymbolResolve は gather_context のシンボル解決が正常に実行されることを確認する
func TestE2E_SearchCodeSymbolResolve(t *testing.T) {
	t.Parallel()
	result := runHeadless(t,
		`Use the gather_context tool to search for the symbol "main" and report what you found. Do not use any other tools.`)

	assertSuccess(t, result)
	assertToolUsed(t, result, "gather_context")
}

// TestE2E_SimpleQuery は簡単な調査タスクが完了することを確認する
func TestE2E_SimpleQuery(t *testing.T) {
	t.Parallel()
	result := runHeadless(t,
		`main.go の main 関数の役割を1行で説明せよ。必要に応じてread_fileを使ってよい。`)

	assertSuccess(t, result)

	if result.Response == "" {
		t.Error("expected non-empty response")
	}
}

// truncate は文字列を指定長で切り詰める
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
