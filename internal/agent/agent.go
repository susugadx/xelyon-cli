package agent

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/fatih/color"
	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/history"
	"github.com/susugadx/xelyon-cli/internal/lsp"
	"github.com/susugadx/xelyon-cli/internal/mcp"
	"github.com/susugadx/xelyon-cli/internal/prompt"
	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/ui"

	// Subpackage imports - trigger init() for tool registration
	_ "github.com/susugadx/xelyon-cli/internal/tools/dev"
	_ "github.com/susugadx/xelyon-cli/internal/tools/file"
	_ "github.com/susugadx/xelyon-cli/internal/tools/git"
	toolslsp "github.com/susugadx/xelyon-cli/internal/tools/lsp"
	_ "github.com/susugadx/xelyon-cli/internal/tools/search"
)

// 色定義
var (
	cyan   = color.New(color.FgCyan, color.Bold)
	green  = color.New(color.FgGreen)
	yellow = color.New(color.FgYellow)
	red    = color.New(color.FgRed)
)

// Agent はCLIエージェント
type Agent struct {
	Model                string // 初期モデル（後方互換性のため保持）
	CurrentModel         string // 現在のモデル（再起動なしで切り替え可能）
	CurrentProvider      api.Provider
	ProviderName         string
	History              []api.Message
	SystemPrompt         string
	session              *history.Session
	storage              *history.Storage
	changeStack          []tools.FileChange
	changeStorage        *history.ChangeStorage // 永続的変更履歴
	mcpManager           *mcp.Manager
	lspClient            *lsp.Client         // LSPクライアント
	AutoApprove          bool                // --auto-approve フラグ
	Stats                *SessionStats       // セッション統計情報
	lastOutputs          []string            // 最後のAI出力履歴（最大10件）
	cancelFunc           context.CancelFunc  // 現在のAPI呼び出しをキャンセルするための関数
	strReplaceErrorCount int                 // str_replace連続エラーカウント（old_str not found）
	mlReader             *ui.MultilineReader // 共有入力リーダー（ペーストモードでも使用）
	PlanModeEnabled      bool                // Plan Mode ON/OFF（デフォルト: false）
	ToolCache            *ToolCache          // ツール結果キャッシュ（read_file, list_dir）

	// OpenAI Compact API 関連
	compactedItems  []api.InputItem // 圧縮済みアイテム
	isCompactedMode bool            // 圧縮モードフラグ

	// 並列実行用ミューテックス
	historyMu     sync.Mutex
	changeStackMu sync.Mutex
	statsMu       sync.Mutex
}

// NewAgent は新しいAgentを作成
func NewAgent(model string, provider api.Provider) *Agent {
	storage, err := history.NewStorage()
	if err != nil {
		red.Printf("Warning: Failed to initialize history storage: %v\n", err)
		storage = nil
	}

	// MCP初期化
	mcpManager := mcp.NewManager()
	if err := mcpManager.LoadConfig(); err != nil {
		yellow.Printf("Warning: Failed to load MCP config: %v\n", err)
	}

	ctx := context.Background()
	if err := mcpManager.Connect(ctx); err != nil {
		yellow.Printf("Warning: MCP connection error: %v\n", err)
	}

	// MCPツールをTool Registryに登録
	if len(mcpManager.GetTools()) > 0 {
		mcpManager.RegisterToToolRegistry(tools.DefaultRegistry)
	}

	systemPrompt := prompt.SystemPrompt

	// MCPツールをSystemPromptに追加
	if len(mcpManager.GetTools()) > 0 {
		systemPrompt += buildMCPToolsPrompt(mcpManager)
	}

	// 変更履歴ストレージ初期化
	changeStorage, err := history.NewChangeStorage()
	if err != nil {
		yellow.Printf("Warning: Failed to initialize change storage: %v\n", err)
		changeStorage = nil
	}

	// LSP初期化
	var lspClient *lsp.Client
	cfg := config.GetGlobalConfig()
	if cfg.LSP.Enabled {
		cwd, err := os.Getwd()
		if err == nil {
			lspClient = lsp.NewClient(cwd)
			// Config形式からLSP形式に変換
			servers := make(map[string]lsp.ServerConfig)
			for lang, serverCfg := range cfg.LSP.Servers {
				servers[lang] = lsp.ServerConfig{
					Command:  serverCfg.Command,
					Args:     serverCfg.Args,
					Disabled: serverCfg.Disabled,
				}
			}
			lspClient.SetConfigs(servers)
			toolslsp.LSPClient = lspClient

			// LSPツールはinit()で自動登録済み

			// SystemPromptにLSPツール説明を追加
			systemPrompt += prompt.BuildLSPToolsPrompt()
		}
	}

	// Function Calling 対応プロバイダーは詳細なツール定義を送信するため、
	// System Prompt からツール説明を除去（重複回避、トークン節約）
	if provider.Name() == "Gemini" || provider.Name() == "OpenAI" {
		systemPrompt = removeToolsSection(systemPrompt)
	}

	// MCPToolProviderインターフェースを実装するプロバイダーにMCPツールを設定
	// （Function Calling経由で呼び出し可能にする）
	if len(mcpManager.GetTools()) > 0 {
		// Gemini MCP ツール設定
		if mcpProvider, ok := provider.(api.MCPToolProvider); ok {
			debug := os.Getenv("XELYON_DEBUG_GEMINI") == "1"
			var mcpDeclarations []api.GeminiFunctionDeclaration
			for _, t := range mcpManager.GetTools() {
				// ツール名: mcp_{serverName}_{toolName}
				// sanitizeToolName は agent_mcp.go に定義済み
				// MCPToolWrapper.Name() と同じロジックで名前の一貫性を保証
				name := fmt.Sprintf("mcp_%s_%s", sanitizeToolName(t.ServerName), sanitizeToolName(t.Name))
				decl := api.ConvertMCPToolToGeminiDeclaration(name, t.Description, t.InputSchema)
				mcpDeclarations = append(mcpDeclarations, decl)

				// デバッグログ
				if debug {
					fmt.Fprintf(os.Stderr, "[DEBUG Gemini] MCP tool registered: %s\n", name)
				}
			}
			mcpProvider.SetMCPTools(mcpDeclarations)
		}

		// OpenAI MCP ツール設定
		if openaiMCPProvider, ok := provider.(api.OpenAIMCPToolProvider); ok {
			debug := os.Getenv("XELYON_DEBUG_OPENAI") == "1"
			var mcpFunctions []api.OpenAIToolFunction
			for _, t := range mcpManager.GetTools() {
				name := fmt.Sprintf("mcp_%s_%s", sanitizeToolName(t.ServerName), sanitizeToolName(t.Name))
				fn := api.ConvertMCPToolToOpenAIFunction(name, t.Description, t.InputSchema)
				mcpFunctions = append(mcpFunctions, fn)

				// デバッグログ
				if debug {
					fmt.Fprintf(os.Stderr, "[DEBUG OpenAI] MCP tool registered: %s\n", name)
				}
			}
			openaiMCPProvider.SetMCPTools(mcpFunctions)
		}
	}

	// ToolCache 初期化
	toolCache := NewToolCache()
	tools.GlobalToolCache = toolCache

	return &Agent{
		Model:           model,
		CurrentModel:    model,
		CurrentProvider: provider,
		ProviderName:    provider.Name(),
		History:         []api.Message{},
		session:         history.NewSession(model),
		storage:         storage,
		changeStack:     []tools.FileChange{},
		changeStorage:   changeStorage,
		mcpManager:      mcpManager,
		lspClient:       lspClient,
		SystemPrompt:    systemPrompt,
		Stats:           NewSessionStats(provider.Name()),
		lastOutputs:     []string{},
		ToolCache:       toolCache,
	}
}

// removeToolsSection は System Prompt から ## Available Tools セクションを除去
// Gemini は Function Calling で詳細なツール定義を受け取るため、重複を避ける
// ## Workflow Rules 以降は保持（動作指針として重要）
func removeToolsSection(prompt string) string {
	const toolsStart = "## Available Tools"
	const toolsEnd = "## Workflow Rules"

	startIdx := strings.Index(prompt, toolsStart)
	if startIdx == -1 {
		return prompt
	}

	endIdx := strings.Index(prompt, toolsEnd)
	if endIdx == -1 {
		return prompt
	}

	// toolsStart から toolsEnd の直前までを削除
	return prompt[:startIdx] + prompt[endIdx:]
}

// Cleanup はエージェントのリソースをクリーンアップ
func (a *Agent) Cleanup() {
	if a.mcpManager != nil {
		a.mcpManager.Close()
	}
	// LSPクリーンアップ
	if a.lspClient != nil {
		a.lspClient.Close()
	}
	// セッション保存
	if a.storage != nil && a.session != nil {
		if err := a.storage.Save(a.session); err != nil {
			yellow.Printf("Warning: Failed to save session: %v\n", err)
		}
	}
}

// GetLSPClient はLSPクライアントを返す（コマンド用）
func (a *Agent) GetLSPClient() *lsp.Client {
	return a.lspClient
}

// appendHistory は History へスレッドセーフに追加（並列実行時用）
func (a *Agent) appendHistory(msg api.Message) {
	a.historyMu.Lock()
	defer a.historyMu.Unlock()
	a.History = append(a.History, msg)
}

// getHistorySnapshot は History のスナップショットをスレッドセーフに取得（並列実行時用）
func (a *Agent) getHistorySnapshot() []api.Message {
	a.historyMu.Lock()
	defer a.historyMu.Unlock()
	snapshot := make([]api.Message, len(a.History))
	copy(snapshot, a.History)
	return snapshot
}

// appendChange は changeStack へスレッドセーフに追加（並列実行時用）
func (a *Agent) appendChange(change tools.FileChange) {
	a.changeStackMu.Lock()
	defer a.changeStackMu.Unlock()
	a.changeStack = append(a.changeStack, change)
	if len(a.changeStack) > config.MaxChangeStack {
		a.changeStack = a.changeStack[1:]
	}
}

// incrementToolExecution は Stats をスレッドセーフに更新（並列実行時用）
func (a *Agent) incrementToolExecution(toolName string) {
	a.statsMu.Lock()
	defer a.statsMu.Unlock()
	if a.Stats != nil {
		a.Stats.AddToolExecution(toolName)
	}
}

// incrementAssistantMessages は Stats をスレッドセーフに更新（並列実行時用）
func (a *Agent) incrementAssistantMessages() {
	a.statsMu.Lock()
	defer a.statsMu.Unlock()
	if a.Stats != nil {
		a.Stats.AssistantMessages++
	}
}
