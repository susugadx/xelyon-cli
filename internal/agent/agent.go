package agent

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/history"
	"github.com/susugadx/xelyon-cli/internal/mcp"
	"github.com/susugadx/xelyon-cli/internal/memory"
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
	changeStorage   *history.ChangeStorage // 永続的変更履歴
	mcpManager      *mcp.Manager
	AutoApprove     bool               // --auto-approve フラグ
	DryRunMode      bool               // --dry-run フラグ
	PlanMode        bool               // --plan フラグ（Plan Mode有効化）
	Stats           *SessionStats      // セッション統計情報
	lastOutputs     []string           // 最後のAI出力履歴（最大10件）
	cancelFunc      context.CancelFunc // 現在のAPI呼び出しをキャンセルするための関数
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

### Plan Mode (if enabled)
When Plan Mode is enabled, follow this workflow:
1. Analyze the user's request and break it down into steps
2. Output a JSON plan in this exact format:
   {"steps": [
     {"id": 1, "description": "Step description", "tools": ["tool1", "tool2"], "depends_on": [], "parallel": true},
     {"id": 2, "description": "Next step", "tools": ["tool3"], "depends_on": [1], "parallel": false}
   ]}
3. Wait for user approval (y/n/c)
4. After approval, execute steps autonomously:
   - Execute steps with depends_on=[] first
   - Parallel steps can run simultaneously
   - Sequential steps run one by one
   - Only ask for confirmation on SafetyLow operations (delete_file, bash, git_push, etc.)

### Phase 2: Execution
5. Use the right tool for each task:
   - search_code: Search code content (NOT bash grep)
   - search_file: Search file names (NOT bash find)
   - str_replace: Edit existing files (NOT bash sed)
   - write_file: Create NEW files only (NOT for editing)

CRITICAL FILE EDITING RULES:
- NEVER use write_file to modify existing files - ALWAYS use str_replace
- write_file is ONLY for creating NEW files that don't exist yet
- For ANY edit to an existing file, use str_replace with minimal old_str/new_str
- Keep each str_replace small (under 20 lines typically)
- If change requires rewriting >50% of file, STOP and ask user first

WRONG approach:
  "I'll rewrite the entire file with the fix"
  → Uses write_file to overwrite 500 lines for a 2-line fix

CORRECT approach:
  "I'll use str_replace to change just the affected lines"
  → Uses str_replace with minimal old_str containing only the lines to change

6. str_replace safety rules:
   - If old_str matches multiple times, include surrounding context to make it unique
   - For large edits, split into multiple str_replace calls (~10 lines each)
   - When editing the same file consecutively, use read_file to verify current state
7. Respond with ONLY a JSON block for tool calls: {"tool": "...", "args": {...}}

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
		systemPrompt += buildMCPToolsPrompt(mcpManager)
	}

	// メモリをSystemPromptに追加
	memoryStore, err := memory.NewMemoryStore()
	if err == nil {
		memoriesText := memoryStore.GetMemoriesAsText()
		if memoriesText != "" {
			systemPrompt += memoriesText
		}
	}

	// 変更履歴ストレージ初期化
	changeStorage, err := history.NewChangeStorage()
	if err != nil {
		yellow.Printf("Warning: Failed to initialize change storage: %v\n", err)
		changeStorage = nil
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
		changeStorage:   changeStorage,
		mcpManager:      mcpManager,
		SystemPrompt:    systemPrompt,
		Stats:           NewSessionStats(provider.Name()),
		lastOutputs:     []string{},
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

// SwitchProvider はプロバイダーを切り替える
func (a *Agent) SwitchProvider(providerName string) error {
	// API キー存在チェック
	if !IsAPIKeyAvailable(providerName) {
		return fmt.Errorf("%s のAPIキーが設定されていません", providerName)
	}

	// プロバイダーインスタンス作成
	provider, err := api.NewProvider(providerName)
	if err != nil {
		return fmt.Errorf("プロバイダーの初期化に失敗しました: %w", err)
	}

	// 設定ファイルから新しいプロバイダーのデフォルトモデルを取得
	cfg, err := config.LoadConfig()
	if err != nil {
		yellow.Printf("Warning: Failed to load config: %v\n", err)
		cfg = config.DefaultConfig()
	}
	newModel := cfg.GetModelForProvider(providerName)
	if newModel == "" {
		// フォールバック: プロバイダー別のハードコードされたデフォルト
		switch providerName {
		case "deepseek":
			newModel = "deepseek-coder"
		case "openai":
			newModel = "gpt-5.2"
		case "gemini":
			newModel = "gemini-2.5-flash"
		case "claude":
			newModel = "claude-sonnet-4-5-20250514"
		case "ollama":
			newModel = "qwen2.5-coder:7b"
		case "groq":
			newModel = "meta-llama/llama-4-scout-17b-16e-instruct"
		default:
			newModel = "default-model"
		}
	}

	// プロバイダー切り替え
	oldProvider := a.ProviderName
	oldModel := a.CurrentModel
	a.CurrentProvider = provider
	a.ProviderName = providerName
	a.CurrentModel = newModel

	// 統計情報のプロバイダー名も更新
	if a.Stats != nil {
		a.Stats.Provider = providerName
	}

	green.Printf("✅ Provider: %s → %s\n", oldProvider, providerName)
	green.Printf("✅ Model: %s → %s\n", oldModel, newModel)
	return nil
}

// IsAPIKeyAvailable は指定されたプロバイダーのAPIキーが利用可能かチェック
func IsAPIKeyAvailable(provider string) bool {
	switch provider {
	case "deepseek":
		return os.Getenv("DEEPSEEK_API_KEY") != ""
	case "openai":
		return os.Getenv("OPENAI_API_KEY") != ""
	case "claude":
		return os.Getenv("ANTHROPIC_API_KEY") != ""
	case "gemini":
		return os.Getenv("GEMINI_API_KEY") != ""
	case "groq":
		return os.Getenv("GROQ_API_KEY") != ""
	case "ollama":
		return true // Ollama はローカルなのでキー不要
	default:
		return false
	}
}

// parseImageInput は入力から画像パスを抽出
// 形式: "image:/path/to/file.png こんにちは" または "こんにちは image:/path/to/file.png"
func parseImageInput(input string) (text string, image *api.ImageData) {
	// image:プレフィックスを探す
	imagePrefix := "image:"

	// 正規表現的な簡易パース
	parts := strings.Fields(input)
	var textParts []string
	var imagePath string

	for _, part := range parts {
		if strings.HasPrefix(part, imagePrefix) {
			imagePath = strings.TrimPrefix(part, imagePrefix)
		} else {
			textParts = append(textParts, part)
		}
	}

	// 画像パスがない場合
	if imagePath == "" {
		return input, nil
	}

	// テキスト部分を結合
	text = strings.Join(textParts, " ")
	if text == "" {
		text = "Please analyze this image." // デフォルトメッセージ
	}

	// 画像読み込み
	img, err := api.LoadImage(imagePath)
	if err != nil {
		red.Printf("Failed to load image: %v\n", err)
		return input, nil
	}

	green.Printf("🖼️  Image loaded: %s (%s)\n", img.Path, api.FormatImageSize(img.Size))
	return text, img
}

// RunOnce は単一のクエリを実行（レガシーAPI）
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

	// Repo Map 生成（キャッシュあり）
	cwd, err := os.Getwd()
	if err != nil {
		yellow.Printf("Warning: Could not get current directory: %v\n", err)
		cwd = "." // フォールバック
	}
	repoMapStr, symbols, files, fromCache := loadRepoMapForProject(cwd, 2000)
	if repoMapStr != "" {
		agent.SystemPrompt += "\n\n" + repoMapStr
		if fromCache {
			green.Println("🗺️  Repo map loaded (cache)")
		} else {
			green.Printf("🗺️  Repo map loaded (%d symbols from %d files)\n", symbols, files)
		}
	}

	fmt.Println()
	agent.chat(query)
}

// RunHeadless はHeadlessモードでクエリを実行
func RunHeadless(query string, model string, provider api.Provider) *HeadlessResult {
	startTime := time.Now()

	// Agent初期化
	agent := NewAgent(model, provider)
	agent.AutoApprove = true // Headlessモードは自動承認（SafetyLow以外）
	tools.SetAutoApprove(true)

	// プロジェクト設定読み込み（UI出力なし）
	if config := loadProjectConfig(); config != "" {
		agent.SystemPrompt += "\n\n## Project Context:\n" + config
	}

	// Repo Map 生成（キャッシュあり / UI出力なし）
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "."
	}
	repoMapStr, _, _, _ := loadRepoMapForProject(cwd, 2000)
	if repoMapStr != "" {
		agent.SystemPrompt += "\n\n" + repoMapStr
	}

	// ツール呼び出し結果を記録
	var toolCalls []ToolCallResult

	// API呼び出し
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	response, err := provider.ChatWithTools(ctx, agent.SystemPrompt, agent.History, model)
	if err != nil {
		duration := time.Since(startTime).Milliseconds()
		return NewErrorResult(provider.Name(), model, "api_error", err.Error(), duration)
	}

	// ツール呼び出し解析（複数対応）
	// TODO: 実際のツール実行を含める場合は agent.Run() のロジックを統合
	parsedCalls := tools.ParseToolCalls(response)
	for _, tc := range parsedCalls {
		// ツール実行（エラーは記録するが続行）
		output, _ := tools.Execute(tc)
		toolCalls = append(toolCalls, ToolCallResult{
			Tool:    tc.Tool,
			Args:    tc.Args,
			Output:  output,
			Success: true,
		})
	}

	duration := time.Since(startTime).Milliseconds()
	return NewSuccessResult(provider.Name(), model, response, toolCalls, duration)
}

// printHeader はセッション開始時のヘッダーを表示
func printHeader(model string, provider api.Provider) {
	cyan.Println("╔═══════════════════════════════════════════╗")
	cyan.Printf("║  🚀 XELYON CLI v%-25s║\n", version.GetVersion())
	cyan.Println("║  AI-powered coding assistant              ║")
	cyan.Println("╚═══════════════════════════════════════════╝")
	green.Printf("🌐 Provider: %s\n", provider.Name())
	fmt.Printf("Model: %s\n", modelDisplayName(model))
	yellow.Println("Type /help for commands, /exit to quit")
}

// modelDisplayName はモデル名を表示用にフォーマット
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

// loadProjectConfig はXELYON.mdをロード
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
