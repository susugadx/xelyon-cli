package e2e

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/agent"
	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"

	// プロバイダー登録（init()で自動登録）
	_ "github.com/susugadx/xelyon-cli/internal/api/providers/openai"
)

// e2eConfig はE2Eテスト用の共通設定を返す
func e2eConfig() *config.Config {
	cfg := config.DefaultConfig()
	cfg.General.UILanguage = "en"
	cfg.General.ToolLoopLimit = 5
	return cfg
}

// e2eProvider はE2Eテスト用のOpenAIプロバイダーを返す。
// OPENAI_API_KEY が未設定の場合はテストをスキップする。
func e2eProvider(t *testing.T) api.Provider {
	t.Helper()
	if os.Getenv("OPENAI_API_KEY") == "" {
		t.Skip("OPENAI_API_KEY not set, skipping E2E test")
	}
	provider, err := api.NewProvider("openai")
	if err != nil {
		t.Fatalf("failed to create OpenAI provider: %v", err)
	}
	return provider
}

// runHeadless はE2E用のヘッドレス実行ヘルパー
func runHeadless(t *testing.T, query string) *agent.HeadlessResult {
	t.Helper()
	provider := e2eProvider(t)
	result := agent.RunHeadlessWithConfig(context.Background(), query, e2eModel, provider, e2eConfig())
	if result == nil {
		t.Fatal("RunHeadlessWithConfig returned nil")
	}
	return result
}

// assertSuccess はresultがsuccessであることを検証する
func assertSuccess(t *testing.T, result *agent.HeadlessResult) {
	t.Helper()
	if result.Status != "success" {
		errMsg := ""
		if result.Error != nil {
			errMsg = result.Error.Message
		}
		t.Fatalf("expected success, got status=%s error=%s", result.Status, errMsg)
	}
}

// assertToolUsed は指定ツールが呼ばれたことを検証する
func assertToolUsed(t *testing.T, result *agent.HeadlessResult, toolName string) {
	t.Helper()
	for _, tc := range result.ToolCalls {
		if tc.Tool == toolName {
			return
		}
	}
	t.Errorf("expected tool %q to be called, but it was not (calls: %v)", toolName, toolCallNames(result))
}

// toolCallNames はツール呼び出し名の一覧を返す
func toolCallNames(result *agent.HeadlessResult) []string {
	names := make([]string, len(result.ToolCalls))
	for i, tc := range result.ToolCalls {
		names[i] = tc.Tool
	}
	return names
}

// findToolOutput は指定ツールの最初の出力を返す
func findToolOutput(result *agent.HeadlessResult, toolName string) (string, bool) {
	for _, tc := range result.ToolCalls {
		if tc.Tool == toolName {
			return tc.Output, true
		}
	}
	return "", false
}

// hasToolOutputContaining は指定ツールのいずれかの実行結果に needle が含まれるかを返す。
func hasToolOutputContaining(result *agent.HeadlessResult, toolName, needle string) bool {
	for _, tc := range result.ToolCalls {
		if tc.Tool == toolName && strings.Contains(tc.Output, needle) {
			return true
		}
	}
	return false
}

// formatToolCalls はE2E失敗時に、モデルが実際に渡した引数と出力を表示する。
func formatToolCalls(result *agent.HeadlessResult, toolName string) string {
	var b strings.Builder
	count := 0
	for i, tc := range result.ToolCalls {
		if tc.Tool != toolName {
			continue
		}
		if count > 0 {
			b.WriteString("; ")
		}
		fmt.Fprintf(&b, "#%d args=%v success=%t output=%q", i+1, tc.Args, tc.Success, truncate(tc.Output, 200))
		count++
	}
	if count == 0 {
		return "<none>"
	}
	return b.String()
}
