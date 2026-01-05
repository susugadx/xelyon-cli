package agent

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/susugadx/xelyon-cli/internal/api"
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
}

// NewAgent は新しいAgentを作成
func NewAgent(model string) *Agent {
	return &Agent{
		Model:   model,
		History: []api.Message{},
		SystemPrompt: `You are XELYON, an expert AI coding assistant.

You have access to the following tools:
- bash: Execute shell commands. Args: {"command": "..."}
- read_file: Read file contents. Args: {"path": "..."}
- write_file: Write content to a file. Args: {"path": "...", "content": "..."}
- list_dir: List directory contents. Args: {"path": "..."} (optional, defaults to current dir)
- git_status: Show git status. Args: {}
- git_diff: Show git diff. Args: {"path": "..."} (optional)
- git_add: Stage files. Args: {"path": "..."} (default: ".")
- git_commit: Commit changes. Args: {"message": "..."}
- git_push: Push to remote. Args: {}
- git_log: Show recent commits. Args: {}

When you need to use a tool, respond with ONLY a JSON block like this:
{"tool": "tool_name", "args": {"arg1": "value1"}}

Rules:
1. Always check the current state before making changes (use list_dir, read_file first)
2. Explain what you're about to do before executing tools
3. For file writes, include the COMPLETE file content
4. After using a tool, analyze the result and decide next steps
5. When task is complete, give a summary without tool calls

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
		break
	}
}

// handleSpecialCommand は特殊コマンドを処理
func handleSpecialCommand(input string, agent *Agent) bool {
	switch strings.ToLower(input) {
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

// printHeader はヘッダーを表示
func printHeader(model string) {
	cyan.Println("╔═══════════════════════════════════════════╗")
	cyan.Println("║  🚀 XELYON CLI v0.2.0                     ║")
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
  /model            - Show current model
  /help             - Show this help

Available tools (AI will use automatically):
  bash       - Execute shell commands
  read_file  - Read file contents
  write_file - Write/create files
  list_dir   - List directory contents

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