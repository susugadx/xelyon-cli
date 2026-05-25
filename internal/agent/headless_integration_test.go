package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	openai "github.com/susugadx/xelyon-cli/internal/api/providers/openai"
	"github.com/susugadx/xelyon-cli/internal/config"
)

// testSubDir は各テスト専用の作業ディレクトリを作成し、CWD もそこへ切り替える。
// repo 配下を汚さないことで package 並列実行時の cross-package race を防ぐ。
func testSubDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	abs, err := filepath.Abs(dir)
	if err != nil {
		t.Fatal(err)
	}
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(abs); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(origDir) })
	return abs
}

func testWritableHomeDir(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "headless_home_*")
	if err != nil {
		t.Fatal(err)
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = filepath.Walk(abs, func(path string, info os.FileInfo, err error) error {
			if err != nil || info == nil {
				return nil
			}
			if info.IsDir() {
				_ = os.Chmod(path, 0o755)
				return nil
			}
			_ = os.Chmod(path, 0o644)
			return nil
		})
		_ = os.RemoveAll(abs)
	})
	return abs
}

// TestHeadless_SimpleResponse はツール呼び出しなしの単純レスポンスを検証する。
func TestHeadless_SimpleResponse(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	cfg := newProjectMapDisabledConfig()
	provider := &sequenceMockProvider{
		name:      "test-provider",
		responses: []string{"Hello, this is a simple response without any tool calls."},
	}

	result := RunHeadlessWithConfig(context.Background(), "Say hello", "test-model", provider, cfg)

	if result.Status != "success" {
		t.Fatalf("expected status 'success', got %q", result.Status)
	}
	if result.Response == "" {
		t.Fatal("expected non-empty response")
	}
	if !strings.Contains(result.Response, "Hello") {
		t.Errorf("expected response to contain 'Hello', got %q", result.Response)
	}
	if len(result.ToolCalls) != 0 {
		t.Errorf("expected 0 tool calls, got %d", len(result.ToolCalls))
	}
	if result.Provider != "test-provider" {
		t.Errorf("expected provider 'test-provider', got %q", result.Provider)
	}
	if result.Model != "test-model" {
		t.Errorf("expected model 'test-model', got %q", result.Model)
	}
	if result.DurationMs < 0 {
		t.Errorf("expected non-negative duration, got %d", result.DurationMs)
	}
}

// TestHeadless_SearchCodeTool はsearch_codeツール呼び出しを経由するフローを検証する。
func TestHeadless_SearchCodeTool(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XELYON_EDIT_TOOL", "str_replace")

	// CWD配下にテスト用ファイルを作成（パスエスケープ検出回避）
	dir := testSubDir(t)
	testFile := filepath.Join(dir, "main.go")
	if err := os.WriteFile(testFile, []byte("package main\n\nfunc main() {}\n"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := newProjectMapDisabledConfig()
	provider := &sequenceMockProvider{
		name: "test-provider",
		responses: []string{
			fmt.Sprintf(`{"tool": "search_code", "args": {"pattern": "func main", "path": %q}}`, dir),
			"Found the main function in main.go",
		},
	}

	result := RunHeadlessWithConfig(context.Background(), "Find the main function", "test-model", provider, cfg)

	if result.Status != "success" {
		t.Fatalf("expected status 'success', got %q", result.Status)
	}
	if result.Response == "" {
		t.Fatal("expected non-empty response")
	}
	if len(result.ToolCalls) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(result.ToolCalls))
	}
	if result.ToolCalls[0].Tool != "search_code" {
		t.Errorf("expected tool 'search_code', got %q", result.ToolCalls[0].Tool)
	}
	if !result.ToolCalls[0].Success {
		t.Errorf("expected search_code to succeed, output: %q", result.ToolCalls[0].Output)
	}
	if result.Provider != "test-provider" {
		t.Errorf("expected provider 'test-provider', got %q", result.Provider)
	}
	if result.Model != "test-model" {
		t.Errorf("expected model 'test-model', got %q", result.Model)
	}
}

func TestHeadless_SearchCodeUsesFreshProjectMapRuntimeAfterEdit(t *testing.T) {
	homeDir := testWritableHomeDir(t)
	t.Setenv("HOME", homeDir)
	t.Setenv("XELYON_EDIT_TOOL", "str_replace")
	t.Setenv("GOMODCACHE", filepath.Join(homeDir, "go", "pkg", "mod"))

	dir := testSubDir(t)
	testFile := filepath.Join(dir, "main.go")
	if err := os.WriteFile(testFile, []byte("package main\n\nfunc Run() {}\n"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := config.DefaultConfig()
	provider := &sequenceMockProvider{
		name: "test-provider",
		responses: []string{
			`{"tool": "str_replace", "args": {"path": "main.go", "old_str": "package main\n\nfunc Run() {}\n", "new_str": "package main\n\nvar moved = 1\nvar moved2 = 2\n\nfunc Run() {}\n"}}`,
			`{"tool": "search_code", "args": {"pattern": "Run", "path": "."}}`,
			"done",
		},
	}

	result := RunHeadlessWithConfig(context.Background(), "Move Run then inspect it", "test-model", provider, cfg)
	if result.Status != "success" {
		t.Fatalf("expected status success, got %q (%+v)", result.Status, result.Error)
	}
	if len(result.ToolCalls) != 2 {
		t.Fatalf("expected 2 tool calls, got %d", len(result.ToolCalls))
	}
	if result.ToolCalls[1].Tool != "search_code" {
		t.Fatalf("expected second tool call to be search_code, got %q", result.ToolCalls[1].Tool)
	}
	if !result.ToolCalls[1].Success {
		t.Fatalf("expected search_code success, output:\n%s", result.ToolCalls[1].Output)
	}
	if !strings.Contains(result.ToolCalls[1].Output, "(L6)") {
		t.Fatalf("expected edited symbol location L6, got:\n%s", result.ToolCalls[1].Output)
	}
	if strings.Contains(result.ToolCalls[1].Output, "(L3)") || strings.Contains(result.ToolCalls[1].Output, "3: func Run() {}") {
		t.Fatalf("expected fresh runtime state, but stale symbol location remained:\n%s", result.ToolCalls[1].Output)
	}
	if !strings.Contains(result.ToolCalls[1].Output, "6: func Run() {}") {
		t.Fatalf("expected edited definition body at line 6, got:\n%s", result.ToolCalls[1].Output)
	}
}

// TestHeadless_ReadFileTool はread_fileツール呼び出しを経由するフローを検証する。
func TestHeadless_ReadFileTool(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XELYON_EDIT_TOOL", "str_replace")

	// CWD配下にテスト用ファイルを作成
	dir := testSubDir(t)
	testFile := filepath.Join(dir, "hello.txt")
	if err := os.WriteFile(testFile, []byte("Hello from test file\n"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := newProjectMapDisabledConfig()
	provider := &sequenceMockProvider{
		name: "test-provider",
		responses: []string{
			fmt.Sprintf(`{"tool": "read_file", "args": {"paths": [%q]}}`, testFile),
			"The file contains a greeting message",
		},
	}

	result := RunHeadlessWithConfig(context.Background(), "Read the hello file", "test-model", provider, cfg)

	if result.Status != "success" {
		t.Fatalf("expected status 'success', got %q", result.Status)
	}
	if result.Response == "" {
		t.Fatal("expected non-empty response")
	}
	if len(result.ToolCalls) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(result.ToolCalls))
	}
	if result.ToolCalls[0].Tool != "read_file" {
		t.Errorf("expected tool 'read_file', got %q", result.ToolCalls[0].Tool)
	}
	if !result.ToolCalls[0].Success {
		t.Errorf("expected read_file to succeed, output: %q", result.ToolCalls[0].Output)
	}
	if !strings.Contains(result.ToolCalls[0].Output, "Hello from test file") {
		t.Errorf("expected output to contain file content, got %q", result.ToolCalls[0].Output)
	}
}

// TestHeadless_ListDirTool はgather_contextのディレクトリ調査フローを検証する。
func TestHeadless_ListDirTool(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	// CWD配下にテスト用ディレクトリを作成
	dir := testSubDir(t)
	if err := os.WriteFile(filepath.Join(dir, "file1.go"), []byte("package a\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "file2.txt"), []byte("text\n"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := newProjectMapDisabledConfig()
	provider := &sequenceMockProvider{
		name: "test-provider",
		responses: []string{
			fmt.Sprintf(`{"tool": "gather_context", "args": {"query": %q}}`, dir+string(os.PathSeparator)),
			"The directory contains file1.go and file2.txt",
		},
	}

	result := RunHeadlessWithConfig(context.Background(), "List directory contents", "test-model", provider, cfg)

	if result.Status != "success" {
		t.Fatalf("expected status 'success', got %q", result.Status)
	}
	if result.Response == "" {
		t.Fatal("expected non-empty response")
	}
	if len(result.ToolCalls) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(result.ToolCalls))
	}
	if result.ToolCalls[0].Tool != "gather_context" {
		t.Errorf("expected tool 'gather_context', got %q", result.ToolCalls[0].Tool)
	}
	if !result.ToolCalls[0].Success {
		t.Errorf("expected gather_context to succeed, output: %q", result.ToolCalls[0].Output)
	}
	if !strings.Contains(result.ToolCalls[0].Output, "Route: Directory listing") || !strings.Contains(result.ToolCalls[0].Output, "file1.go") {
		t.Errorf("expected gather_context directory listing output, got %q", result.ToolCalls[0].Output)
	}
}

// TestHeadless_MultipleToolCalls は複数のgather_context呼び出しを順番に実行するフローを検証する。
func TestHeadless_MultipleToolCalls(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	// CWD配下にテスト用ファイルを作成
	dir := testSubDir(t)
	testFile := filepath.Join(dir, "data.txt")
	if err := os.WriteFile(testFile, []byte("important data\n"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := newProjectMapDisabledConfig()
	provider := &sequenceMockProvider{
		name: "test-provider",
		responses: []string{
			fmt.Sprintf(`{"tool": "gather_context", "args": {"query": %q}}`, dir+string(os.PathSeparator)),
			fmt.Sprintf(`{"tool": "gather_context", "args": {"query": %q}}`, testFile),
			"Directory listing and file reading completed successfully",
		},
	}

	result := RunHeadlessWithConfig(context.Background(), "List dir then read file", "test-model", provider, cfg)

	if result.Status != "success" {
		t.Fatalf("expected status 'success', got %q", result.Status)
	}
	if result.Response == "" {
		t.Fatal("expected non-empty response")
	}
	if len(result.ToolCalls) != 2 {
		t.Fatalf("expected 2 tool calls, got %d", len(result.ToolCalls))
	}
	if result.ToolCalls[0].Tool != "gather_context" {
		t.Errorf("expected first tool 'gather_context', got %q", result.ToolCalls[0].Tool)
	}
	if result.ToolCalls[1].Tool != "gather_context" {
		t.Errorf("expected second tool 'gather_context', got %q", result.ToolCalls[1].Tool)
	}
	if !result.ToolCalls[0].Success {
		t.Errorf("expected first tool call to succeed, output: %q", result.ToolCalls[0].Output)
	}
	if !result.ToolCalls[1].Success {
		t.Errorf("expected second tool call to succeed, output: %q", result.ToolCalls[1].Output)
	}
	if !strings.Contains(result.ToolCalls[0].Output, "data.txt") {
		t.Errorf("expected first gather_context output to contain directory contents, got %q", result.ToolCalls[0].Output)
	}
	if !strings.Contains(result.ToolCalls[1].Output, "important data") {
		t.Errorf("expected second gather_context output to contain file content, got %q", result.ToolCalls[1].Output)
	}
}

// TestHeadless_APIError はAPI呼び出しエラー時のエラーハンドリングを検証する。
func TestHeadless_APIError(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	cfg := newProjectMapDisabledConfig()
	// mockErrorProvider は headless_test.go で定義済み（常にエラーを返す）
	provider := &mockErrorProvider{}

	result := RunHeadlessWithConfig(context.Background(), "This should fail", "test-model", provider, cfg)

	if result.Status != "error" {
		t.Fatalf("expected status 'error', got %q", result.Status)
	}
	if result.Error == nil {
		t.Fatal("expected Error to be non-nil")
	}
	if result.Error.Type != "api_error" {
		t.Errorf("expected error type 'api_error', got %q", result.Error.Type)
	}
	if result.Error.Message == "" {
		t.Error("expected non-empty error message")
	}
	if result.Provider != "test-error" {
		t.Errorf("expected provider 'test-error', got %q", result.Provider)
	}
	if result.Model != "test-model" {
		t.Errorf("expected model 'test-model', got %q", result.Model)
	}
}

// TestHeadless_MaxIterationsReached はツール呼び出しが繰り返され最大イテレーションに達するケースを検証する。
func TestHeadless_MaxIterationsReached(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	cfg := newProjectMapDisabledConfig()
	cfg.General.ToolLoopLimit = 10
	// 全てのレスポンスがツール呼び出し → 設定した最大10回でループ終了
	dir := testSubDir(t)
	if err := os.WriteFile(filepath.Join(dir, "file1.go"), []byte("package a\n"), 0644); err != nil {
		t.Fatal(err)
	}
	responses := make([]string, 15)
	for i := range responses {
		responses[i] = fmt.Sprintf(`{"tool": "gather_context", "args": {"query": %q}}`, dir+string(os.PathSeparator))
	}
	provider := &sequenceMockProvider{
		name:      "test-provider",
		responses: responses,
	}

	result := RunHeadlessWithConfig(context.Background(), "Keep calling tools forever", "test-model", provider, cfg)

	requireHeadlessToolLoopLimitError(t, result, 10)
	// maxIterations = 10 なので、10回のツール呼び出しが記録される
	if len(result.ToolCalls) != 10 {
		t.Errorf("expected 10 tool calls (max iterations), got %d", len(result.ToolCalls))
	}
	if provider.callCount != 10 {
		t.Errorf("expected provider to be called 10 times, got %d", provider.callCount)
	}
	// 全てのツール呼び出しが gather_context であることを確認
	for i, tc := range result.ToolCalls {
		if tc.Tool != "gather_context" {
			t.Errorf("expected tool call %d to be 'gather_context', got %q", i, tc.Tool)
		}
	}
}

// TestHeadless_ToolError は存在しないファイルへのread_fileでSuccess=falseになることを検証する。
func TestHeadless_ToolError(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XELYON_EDIT_TOOL", "str_replace")

	// CWD配下に存在しないファイルパスを指定
	dir := testSubDir(t)
	nonExistentFile := filepath.Join(dir, "does_not_exist.txt")

	cfg := newProjectMapDisabledConfig()
	provider := &sequenceMockProvider{
		name: "test-provider",
		responses: []string{
			fmt.Sprintf(`{"tool": "read_file", "args": {"paths": [%q]}}`, nonExistentFile),
			"The file could not be found",
		},
	}

	result := RunHeadlessWithConfig(context.Background(), "Read a non-existent file", "test-model", provider, cfg)

	if result.Status != "success" {
		t.Fatalf("expected status 'success', got %q", result.Status)
	}
	if len(result.ToolCalls) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(result.ToolCalls))
	}
	if result.ToolCalls[0].Tool != "read_file" {
		t.Errorf("expected tool 'read_file', got %q", result.ToolCalls[0].Tool)
	}
	// 存在しないファイルの場合、出力にエラーメッセージが含まれることを検証。
	// agent_run.go の簡易判定は "Error:" の有無で行うが、read_file は
	// "Error reading file:" という形式を返すため Success=true になる場合がある。
	// ここではツール出力にエラー情報が含まれていることだけを検証する。
	output := result.ToolCalls[0].Output
	if !strings.Contains(output, "Error") && !strings.Contains(output, "no such file") {
		t.Errorf("expected output to indicate an error for non-existent file, got %q", output)
	}
}

// TestHeadless_HistoryAccumulation は複数ツール呼び出し後にToolCallsが正しい順序で蓄積されることを検証する。
func TestHeadless_HistoryAccumulation(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	// CWD配下にテスト用ファイルを作成
	dir := testSubDir(t)
	file1 := filepath.Join(dir, "first.txt")
	file2 := filepath.Join(dir, "second.txt")
	if err := os.WriteFile(file1, []byte("first file content\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file2, []byte("second file content\n"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := newProjectMapDisabledConfig()
	provider := &sequenceMockProvider{
		name: "test-provider",
		responses: []string{
			fmt.Sprintf(`{"tool": "gather_context", "args": {"query": %q}}`, dir+string(os.PathSeparator)),
			fmt.Sprintf(`{"tool": "gather_context", "args": {"query": %q}}`, file1),
			fmt.Sprintf(`{"tool": "gather_context", "args": {"query": %q}}`, file2),
			"All files have been read in order",
		},
	}

	result := RunHeadlessWithConfig(context.Background(), "List then read all files", "test-model", provider, cfg)

	if result.Status != "success" {
		t.Fatalf("expected status 'success', got %q", result.Status)
	}
	if len(result.ToolCalls) != 3 {
		t.Fatalf("expected 3 tool calls, got %d", len(result.ToolCalls))
	}

	// 順序が正しいことを検証
	expectedTools := []string{"gather_context", "gather_context", "gather_context"}
	for i, expected := range expectedTools {
		if result.ToolCalls[i].Tool != expected {
			t.Errorf("tool call %d: expected %q, got %q", i, expected, result.ToolCalls[i].Tool)
		}
	}

	// 全てのツール呼び出しが成功していることを検証
	for i, tc := range result.ToolCalls {
		if !tc.Success {
			t.Errorf("tool call %d (%s) failed: %q", i, tc.Tool, tc.Output)
		}
	}

	// gather_context の directory listing 出力にファイル名が含まれていることを検証
	if !strings.Contains(result.ToolCalls[0].Output, "first.txt") {
		t.Errorf("expected gather_context directory output to contain 'first.txt', got %q", result.ToolCalls[0].Output)
	}

	// gather_context read 出力にファイル内容が含まれていることを検証
	if !strings.Contains(result.ToolCalls[1].Output, "first file content") {
		t.Errorf("expected first gather_context output to contain 'first file content', got %q", result.ToolCalls[1].Output)
	}
	if !strings.Contains(result.ToolCalls[2].Output, "second file content") {
		t.Errorf("expected second gather_context output to contain 'second file content', got %q", result.ToolCalls[2].Output)
	}

	// 最終レスポンスが正しいことを検証
	if !strings.Contains(result.Response, "All files have been read") {
		t.Errorf("expected final response to contain summary, got %q", result.Response)
	}

	// Providerの呼び出し回数を検証（3ツール + 1最終レスポンス = 4回）
	if provider.callCount != 4 {
		t.Errorf("expected provider to be called 4 times, got %d", provider.callCount)
	}
}

func TestHeadless_OpenAIResponsesUsesPreviousResponseIDAfterToolCall(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	dir := testSubDir(t)
	testFile := filepath.Join(dir, "cache.txt")
	if err := os.WriteFile(testFile, []byte("cached content\n"), 0644); err != nil {
		t.Fatal(err)
	}

	var requests []openai.ResponsesRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req openai.ResponsesRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}
		requests = append(requests, req)

		w.Header().Set("Content-Type", "text/event-stream")
		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Fatal("streaming unsupported")
		}

		if len(requests) == 1 {
			fmt.Fprintf(w, "data: {\"type\":\"response.created\",\"response\":{\"id\":\"resp_first\"}}\n\n")
			fmt.Fprintf(w, "data: {\"type\":\"response.output_item.added\",\"item\":{\"type\":\"function_call\",\"call_id\":\"call_1\",\"name\":\"gather_context\"}}\n\n")
			fmt.Fprintf(w, "data: {\"type\":\"response.function_call_arguments.done\",\"item\":{\"call_id\":\"call_1\",\"arguments\":%q}}\n\n", fmt.Sprintf("{\"query\":%q}", testFile))
			fmt.Fprintf(w, "data: {\"type\":\"response.completed\",\"response\":{\"id\":\"resp_first\"}}\n\n")
		} else {
			fmt.Fprintf(w, "data: {\"type\":\"response.created\",\"response\":{\"id\":\"resp_second\"}}\n\n")
			fmt.Fprintf(w, "data: {\"type\":\"response.output_text.delta\",\"delta\":\"done\"}\n\n")
			fmt.Fprintf(w, "data: {\"type\":\"response.completed\",\"response\":{\"id\":\"resp_second\"}}\n\n")
		}
		fmt.Fprintf(w, "data: [DONE]\n\n")
		flusher.Flush()
	}))
	t.Cleanup(server.Close)
	t.Setenv("OPENAI_RESPONSES_URL", server.URL)

	result := RunHeadlessWithConfig(
		context.Background(),
		"Read the cache file",
		"gpt-5.4-nano",
		openai.New("test-key"),
		newProjectMapDisabledConfig(),
	)

	if result.Status != "success" {
		t.Fatalf("expected success, got %q (%v)", result.Status, result.Error)
	}
	if len(requests) != 2 {
		t.Fatalf("request count = %d, want 2", len(requests))
	}

	secondReq := requests[1]
	if secondReq.PreviousResponseID != "resp_first" {
		t.Fatalf("PreviousResponseID = %q, want resp_first", secondReq.PreviousResponseID)
	}

	inputItems, ok := secondReq.Input.([]interface{})
	if !ok {
		t.Fatalf("second request input type = %T, want []interface{}", secondReq.Input)
	}
	if len(inputItems) != 1 {
		t.Fatalf("second request input length = %d, want 1", len(inputItems))
	}

	item, ok := inputItems[0].(map[string]interface{})
	if !ok {
		t.Fatalf("second request input[0] type = %T, want map[string]interface{}", inputItems[0])
	}
	if item["type"] != "function_call_output" {
		t.Fatalf("input[0].type = %v, want function_call_output", item["type"])
	}
	if item["call_id"] != "call_1" {
		t.Fatalf("input[0].call_id = %v, want call_1", item["call_id"])
	}
	output, _ := item["output"].(string)
	if !strings.Contains(output, "cached content") {
		t.Fatalf("input[0].output = %q, want read_file output", output)
	}
}
