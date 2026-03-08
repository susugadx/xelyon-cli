package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
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
	toolsdev "github.com/susugadx/xelyon-cli/internal/tools/dev"
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
	Runtime              *AgentRuntime
	History              []api.Message
	SystemPrompt         string
	session              *history.Session
	storage              *history.Storage
	changeStack          []tools.FileChange
	taskChangeOffset     int                    // タスク開始時の changeStack 長（タスク単位のサマリー表示用）
	changeStorage        *history.ChangeStorage // 永続的変更履歴
	mcpManager           *mcp.Manager
	lspClient            *lsp.Client        // LSPクライアント
	AutoApprove          bool               // --auto-approve フラグ
	Stats                *SessionStats      // セッション統計情報
	lastOutputs          []string           // 最後のAI出力履歴（最大10件）
	cancelFunc           context.CancelFunc // 現在のAPI呼び出しをキャンセルするための関数
	strReplaceErrorCount int                // str_replace連続エラーカウント（old_str not found）
	PlanModeEnabled      bool               // Plan Mode ON/OFF（デフォルト: false）
	ToolCache            *ToolCache         // ツール結果キャッシュ（read_file, list_dir）
	taskBaseCommitHash   string             // タスク開始時のHEADコミットハッシュ（completion hook の diff 空チェック判定用）
	status               statusHolder

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

func (a *Agent) setPromptReader(reader *ui.MultilineReader) {
	if a == nil {
		return
	}
	a.ui().SetPromptReader(reader)
}

func (a *Agent) output() io.Writer {
	if a == nil {
		return ui.DefaultRuntime().Output()
	}
	return a.ui().Output()
}

func (a *Agent) errorOutput() io.Writer {
	if a == nil {
		return ui.DefaultRuntime().ErrorOutput()
	}
	return a.ui().ErrorOutput()
}

// NewAgent は新しいAgentを作成
func NewAgent(model string, provider api.Provider, headless bool) *Agent {
	return NewAgentWithRuntime(model, provider, headless, nil)
}

// NewAgentWithRuntime は runtime を指定して新しい Agent を作成する。
func NewAgentWithRuntime(model string, provider api.Provider, headless bool, runtime *AgentRuntime) *Agent {
	runtime = normalizeAgentRuntime(runtime)
	api.ApplyRuntimeConfig(provider, runtime.effectiveConfig())
	runtimeUI := runtime.effectiveUI()
	out := runtimeUI.Output()
	errOut := runtimeUI.ErrorOutput()

	// 言語設定を適用
	cfg := runtime.effectiveConfig()
	if cfg.General.Language != "" {
		i18n.SetLang(cfg.General.Language)
	}

	// 古いartifactファイルを削除
	toolsdev.CleanupArtifacts()

	storage, err := history.NewStorage()
	if err != nil {
		red.Fprintf(out, "Warning: Failed to initialize history storage: %v\n", err)
		storage = nil
	}

	// MCP初期化（設定と環境変数で制御）
	mcpManager := mcp.NewManager()
	mcpManager.SetOutput(out)
	if cfg.MCP.Enabled && (!headless || cfg.MCP.Headless) && os.Getenv("XELYON_DISABLE_MCP") != "1" {
		if err := mcpManager.LoadConfig(); err != nil {
			yellow.Fprintf(out, "Warning: Failed to load MCP config: %v\n", err)
		}

		ctx := context.Background()
		if err := mcpManager.Connect(ctx); err != nil {
			yellow.Fprintf(out, "Warning: MCP connection error: %v\n", err)
		}

		// MCPツールをTool Registryに登録
		if len(mcpManager.GetTools()) > 0 {
			mcpManager.RegisterToToolRegistry(runtime.effectiveRegistry())
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
		yellow.Fprintf(out, "Warning: Failed to initialize change storage: %v\n", err)
		changeStorage = nil
	}

	// LSP初期化
	var lspClient *lsp.Client
	cfg = runtime.effectiveConfig()
	if cfg.LSP.Enabled {
		cwd, err := os.Getwd()
		if err == nil {
			lspClient = lsp.NewClient(cwd)
			lspClient.SetOutput(out)
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
					_, _ = fmt.Fprintf(errOut, "[DEBUG Gemini] MCP tool registered: %s\n", name)
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
					_, _ = fmt.Fprintf(errOut, "[DEBUG OpenAI] MCP tool registered: %s\n", name)
				}
			}
			openaiMCPProvider.SetMCPTools(mcpFunctions)
		}
	}

	// プロバイダー別プレフィックスを Workflow Rules の直前に注入
	systemPrompt = prompt.BuildProviderSystemPrompt(systemPrompt, provider.Name(), model)

	// ToolCache 初期化（ディスクから復元）
	toolCache := runtime.effectiveToolCache()

	// Agent を作成
	agent := &Agent{
		Model:           model,
		CurrentModel:    model,
		CurrentProvider: provider,
		ProviderName:    strings.ToLower(provider.Name()),
		Runtime:         runtime,
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
		status:          statusHolder{status: defaultStatus()},
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

// cleanupHook はテスト用フック（非nil時にCleanupから呼ばれる）
var cleanupHook func()

// syncResponseIDToSession はプロバイダーの ResponseID をセッションに同期する（保存前に呼ぶ）
func (a *Agent) syncResponseIDToSession() {
	if a.session == nil {
		return
	}
	if ridProvider, ok := a.CurrentProvider.(ResponseIDCapable); ok {
		if ridProvider.HasCachedResponseID() {
			a.session.ResponseID = ridProvider.GetResponseID()
		}
	}
}

// Cleanup はエージェントのリソースをクリーンアップ
func (a *Agent) Cleanup() {
	if cleanupHook != nil {
		cleanupHook()
	}
	if a.mcpManager != nil {
		a.mcpManager.Close()
	}
	// LSPクリーンアップ
	if a.lspClient != nil {
		a.lspClient.Close()
	}
	// ToolCache 永続化
	if a.ToolCache != nil {
		if err := a.ToolCache.Save(); err != nil {
			yellow.Fprintf(a.output(), "Warning: Failed to save tool cache: %v\n", err)
		}
	}
	// セッション保存
	if a.storage != nil && a.session != nil {
		a.syncResponseIDToSession()
		if err := a.storage.Save(a.session); err != nil {
			yellow.Fprintf(a.output(), "Warning: Failed to save session: %v\n", err)
		}
	}
}

// GetLSPClient はLSPクライアントを返す（コマンド用）
func (a *Agent) GetLSPClient() *lsp.Client {
	return a.lspClient
}

// deduplicateToolResult は同一内容の tool result を参照文字列に差し替える（履歴用）。
// 重複の場合は短い参照文字列を返し、新規の場合は content をそのまま返す。
func (a *Agent) deduplicateToolResult(toolName, content string) string {
	if a.ToolCache == nil {
		return content
	}
	turn := countUserTurns(a.History)
	if ref := a.ToolCache.DeduplicateResult(toolName, content, turn); ref != "" {
		a.addOptimizationMetrics(OptimizationMetrics{
			DeduplicateCount:       1,
			DeduplicateTokensSaved: len(content) / 4,
		})
		return ref
	}
	return content
}

// countUserTurns は History 内の user メッセージ数を返す
func countUserTurns(history []api.Message) int {
	count := 0
	for _, msg := range history {
		if msg.Role == "user" {
			count++
		}
	}
	return count
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

func (a *Agent) addOptimizationMetrics(metrics OptimizationMetrics) {
	a.statsMu.Lock()
	defer a.statsMu.Unlock()
	if a.Stats != nil {
		a.Stats.Optimizations.add(metrics)
	}
}

func (a *Agent) addCompactionMetrics(metrics CompactionMetrics) {
	a.statsMu.Lock()
	defer a.statsMu.Unlock()
	if a.Stats != nil {
		a.Stats.Optimizations.addCompaction(metrics)
	}
}

func (a *Agent) recordToolResultOptimizations(toolName, result string) {
	if toolName == "read_file" && strings.Contains(result, "Use start_line/end_line to read function body") {
		a.addOptimizationMetrics(OptimizationMetrics{OutlineFirstCount: 1})
	}
}
