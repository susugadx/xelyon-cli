package agent

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/history"
	"github.com/susugadx/xelyon-cli/internal/tools"
)

// 色定義
var (
	cyan    = color.New(color.FgCyan, color.Bold)
	green   = color.New(color.FgGreen)
	yellow  = color.New(color.FgYellow)
	red     = color.New(color.FgRed)
)

// Agent はCLIエージェント
type Agent struct {
	Model        string
	History      []api.Message
	SystemPrompt string
	session      *history.Session
	storage      *history.Storage
}

// NewAgent は新しいAgentを作成
func NewAgent(model string) *Agent {
	storage, err := history.NewStorage()
	if err != nil {
		red.Printf("Warning: Failed to initialize history storage: %v\n", err)
		storage = nil
	}

	return &Agent{
		Model:   model,
		History: []api.Message{},
		session: history.NewSession(model),
		storage: storage,
		SystemPrompt: `You are XELYON, an expert AI coding assistant.

You have access to the following tools:
- bash: Execute shell commands. Args: {"command": "..."}
- read_file: Read file contents. Args: {"path": "..."}
- write_file: Write content to a file. Args: {"path": "...", "content": "..."}
- str_replace: Replace text in file. Args: {"path": "...", "old_str": "...", "new_str": "..."}
- list_dir: List directory contents. Args: {"path": "..."} (optional, defaults to current dir)
- git_status: Show git status. Args: {}
- git_diff: Show git diff. Args: {"path": "..."} (optional)
- git_add: Stage files. Args: {"path": "..."} (default: ".")
- git_commit: Commit changes. Args: {"message": "..."}
- git_push: Push to remote. Args: {}
- git_log: Show recent commits. Args: {}
- search_code: Search for pattern in code files. Args: {"pattern": "...", "path": "..."} (path optional)
- search_file: Search for files by name. Args: {"pattern": "...", "path": "..."} (path optional)

When you need to use a tool, respond with ONLY a JSON block like this:
{"tool": "tool_name", "args": {"arg1": "value1"}}

Rules:
1. Always check the current state before making changes (use list_dir, read_file first)
2. Explain what you're about to do before executing tools
3. For file writes, include the COMPLETE file content
4. After using a tool, analyze the result and decide next steps
5. When task is complete, give a summary without tool calls
6. For searching code content, use search_code tool (not bash grep)
7. For searching files by name, use search_file tool (not bash find)

Respond in the same language as the user (Japanese or English).
Be concise but helpful.`,
	}
}

// RunInteractive は対話モードを開始
func RunInteractive(model string) {
	agent := NewAgent(model)

	// ヘッダー表示
	printHeader(model)

	// XELYON.md読み込み
	if config := loadProjectConfig(); config != "" {
		agent.SystemPrompt += "\n\n## Project Context:\n" + config
		green.Println("📋 XELYON.md loaded")
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

// RunOnce はワンショットモードを実行
func RunOnce(query string, model string) {
	agent := NewAgent(model)

	if config := loadProjectConfig(); config != "" {
		agent.SystemPrompt += "\n\n## Project Context:\n" + config
		green.Println("📋 XELYON.md loaded")
	}

	fmt.Println()
	agent.chat(query)
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
		a.session.AddMessage("user", input, a.Model)
	}

	// AIに送信（ツール実行ループ）
	maxIterations := 10 // 無限ループ防止
	for i := 0; i < maxIterations; i++ {
		response, err := api.ChatWithTools(a.SystemPrompt, a.History, a.Model)
		if err != nil {
			red.Printf("エラー: %v\n", err)
			return
		}

		// ツール呼び出しをチェック
		toolCall := tools.ParseToolCall(response)
		if toolCall != nil {
			// 結果を履歴に追加
			a.History = append(a.History, api.Message{
				Role:    "assistant",
				Content: response,
			})

			// ツール実行
			result := tools.Execute(toolCall)

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
			a.session.AddMessage("assistant", response, a.Model)
			if a.storage != nil {
				if err := a.storage.Save(a.session); err != nil {
					// サイレント失敗（ユーザー体験を妨げない）
				}
			}
		}

		break
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
		fmt.Printf("🤖 Current model: %s\n", modelDisplayName(agent.Model))
		return true
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

// printHeader はヘッダーを表示
func printHeader(model string) {
	cyan.Println("╔═══════════════════════════════════════════╗")
	cyan.Println("║  🚀 XELYON CLI v0.3.0                     ║")
	cyan.Println("║  AI-powered coding assistant              ║")
	cyan.Println("╚═══════════════════════════════════════════╝")
	fmt.Printf("Model: %s\n", modelDisplayName(model))
	yellow.Println("Type /help for commands, /exit to quit")
}

// printHelp はヘルプを表示
func printHelp() {
	fmt.Println(`
Commands:
  /exit, /quit, /q  - Exit the CLI
  /clear            - Clear conversation history
  /history          - Show conversation history
  /save             - Save current session
  /load [id]        - Load session (or last if no ID)
  /sessions         - List recent sessions
  /model            - Show current model
  /help             - Show this help

Available tools (AI will use automatically):
  bash        - Execute shell commands
  read_file   - Read file contents
  write_file  - Write/create files
  str_replace - Replace text in file
  list_dir    - List directory contents
  git_*       - Git operations (status, diff, add, commit, push, log)
  search_code - Search in code files
  search_file - Search for files by name

Tips:
  - Just describe what you want in natural language
  - AI will ask confirmation for dangerous operations
  - Use Ctrl+C to cancel current operation
`)
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
func RunInteractiveWithResume(model string) {
	storage, err := history.NewStorage()
	if err != nil {
		red.Printf("Failed to initialize storage: %v\n", err)
		RunInteractive(model)
		return
	}

	sessionID, err := storage.GetLastSession()
	if err != nil {
		yellow.Println("No previous session found, starting new session")
		RunInteractive(model)
		return
	}

	session, err := storage.Load(sessionID)
	if err != nil {
		red.Printf("Failed to load session: %v\n", err)
		RunInteractive(model)
		return
	}

	// ロード済みセッションでAgent作成
	agent := NewAgent(model)
	agent.session = session
	agent.History = session.ToAPIMessages()

	printHeader(model)
	green.Printf("📂 Resumed session %s (%d messages)\n", sessionID, len(session.Messages))

	if config := loadProjectConfig(); config != "" {
		agent.SystemPrompt += "\n\n## Project Context:\n" + config
		green.Println("📋 XELYON.md loaded")
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