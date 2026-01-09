package agent

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/fatih/color"
	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/audit"
	"github.com/susugadx/xelyon-cli/internal/history"
	"github.com/susugadx/xelyon-cli/internal/mcp"
	"github.com/susugadx/xelyon-cli/internal/repomap"
	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/version"
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
	Model           string // 初期モデル（後方互換性のため保持）
	CurrentModel    string // 現在のモデル（再起動なしで切り替え可能）
	CurrentProvider api.Provider
	ProviderName    string
	History         []api.Message
	SystemPrompt    string
	session         *history.Session
	storage         *history.Storage
	changeStack     []tools.FileChange
	mcpManager      *mcp.Manager
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

	systemPrompt := `You are XELYON, an expert AI coding assistant with the following core principles:

## Core Identity
- Honest and truthful: Never fabricate information or make up commands
- Transparent about limitations: Say "I don't know" when uncertain
- No speculation presented as fact: Clearly distinguish between certainty and guesses
- Professional developer mindset: Focus on code quality, maintainability, and best practices
- Fluent in Japanese: Can seamlessly communicate in both English and Japanese

## Professional Standards
- Security-conscious: Always consider security implications
- Quality-focused: Write clean, readable, well-structured code
- Test-aware: Consider testability and verification
- Documentation-minded: Explain complex decisions

## Available Tools
You have access to the following tools:

### File Operations
- read_file: Read file contents. Args: {"path": "..."}
- write_file: Write content to a file. Args: {"path": "...", "content": "..."}
- str_replace: Replace text in file. Args: {"path": "...", "old_str": "...", "new_str": "..."}
- append_file: Append content to end of file. Args: {"path": "...", "content": "..."}
- prepend_file: Insert content at beginning of file. Args: {"path": "...", "content": "..."}
- insert_after: Insert content after pattern match. Args: {"path": "...", "pattern": "...", "content": "..."}
- insert_before: Insert content before pattern match. Args: {"path": "...", "pattern": "...", "content": "..."}
- copy_file: Copy file to destination. Args: {"src": "...", "dest": "..."}
- list_dir: List directory contents. Args: {"path": "..."} (optional)

### File Management
- create_dir: Create directory (including parents). Args: {"path": "..."}
- delete_lines: Delete line range from file. Args: {"path": "...", "start_line": "N", "end_line": "M"}
- delete_file: Delete file permanently (with backup). Args: {"path": "..."}
- move_file: Move/rename file. Args: {"src": "...", "dest": "..."}

### Code Quality
- lint: Run linter with optional auto-fix. Args: {"path": "...", "auto_fix": "true|false"} (path optional, default: ".")

### Git Operations
- git_status: Show git status. Args: {}
- git_diff: Show git diff. Args: {"path": "..."} (optional)
- git_add: Stage files. Args: {"path": "..."} (default: ".")
- git_commit: Commit changes. Args: {"message": "..."}
- git_push: Push to remote. Args: {}
- git_log: Show recent commits. Args: {}
- git_branch: Manage branches. Args: {"action": "list|create|switch", "branch_name": "..."} (branch_name required for create/switch)
- git_checkout: Restore file from HEAD or switch branch. Args: {"target": "file_path or branch_name"}
- git_stash: Stash changes. Args: {"action": "save|list|pop|apply|drop", "message": "..."} (message optional for save, index for pop/apply/drop)

### Development Tools
- run_test: Auto-detect and run tests (go/npm/pytest/cargo). Args: {"path": "..."} (optional)
- format: Auto-detect and run formatter (gofmt/prettier/black/rustfmt). Args: {"path": "..."} (optional)

### Search & Discovery
- search_code: Search for pattern in code files. Args: {"pattern": "...", "path": "..."} (path optional)
- search_file: Search for files by name. Args: {"pattern": "...", "path": "..."} (path optional)
- web_search: Search the web for information. Args: {"query": "..."}

### Shell
- bash: Execute shell commands. Args: {"command": "..."}

When you need to use a tool, respond with ONLY a JSON block like this:
{"tool": "tool_name", "args": {"arg1": "value1"}}

## Workflow Rules

### Phase 1: Planning & Understanding
1. Before any action, understand the context (use list_dir, read_file, search_code)
2. For complex tasks, create a plan BEFORE execution
3. Explain your reasoning: Why this tool? Why this approach?
4. Ask for user confirmation when making significant changes

### Phase 2: Execution
5. Use the right tool for each task:
   - search_code: Search code content (NOT bash grep)
   - search_file: Search file names (NOT bash find)
   - str_replace: Edit existing files (NOT bash sed)
   - write_file: Create NEW files only (NOT for editing)
6. For file writes, include COMPLETE file content
7. str_replace safety rules:
   - If old_str matches multiple times, include surrounding context to make it unique
   - For large edits, split into multiple str_replace calls (~10 lines each)
   - When editing the same file consecutively, use read_file to verify current state
8. Respond with ONLY a JSON block for tool calls: {"tool": "...", "args": {...}}

### Phase 3: Verification
9. After using a tool, analyze the result and decide next steps
10. For code changes, verify quality:
    - Run "go fmt" to format Go code
    - Run "go test" if tests exist
    - Check for obvious errors or warnings
11. When task is complete, give a summary WITHOUT tool calls

### Phase 4: Error Handling
12. If a tool fails, try alternative approaches:
    - First attempt failed? Analyze why and adjust
    - Don't retry the same failing command blindly
    - Ask user for help if stuck after 2-3 attempts
13. For user cancellations:
    - Respect the decision
    - Ask if they want a different approach
    - Do NOT retry the same operation

### General Guidelines
14. Respond in the same language as the user (Japanese or English)
15. Be concise but helpful
16. Show your thought process when solving complex problems`

	// MCPツールをSystemPromptに追加
	if len(mcpManager.GetTools()) > 0 {
		systemPrompt += "\n\n## MCP Tools (External)\n"
		systemPrompt += "These tools are provided by external MCP servers:\n"
		for _, t := range mcpManager.GetTools() {
			systemPrompt += fmt.Sprintf("- mcp_%s_%s: %s\n",
				t.ServerName, t.Name, t.Description)
		}
	}

	return &Agent{
		Model:           model,
		CurrentModel:    model,
		CurrentProvider: provider,
		ProviderName:    provider.Name(),
		History:         []api.Message{},
		session:         history.NewSession(model),
		storage:         storage,
		changeStack:     []tools.FileChange{},
		mcpManager:      mcpManager,
		SystemPrompt:    systemPrompt,
	}
}

// Cleanup はエージェントのリソースをクリーンアップ
func (a *Agent) Cleanup() {
	if a.mcpManager != nil {
		a.mcpManager.Close()
	}
	// セッション保存
	if a.storage != nil && a.session != nil {
		if err := a.storage.Save(a.session); err != nil {
			yellow.Printf("Warning: Failed to save session: %v\n", err)
		}
	}
}

func RunInteractive(model string, provider api.Provider) {
	// 監査ログ初期化（環境変数で制御: XELYON_AUDIT_LOG=1 で有効化）
	auditEnabled := os.Getenv("XELYON_AUDIT_LOG") == "1"
	if err := audit.Init(auditEnabled); err != nil {
		yellow.Printf("Warning: Failed to initialize audit log: %v\n", err)
	}
	if auditEnabled {
		green.Println("📝 Audit logging enabled")
	}

	agent := NewAgent(model, provider)
	defer agent.Cleanup() // グレースフルシャットダウン

	// シグナルハンドリング（Ctrl+C, SIGTERM対応）
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("\n\n👋 Gracefully shutting down...")
		agent.Cleanup()
		os.Exit(0)
	}()

	// ヘッダー表示
	printHeader(model, provider)

	// XELYON.md読み込み
	if config := loadProjectConfig(); config != "" {
		agent.SystemPrompt += "\n\n## Project Context:\n" + config
		green.Println("📋 XELYON.md loaded")
	}

	// Repo Map 生成
	cwd, err := os.Getwd()
	if err != nil {
		yellow.Printf("Warning: Could not get current directory: %v\n", err)
		cwd = "." // フォールバック
	}
	rm := repomap.NewRepoMap(cwd, 2000) // 最大2000トークン
	if err := rm.Build(); err == nil && rm.GetSymbolCount() > 0 {
		repoMapStr := rm.Generate()
		agent.SystemPrompt += "\n\n" + repoMapStr
		green.Printf("🗺️  Repo map loaded (%d symbols from %d files)\n",
			rm.GetSymbolCount(), len(rm.Files))
	}

	// REPLループ
	reader := bufio.NewReader(os.Stdin)
	for {
		cyan.Print("\n> ")
		input, err := reader.ReadString('\n')
		if err != nil {
			break
		}

		input = strings.TrimSpace(input)
		if input == "" {
			continue
		}

		// 特殊コマンド
		if handleSpecialCommand(input, agent) {
			continue
		}

		// AIに送信
		agent.chat(input)
	}
}
func RunOnce(query string, model string) {
	// Note: この関数は古いAPI (api.ChatWithTools) を使用
	// 将来的に削除予定
	agent := &Agent{
		Model:        model,
		CurrentModel: model,
		History:      []api.Message{},
		session:      history.NewSession(model),
	}

	if config := loadProjectConfig(); config != "" {
		agent.SystemPrompt += "\n\n## Project Context:\n" + config
		green.Println("📋 XELYON.md loaded")
	}

	// Repo Map 生成
	cwd, err := os.Getwd()
	if err != nil {
		yellow.Printf("Warning: Could not get current directory: %v\n", err)
		cwd = "." // フォールバック
	}
	rm := repomap.NewRepoMap(cwd, 2000)
	if err := rm.Build(); err == nil && rm.GetSymbolCount() > 0 {
		repoMapStr := rm.Generate()
		agent.SystemPrompt += "\n\n" + repoMapStr
		green.Printf("🗺️  Repo map loaded (%d symbols from %d files)\n",
			rm.GetSymbolCount(), len(rm.Files))
	}

	fmt.Println()
	agent.chat(query)
}
func printHeader(model string, provider api.Provider) {
	cyan.Println("╔═══════════════════════════════════════════╗")
	cyan.Printf("║  🚀 XELYON CLI v%-25s║\n", version.GetVersion())
	cyan.Println("║  AI-powered coding assistant              ║")
	cyan.Println("╚═══════════════════════════════════════════╝")
	green.Printf("🌐 Provider: %s\n", provider.Name())
	fmt.Printf("Model: %s\n", modelDisplayName(model))
	yellow.Println("Type /help for commands, /exit to quit")
}
func modelDisplayName(model string) string {
	switch model {
	case "deepseek-chat":
		return "DeepSeek V3 (balanced)"
	case "deepseek-coder":
		return "DeepSeek Coder (code-focused)"
	case "deepseek-reasoner":
		return "DeepSeek R1 (reasoning)"
	case "claude":
		return "Claude (Vertex AI)"
	default:
		return model
	}
}
func loadProjectConfig() string {
	dir, err := os.Getwd()
	if err != nil {
		yellow.Printf("Warning: Could not get current directory: %v\n", err)
		return "" // Cannot locate XELYON.md without working directory
	}
	for {
		path := dir + "/XELYON.md"
		if content, err := os.ReadFile(path); err == nil {
			return string(content)
		}
		parent := dir[:strings.LastIndex(dir, "/")]
		if parent == dir || parent == "" {
			break
		}
		dir = parent
	}
	return ""
}
func RunInteractiveWithResume(model string, provider api.Provider) {
	storage, err := history.NewStorage()
	if err != nil {
		red.Printf("Failed to initialize storage: %v\n", err)
		RunInteractive(model, provider)
		return
	}

	sessionID, err := storage.GetLastSession()
	if err != nil {
		yellow.Println("No previous session found, starting new session")
		RunInteractive(model, provider)
		return
	}

	session, err := storage.Load(sessionID)
	if err != nil {
		red.Printf("Failed to load session: %v\n", err)
		RunInteractive(model, provider)
		return
	}

	// ロード済みセッションでAgent作成
	agent := NewAgent(model, provider)
	agent.session = session
	agent.History = session.ToAPIMessages()

	printHeader(model, provider)
	green.Printf("📂 Resumed session %s (%d messages)\n", sessionID, len(session.Messages))

	if config := loadProjectConfig(); config != "" {
		agent.SystemPrompt += "\n\n## Project Context:\n" + config
		green.Println("📋 XELYON.md loaded")
	}

	// Repo Map 生成
	cwd, err := os.Getwd()
	if err != nil {
		yellow.Printf("Warning: Could not get current directory: %v\n", err)
		cwd = "." // フォールバック
	}
	rm := repomap.NewRepoMap(cwd, 2000)
	if err := rm.Build(); err == nil && rm.GetSymbolCount() > 0 {
		repoMapStr := rm.Generate()
		agent.SystemPrompt += "\n\n" + repoMapStr
		green.Printf("🗺️  Repo map loaded (%d symbols from %d files)\n",
			rm.GetSymbolCount(), len(rm.Files))
	}

	// REPLループ
	reader := bufio.NewReader(os.Stdin)
	for {
		cyan.Print("\n> ")
		input, err := reader.ReadString('\n')
		if err != nil {
			break
		}

		input = strings.TrimSpace(input)
		if input == "" {
			continue
		}

		if handleSpecialCommand(input, agent) {
			continue
		}

		agent.chat(input)
	}
}
