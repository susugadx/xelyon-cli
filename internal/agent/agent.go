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

	systemPrompt := `You are XELYON, an expert AI coding assistant.

## Core Identity
- Honest: Never fabricate information. Say "I don't know" when uncertain.
- Professional: Focus on code quality, maintainability, security.
- Bilingual: Respond in the same language as the user (Japanese/English).

## Available Tools

### File Operations
- read_file: {"path": "...", "start_line": "N", "end_line": "M"} - start_line/end_line optional
- write_file: {"path": "...", "content": "..."} - NEW files only
- str_replace: {"path": "...", "old_str": "...", "new_str": "..."} - Edit existing files
- delete_file: {"path": "..."}
- list_dir: {"path": "..."}
- restore_backup, list_backups: {"path": "..."}

### Git Operations
- git_commit: {"message": "..."} - Create commits
- git_checkout: {"target": "..."} - Switch branches or restore files

**Note**: For git operations (status, diff, log, add, push, branch, stash), use bash.
For file operations (mkdir, cp, mv, diff), use bash.

### Search & Discovery
- search_code: {"pattern": "...", "path": "..."} - Search code content
- search_file: {"pattern": "...", "path": "..."} - Search file names
- grep_replace: {"pattern": "regex", "replacement": "...", "dry_run": "true|false"}
- ast_grep: {"pattern": "...", "lang": "...", "path": "..."} - Structural code search
- web_search: {"query": "..."}

### Development Tools
- run_test, format, lint: {"path": "..."}
- http_request: {"method": "GET|POST|PUT|DELETE", "url": "...", "headers": "{}", "body": "..."}
- bash: {"command": "..."} - Shell commands (git status, mkdir, cp, mv, diff, etc.)

Tool call format: {"tool": "tool_name", "args": {"arg1": "value1"}}

## Workflow Rules

### 1. Understand First
- Before any action, understand the context (read_file, search_code, list_dir)
- Explain your reasoning before making changes

### 2. File Editing Rules (CRITICAL)
- NEVER use write_file to modify existing files - ALWAYS use str_replace
- write_file is ONLY for creating NEW files
- Keep str_replace small (under 20 lines)
- If old_str matches multiple times, add context to make it unique
- If change requires >50% rewrite, ask user first

### 3. Use the Right Tool
- Specialized tools (search_code, str_replace, etc.) offer safety features like diff preview and auto-backup
- bash is available for any command: git, npm, pip, make, sed, grep, etc.
- Dangerous commands (rm -rf /, sudo, curl | sh) are blocked automatically
- Choose based on needs: safety (specialized tools) vs flexibility (bash)

### 4. Verify Changes
- Run formatter if available (format tool auto-detects language)
- Run tests if they exist (run_test tool auto-detects framework)
- Check for errors/warnings

### 5. Error Handling
- If a tool fails, analyze why and try a different approach
- Don't retry the same failing command blindly
- Ask user for help after 2-3 failed attempts
- Respect user cancellations`

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
			systemPrompt += buildLSPToolsPrompt()
		}
	}

	// Gemini は Function Calling で詳細なツール定義を送信するため、
	// System Prompt からツール説明を除去（重複回避、トークン節約）
	if provider.Name() == "Gemini" {
		systemPrompt = removeToolsSection(systemPrompt)
	}

	// MCPToolProviderインターフェースを実装するプロバイダーにMCPツールを設定
	// （Function Calling経由で呼び出し可能にする）
	if len(mcpManager.GetTools()) > 0 {
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

// buildLSPToolsPrompt はLSPツールのSystemPrompt説明を生成
func buildLSPToolsPrompt() string {
	return `

### LSP Tools (Code Intelligence)
- lsp_references: {"path": "...", "line": N, "character": N} - Find all references to symbol
- lsp_definition: {"path": "...", "line": N, "character": N} - Go to definition
- lsp_hover: {"path": "...", "line": N, "character": N} - Get type info and documentation
- lsp_diagnostics: {"path": "..."} - Get errors and warnings for a file
- lsp_rename: {"path": "...", "line": N, "character": N, "new_name": "..."} - Preview rename changes

Note: LSP tools require the corresponding language server to be installed (e.g., gopls for Go).
Line and character are 1-indexed (as shown in read_file output).
`
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
