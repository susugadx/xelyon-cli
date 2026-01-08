package agent

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
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

// RunInteractive は対話モードを開始
func RunInteractive(model string, provider api.Provider) {
	agent := NewAgent(model, provider)

	// ヘッダー表示
	printHeader(model, provider)

	// XELYON.md読み込み
	if config := loadProjectConfig(); config != "" {
		agent.SystemPrompt += "\n\n## Project Context:\n" + config
		green.Println("📋 XELYON.md loaded")
	}

	// Repo Map 生成
	cwd, _ := os.Getwd()
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

// RunOnce はワンショットモードを実行（後方互換用、Providerは未使用）
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
	cwd, _ := os.Getwd()
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

// isSameToolCall は2つのToolCallが同じかを判定
func isSameToolCall(tc1, tc2 *tools.ToolCall) bool {
	if tc1 == nil || tc2 == nil {
		return false
	}
	if tc1.Tool != tc2.Tool {
		return false
	}

	// args を JSON 化して比較
	args1, _ := json.Marshal(tc1.Args)
	args2, _ := json.Marshal(tc2.Args)
	return string(args1) == string(args2)
}

// chat はAIと対話する
func (a *Agent) chat(input string) {
	// 履歴に追加
	a.History = append(a.History, api.Message{
		Role:    "user",
		Content: input,
	})

	// セッションに保存
	if a.session != nil {
		a.session.AddMessage("user", input, a.CurrentModel)
	}

	// AIに送信（ツール実行ループ）
	const maxIterations = 10 // 無限ループ防止
	var lastToolCall *tools.ToolCall
	var sameCallCount int
	var loopCount int
	var normalExit bool // 正常終了フラグ

	for i := 0; i < maxIterations; i++ {
		loopCount = i + 1 // 実際のループ回数を記録
		// API呼び出しのリトライ処理
		const maxRetries = 2
		var response string
		var err error

		for retry := 0; retry < maxRetries; retry++ {
			ctx := context.Background()
			response, err = a.CurrentProvider.ChatWithTools(ctx, a.SystemPrompt, a.History, a.CurrentModel)
			if err == nil {
				break // 成功
			}

			if retry < maxRetries-1 {
				yellow.Printf("⚠️  API error, retrying (%d/%d)...\n", retry+1, maxRetries-1)
				time.Sleep(time.Second * time.Duration(retry+1)) // 指数バックオフ的な待機
			}
		}

		if err != nil {
			red.Printf("エラー: %v\n", err)
			yellow.Println("API呼び出しに失敗しました。ネットワーク接続を確認してください。")
			return
		}

		// ツール呼び出しをチェック
		toolCall := tools.ParseToolCall(response)
		if toolCall != nil {
			// 同じツール呼び出しをカウント
			if isSameToolCall(toolCall, lastToolCall) {
				sameCallCount++
				if sameCallCount >= 3 {
					yellow.Printf("⚠️  Warning: Same tool call repeated %d times, stopping to prevent infinite loop\n", sameCallCount)
					yellow.Printf("   Tool: %s\n", toolCall.Tool)

					// AI に警告メッセージを返す
					a.History = append(a.History, api.Message{
						Role:    "user",
						Content: "[SYSTEM WARNING] The same tool call was repeated 3 times. Please try a different approach or ask the user for clarification.",
					})
					continue // 次のイテレーションで AI に考え直させる
				}
			} else {
				// 異なるツール呼び出し → カウンターリセット
				sameCallCount = 1
				lastToolCall = toolCall
			}

			// 結果を履歴に追加
			a.History = append(a.History, api.Message{
				Role:    "assistant",
				Content: response,
			})

			// ツール実行
			result, change := tools.Execute(toolCall)

			// 変更履歴を保存
			if change != nil {
				a.changeStack = append(a.changeStack, *change)
				// 最大10件まで保持
				if len(a.changeStack) > 10 {
					a.changeStack = a.changeStack[1:]
				}

				// Goファイル変更時の自動検証提案
				if verifyResult := ShouldVerify(change.FilePath); verifyResult.NeedsVerify {
					a.suggestVerification(change.FilePath, verifyResult)
				}
			}

			// 結果を履歴に追加
			a.History = append(a.History, api.Message{
				Role:    "user",
				Content: fmt.Sprintf("[Tool Result for %s]\n%s", toolCall.Tool, result),
			})

			// 続けて処理
			fmt.Println()
			continue
		}

		// 通常の回答（ツール呼び出しなし）
		a.History = append(a.History, api.Message{
			Role:    "assistant",
			Content: response,
		})

		// セッションに保存
		if a.session != nil {
			a.session.AddMessage("assistant", response, a.CurrentModel)
			if a.storage != nil {
				if err := a.storage.Save(a.session); err != nil {
					// サイレント失敗（ユーザー体験を妨げない）
				}
			}
		}

		normalExit = true // 正常終了
		break
	}

	// maxIterations に到達した場合の警告（正常終了でない場合のみ）
	if !normalExit && loopCount >= maxIterations {
		yellow.Printf("⚠️  Warning: Maximum iterations (%d) reached\n", maxIterations)
		yellow.Println("The task may be too complex or the AI is stuck in a loop.")
		yellow.Println("Consider breaking down the task into smaller steps.")
	}
}

// handleSpecialCommand は特殊コマンドを処理
func handleSpecialCommand(input string, agent *Agent) bool {
	parts := strings.Fields(input)
	if len(parts) == 0 {
		return false
	}

	cmd := strings.ToLower(parts[0])
	args := parts[1:]

	switch cmd {
	case "/save":
		return handleSaveCommand(agent)
	case "/load":
		return handleLoadCommand(agent, args)
	case "/sessions":
		return handleSessionsCommand(agent)
	case "/undo":
		return handleUndoCommand(agent)
	case "/config":
		return handleConfigCommand(args)
	case "/exit", "/quit", "/q":
		yellow.Println("👋 See you!")
		os.Exit(0)
	case "/clear":
		agent.History = []api.Message{}
		green.Println("🗑️  History cleared")
		return true
	case "/history":
		fmt.Printf("📜 %d messages in history\n", len(agent.History))
		for i, msg := range agent.History {
			role := "👤"
			if msg.Role == "assistant" {
				role = "🤖"
			}
			preview := msg.Content
			if len(preview) > 50 {
				preview = preview[:50] + "..."
			}
			fmt.Printf("  %d. %s %s\n", i+1, role, preview)
		}
		return true
	case "/help":
		printHelp()
		return true
	case "/model":
		return handleModelCommand(agent, args)
	case "/version":
		cyan.Printf("🚀 XELYON CLI v%s\n", version.GetVersion())
		return true
	case "/repomap":
		return handleRepoMapCommand()
	}
	return false
}

// handleSaveCommand はセッション保存を処理
func handleSaveCommand(agent *Agent) bool {
	if agent.storage == nil {
		red.Println("History storage not available")
		return true
	}

	if err := agent.storage.Save(agent.session); err != nil {
		red.Printf("Failed to save session: %v\n", err)
		return true
	}

	green.Printf("💾 Session saved: %s\n", agent.session.ID)
	return true
}

// handleLoadCommand はセッション読み込みを処理
func handleLoadCommand(agent *Agent, args []string) bool {
	if agent.storage == nil {
		red.Println("History storage not available")
		return true
	}

	sessionID := ""
	if len(args) > 0 {
		sessionID = args[0]
	} else {
		lastID, err := agent.storage.GetLastSession()
		if err != nil {
			red.Printf("No sessions found: %v\n", err)
			return true
		}
		sessionID = lastID
	}

	session, err := agent.storage.Load(sessionID)
	if err != nil {
		red.Printf("Failed to load session: %v\n", err)
		return true
	}

	// セッション置き換え
	agent.session = session
	agent.History = session.ToAPIMessages()

	green.Printf("📂 Loaded session %s (%d messages)\n", sessionID, len(session.Messages))
	return true
}

// handleSessionsCommand はセッション一覧を表示
func handleSessionsCommand(agent *Agent) bool {
	if agent.storage == nil {
		red.Println("History storage not available")
		return true
	}

	sessions, err := agent.storage.ListSessions()
	if err != nil {
		red.Printf("Failed to list sessions: %v\n", err)
		return true
	}

	if len(sessions) == 0 {
		yellow.Println("No sessions found")
		return true
	}

	cyan.Println("\n📚 Recent Sessions:")
	for i, s := range sessions {
		if i >= 10 {
			break
		}

		timeStr := s.LastModified.Format("2006-01-02 15:04")
		preview := s.Preview
		if len(preview) > 60 {
			preview = preview[:60] + "..."
		}

		fmt.Printf("  [%s] %s - %s (%d msgs)\n",
			s.ID, timeStr, preview, s.MessageCount)
	}
	fmt.Println()
	return true
}

// handleUndoCommand は直前のファイル変更を取り消す
func handleUndoCommand(agent *Agent) bool {
	if len(agent.changeStack) == 0 {
		yellow.Println("No changes to undo")
		return true
	}

	// 最後の変更を取得
	lastChange := agent.changeStack[len(agent.changeStack)-1]

	// バックアップが存在しない場合
	if lastChange.BackupPath == "" {
		red.Println("No backup available for last change")
		return true
	}

	// バックアップファイルを確認
	if _, err := os.Stat(lastChange.BackupPath); os.IsNotExist(err) {
		red.Printf("Backup file not found: %s\n", lastChange.BackupPath)
		return true
	}

	// 確認プロンプト
	yellow.Printf("Undo last change?\n")
	fmt.Printf("  File: %s\n", lastChange.FilePath)
	fmt.Printf("  Tool: %s\n", lastChange.Tool)
	fmt.Printf("  Time: %s\n", lastChange.Timestamp.Format("2006-01-02 15:04:05"))
	yellow.Print("Continue? (y/n): ")

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		red.Printf("Failed to read input: %v\n", err)
		return true
	}

	input = strings.TrimSpace(strings.ToLower(input))
	if input != "y" && input != "yes" {
		yellow.Println("Undo cancelled")
		return true
	}

	// バックアップから復元
	backupContent, err := os.ReadFile(lastChange.BackupPath)
	if err != nil {
		red.Printf("Failed to read backup: %v\n", err)
		return true
	}

	if err := os.WriteFile(lastChange.FilePath, backupContent, 0644); err != nil {
		red.Printf("Failed to restore file: %v\n", err)
		return true
	}

	// スタックから削除
	agent.changeStack = agent.changeStack[:len(agent.changeStack)-1]

	green.Printf("✅ Undone: %s\n", lastChange.Description)
	green.Printf("   Restored from: %s\n", lastChange.BackupPath)
	return true
}

// handleModelCommand はモデルの表示・切り替えを処理
func handleModelCommand(agent *Agent, args []string) bool {
	// 引数なし → 現在のモデルとプロバイダーを表示
	if len(args) == 0 {
		fmt.Printf("🤖 Current model: %s\n", agent.CurrentModel)
		fmt.Printf("🌐 Provider: %s\n", agent.ProviderName)
		yellow.Println("\nUsage: /model <model-name>")
		yellow.Println("Enter any model name supported by your provider.")

		// Ollamaの場合だけインストール済みモデルを表示
		if agent.ProviderName == "ollama" {
			if ollamaProvider, ok := agent.CurrentProvider.(*api.OllamaProvider); ok {
				models, err := ollamaProvider.ListModels()
				if err != nil {
					yellow.Printf("\nWarning: Could not list Ollama models: %v\n", err)
				} else if len(models) > 0 {
					yellow.Println("\nInstalled Ollama models:")
					for _, model := range models {
						fmt.Printf("  - %s\n", model)
					}
				}
			}
		}
		return true
	}

	// /model <model-name> → モデル切り替え
	newModel := args[0]

	// モデルを切り替え
	oldModel := agent.CurrentModel
	agent.CurrentModel = newModel

	green.Printf("✅ Model switched: %s → %s\n", oldModel, newModel)

	// 設定ファイルにも保存
	cfg, err := config.LoadConfig()
	if err != nil {
		yellow.Printf("Warning: Failed to load config: %v\n", err)
		return true
	}

	cfg.DefaultModel = newModel
	if err := config.SaveConfig(cfg); err != nil {
		yellow.Printf("Warning: Failed to save config: %v\n", err)
		yellow.Println("Model switched for this session only")
		return true
	}

	green.Println("💾 Default model saved to config")
	return true
}

// handleConfigCommand は設定の表示・変更を処理
func handleConfigCommand(args []string) bool {
	cfg, err := config.LoadConfig()
	if err != nil {
		red.Printf("Failed to load config: %v\n", err)
		return true
	}

	// 引数なし → 現在の設定を表示
	if len(args) == 0 {
		cyan.Println("⚙️  Current Configuration:")
		fmt.Printf("  default_model: %s\n", cfg.DefaultModel)
		fmt.Printf("  default_provider: %s\n", cfg.DefaultProvider)
		yellow.Println("\nUsage: /config model <model-name>")
		yellow.Println("Enter any model name supported by your provider.")
		return true
	}

	// /config model <model-name> → モデル変更
	if len(args) >= 2 && args[0] == "model" {
		newModel := args[1]

		// 設定更新（バリデーションなし、任意のモデル名を受け付ける）
		cfg.DefaultModel = newModel
		if err := config.SaveConfig(cfg); err != nil {
			red.Printf("Failed to save config: %v\n", err)
			return true
		}

		green.Printf("✅ Default model updated to: %s\n", newModel)
		yellow.Println("Restart CLI for changes to take effect")
		return true
	}

	yellow.Println("Usage: /config [model <model-name>]")
	return true
}

// handleRepoMapCommand はRepo Mapを表示
func handleRepoMapCommand() bool {
	cwd, _ := os.Getwd()
	rm := repomap.NewRepoMap(cwd, 0) // 制限なし
	if err := rm.Build(); err != nil {
		red.Printf("Failed to build repo map: %v\n", err)
		return true
	}

	if rm.GetSymbolCount() == 0 {
		yellow.Println("No symbols found in current directory")
		return true
	}

	cyan.Printf("🗺️  Repository Map (%d symbols from %d files)\n\n",
		rm.GetSymbolCount(), len(rm.Files))
	fmt.Println(rm.Generate())
	return true
}

// printHeader はヘッダーを表示
func printHeader(model string, provider api.Provider) {
	cyan.Println("╔═══════════════════════════════════════════╗")
	cyan.Printf("║  🚀 XELYON CLI v%-25s║\n", version.GetVersion())
	cyan.Println("║  AI-powered coding assistant              ║")
	cyan.Println("╚═══════════════════════════════════════════╝")
	green.Printf("🌐 Provider: %s\n", provider.Name())
	fmt.Printf("Model: %s\n", modelDisplayName(model))
	yellow.Println("Type /help for commands, /exit to quit")
}

// printHelp はヘルプを表示
func printHelp() {
	fmt.Println(`Commands:
  /exit, /quit, /q  - Exit the CLI
  /clear            - Clear conversation history
  /history          - Show conversation history
  /save             - Save current session
  /load [id]        - Load session (or last if no ID)
  /sessions         - List recent sessions
  /undo             - Undo last file change (restore from .bak)
  /config           - Show/change configuration (e.g., /config model deepseek-coder)
  /model [name]     - Show current model or switch model without restart
  /repomap          - Show repository code structure map
  /version          - Show version information
  /help             - Show this help

Available tools (AI will use automatically):
  bash        - Execute shell commands
  read_file   - Read file contents
  write_file  - Write/create files (creates .bak backup)
  str_replace - Replace text in file (creates .bak backup)
  list_dir    - List directory contents
  git_*       - Git operations (status, diff, add, commit, push, log)
  search_code - Search in code files
  search_file - Search for files by name

Tips:
  - Just describe what you want in natural language
  - AI will ask confirmation for dangerous operations
  - Use Ctrl+C to cancel current operation
  - Use /undo to revert file changes (up to 10 recent changes)`)
}

// modelDisplayName はモデルの表示名を返す
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

// loadProjectConfig はXELYON.mdを探して読み込む
func loadProjectConfig() string {
	dir, _ := os.Getwd()
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

// RunInteractiveWithResume は最後のセッションを復元して起動
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
	cwd, _ := os.Getwd()
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

// suggestVerification はGoファイル変更後の検証を提案・実行
func (a *Agent) suggestVerification(filePath string, vr *VerifyResult) {
	if vr.FileType != "go" {
		return
	}

	// go.mod存在チェック
	if !CheckGoModExists() {
		return // Goプロジェクトじゃない
	}

	fmt.Println()
	yellow.Println("🔍 Go file changed. Run verification?")
	fmt.Printf("   File: %s\n", filePath)
	yellow.Print("   Run go fmt + go test? (y/n/f=fmt only): ")

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return
	}

	input = strings.TrimSpace(strings.ToLower(input))

	switch input {
	case "y", "yes":
		a.runVerification(filePath, true, true)
	case "f", "fmt":
		a.runVerification(filePath, true, false)
	default:
		yellow.Println("   Skipped verification")
	}
}

// runVerification は実際に検証を実行
func (a *Agent) runVerification(filePath string, runFmt, runTest bool) {
	if runFmt {
		cyan.Println("\n📝 Running go fmt...")
		output, err := RunGoFmt(filePath)
		if err != nil {
			red.Printf("   go fmt failed: %v\n", err)
		} else if output == "" {
			green.Println("   ✅ Already formatted")
		} else {
			green.Printf("   ✅ Formatted: %s\n", output)
		}
	}

	if runTest {
		cyan.Println("\n🧪 Running go test...")
		output, passed, _ := RunGoTest(filePath)

		// 出力が長い場合は省略
		lines := strings.Split(output, "\n")
		if len(lines) > 20 {
			for _, line := range lines[:10] {
				fmt.Println("   " + line)
			}
			yellow.Printf("   ... (%d lines omitted)\n", len(lines)-20)
			for _, line := range lines[len(lines)-10:] {
				fmt.Println("   " + line)
			}
		} else {
			for _, line := range lines {
				if line != "" {
					fmt.Println("   " + line)
				}
			}
		}

		if passed {
			green.Println("   ✅ All tests passed")
		} else {
			red.Println("   ❌ Tests failed")
			a.suggestRollback()
		}
	}
}

// suggestRollback はテスト失敗時にrollbackを提案
func (a *Agent) suggestRollback() {
	if len(a.changeStack) == 0 {
		return
	}

	yellow.Print("\n   Rollback the change? (y/n): ")

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return
	}

	input = strings.TrimSpace(strings.ToLower(input))
	if input == "y" || input == "yes" {
		handleUndoCommand(a)
	}
}
