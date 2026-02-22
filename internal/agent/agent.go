package agent

import (
	"context"
	"encoding/json"
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

	"github.com/susugadx/xelyon-cli/internal/i18n"

	// Subpackage imports - trigger init() for tool registration
	_ "github.com/susugadx/xelyon-cli/internal/tools/dev"
	_ "github.com/susugadx/xelyon-cli/internal/tools/file"
	toolslsp "github.com/susugadx/xelyon-cli/internal/tools/lsp"
	_ "github.com/susugadx/xelyon-cli/internal/tools/planning"
	_ "github.com/susugadx/xelyon-cli/internal/tools/search"
)

// 色定義
var (
	cyan   = color.New(color.FgCyan, color.Bold)
	green  = color.New(color.FgGreen)
	yellow = color.New(color.FgYellow)
	red    = color.New(color.FgRed)
	dim    = color.New(color.Faint)
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
	taskChangeOffset     int                    // タスク開始時の changeStack 長（タスク単位のサマリー表示用）
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
	taskBaseCommitHash   string              // タスク開始時のHEADコミットハッシュ（completion hook の diff 空チェック判定用）

	// LSP診断遅延バッファ: 連続str_replace途中の一時的エラーによる誤auto-retry防止用。
	// str_replace成功後に対象ファイルを追加し、次の非str_replaceアクション時にフラッシュして再診断する。
	pendingLSPFiles []string

	// OpenAI Compact API 関連
	compactedItems  []api.InputItem // 圧縮済みアイテム
	isCompactedMode bool            // 圧縮モードフラグ

	// トークン上限エラー処理
	tokenLimitRetryCount int // トークン上限エラー時のリトライ回数（最大1回）

	// 並列実行用ミューテックス
	historyMu     sync.Mutex
	changeStackMu sync.Mutex
	statsMu       sync.Mutex
}

// NewAgent は新しいAgentを作成
func NewAgent(model string, provider api.Provider, headless bool) *Agent {
	// 言語設定を適用
	cfg := config.GetGlobalConfig()
	if cfg.General.Language != "" {
		i18n.SetLang(cfg.General.Language)
	}

	storage, err := history.NewStorage()
	if err != nil {
		red.Printf("Warning: Failed to initialize history storage: %v\n", err)
		storage = nil
	}

	// MCP初期化（設定で制御）
	mcpManager := mcp.NewManager()
	if cfg.MCP.Enabled && (!headless || cfg.MCP.Headless) {
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

	// Function Calling 対応プロバイダーは詳細なツール定義を送信するため、
	// System Prompt からツール説明を除去（重複回避、トークン節約）
	// 注: LSP/MCP プロンプト追加より先に実行（後から追加する内容は削除されない）
	if provider.IsFunctionCallingEnabled() {
		systemPrompt = removeToolsSection(systemPrompt)
	}

	// LSP初期化
	var lspClient *lsp.Client
	cfg = config.GetGlobalConfig()
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
			// LSPドキュメントはSystemPromptのWorkflow Rulesに統合済み
		}
	}

	// MCPProviderインターフェースを実装するプロバイダーにMCPツールを設定
	// （Function Calling経由で呼び出し可能にする）
	if len(mcpManager.GetTools()) > 0 {
		// MCP ツール設定
		if mcpProvider, ok := provider.(api.MCPProvider); ok {
			debug := os.Getenv("XELYON_DEBUG_GEMINI") == "1"
			var mcpTools []api.ToolDefinition
			for _, t := range mcpManager.GetTools() {
				// ツール名: mcp_{serverName}_{toolName}
				name := fmt.Sprintf("mcp_%s_%s", sanitizeToolName(t.ServerName), sanitizeToolName(t.Name))
				// デバッグログ
				if debug {
					fmt.Fprintf(os.Stderr, "[DEBUG Gemini] MCP tool registered: %s\n", name)
				}
				// InputSchema を map[string]interface{} に変換
				var params map[string]interface{}
				if len(t.InputSchema) > 0 {
					if err := json.Unmarshal(t.InputSchema, &params); err != nil {
						// 変換失敗時は空のパラメータを使用
						params = nil
					}
				}
				mcpTools = append(mcpTools, api.ToolDefinition{
					Name:        name,
					Description: t.Description,
					Parameters:  params,
				})
			}
			mcpProvider.SetMCPTools(mcpTools)
		}

		// MCP ツール設定（OpenAI用）
		if openaiMCPProvider, ok := provider.(api.MCPProvider); ok {
			debug := os.Getenv("XELYON_DEBUG_OPENAI") == "1"
			var mcpFunctions []api.ToolDefinition
			for _, t := range mcpManager.GetTools() {
				name := fmt.Sprintf("mcp_%s_%s", sanitizeToolName(t.ServerName), sanitizeToolName(t.Name))
				fn := api.ConvertMCPToolToToolDefinition(name, t.Description, t.InputSchema)
				mcpFunctions = append(mcpFunctions, fn)

				// デバッグログ
				if debug {
					fmt.Fprintf(os.Stderr, "[DEBUG OpenAI] MCP tool registered: %s\n", name)
				}
			}
			openaiMCPProvider.SetMCPTools(mcpFunctions)
		}
	}

	// プロバイダー別プレフィックスを冒頭に注入
	systemPrompt = prompt.BuildProviderSystemPrompt(systemPrompt, provider.Name())

	// ToolCache 初期化
	toolCache := NewToolCache()
	tools.GlobalToolCache = toolCache

	// ReadTracker リセット（新セッション開始時）
	tools.GlobalReadTracker.Reset()

	// Agent を作成
	agent := &Agent{
		Model:           model,
		CurrentModel:    model,
		CurrentProvider: provider,
		ProviderName:    strings.ToLower(provider.Name()),
		History:         []api.Message{},
		session:         history.NewSession(model),
		storage:         storage,
		changeStack:     []tools.FileChange{},
		changeStorage:   changeStorage,
		mcpManager:      mcpManager,
		lspClient:       lspClient,
		SystemPrompt:    systemPrompt,
		Stats:           NewSessionStats(strings.ToLower(provider.Name()), model),
		lastOutputs:     []string{},
		ToolCache:       toolCache,
	}

	// Usage callback を設定（プロバイダーがサポートしている場合）
	if reporter, ok := provider.(api.UsageReporter); ok {
		reporter.SetUsageCallback(func(u api.Usage) {
			agent.statsMu.Lock()
			defer agent.statsMu.Unlock()
			agent.Stats.AddUsage(u)
		})
	}

	return agent
}

// removeToolsSection は System Prompt から ## Available Tools セクションを除去
//
// 削除範囲: "## Available Tools" 〜 "## Workflow Rules" の直前まで
// Gemini/OpenAI は Function Calling で詳細なツール定義を受け取るため、重複を避ける
//
// 注意: この範囲に「使い方」の説明を書くと消える！
// - ツール定義（File Operations, Git Operations など）→ ここに書く（削除 OK）
// - 使い方（Tool Usage Priority, Efficient Investigation）→ Workflow Rules に書く
// - LSP/MCP プロンプト → 別ファイル（lsp.go, mcp.go）で後から追加
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
