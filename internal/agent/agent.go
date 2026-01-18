package agent

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/audit"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/history"
	"github.com/susugadx/xelyon-cli/internal/mcp"
	"github.com/susugadx/xelyon-cli/internal/memory"
	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/ui"
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
	AutoApprove          bool                // --auto-approve フラグ
	DryRunMode           bool                // --dry-run フラグ
	Stats                *SessionStats       // セッション統計情報
	lastOutputs          []string            // 最後のAI出力履歴（最大10件）
	cancelFunc           context.CancelFunc  // 現在のAPI呼び出しをキャンセルするための関数
	strReplaceErrorCount int                 // str_replace連続エラーカウント（old_str not found）
	mlReader             *ui.MultilineReader // 共有入力リーダー（ペーストモードでも使用）
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
- append_file, prepend_file: {"path": "...", "content": "..."}
- insert_after, insert_before: {"path": "...", "pattern": "...", "content": "..."}
- copy_file, move_file: {"src": "...", "dest": "..."}
- delete_file, delete_lines: {"path": "..."}
- list_dir, create_dir: {"path": "..."}
- restore_backup, list_backups: {"path": "..."}

### Git Operations
- git_status, git_diff, git_log, git_add, git_commit, git_push
- git_branch: {"action": "list|create|switch", "branch_name": "..."}
- git_checkout: {"target": "..."}
- git_stash: {"action": "save|list|pop|apply|drop"}

### Search & Discovery
- search_code: {"pattern": "...", "path": "..."} - Search code content
- search_file: {"pattern": "...", "path": "..."} - Search file names
- grep_replace: {"pattern": "regex", "replacement": "...", "dry_run": "true|false"}
- ast_grep: {"pattern": "...", "lang": "...", "path": "..."} - Structural code search
- web_search: {"query": "..."}

### Development Tools
- run_test, format, lint: {"path": "..."}
- diff_files: {"file1": "...", "file2": "..."}
- http_request: {"method": "GET|POST|PUT|DELETE", "url": "...", "headers": "{}", "body": "..."}
- bash: {"command": "..."} - Shell commands

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
- search_code for content search (NOT bash grep)
- search_file for file names (NOT bash find)
- str_replace for edits (NOT bash sed)

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
	fmt.Printf("   Model: %s\n", modelDisplayName(model))
}

// printModeInfo はモード情報を表示
func printModeInfo(autoApprove, dryRun bool) {
	var modes []string
	if autoApprove {
		modes = append(modes, "Auto-approve")
	}
	if dryRun {
		modes = append(modes, "Dry-run")
	}

	if len(modes) > 0 {
		yellow.Printf("   Mode: %s\n", strings.Join(modes, ", "))
	} else {
		fmt.Println("   Mode: Normal")
	}
	cyan.Println("───────────────────────────────────────────")
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

// RunOnceWithImage は画像付きの単一クエリを実行（CLIフラグ -i/--image 用）
func RunOnceWithImage(query string, model string, provider api.Provider, imagePath string, autoApprove bool) {
	// 監査ログ初期化（環境変数で制御: XELYON_AUDIT_LOG=1 で有効化）
	auditEnabled := os.Getenv("XELYON_AUDIT_LOG") == "1"
	if err := audit.Init(auditEnabled); err != nil {
		yellow.Printf("Warning: Failed to initialize audit log: %v\n", err)
	}

	agent := NewAgent(model, provider)
	agent.AutoApprove = autoApprove
	tools.SetAutoApprove(autoApprove)
	defer agent.Cleanup()

	// ヘッダー表示
	printHeader(model, provider)
	printModeInfo(autoApprove, false)

	// プロバイダーが画像対応かチェック
	if !api.SupportsImages(provider.Name()) {
		red.Printf("❌ Provider '%s' does not support image input\n", provider.Name())
		fmt.Println("Supported providers for image input: gemini, claude, openai")
		return
	}

	// 画像読み込み
	image, err := api.LoadImage(imagePath)
	if err != nil {
		red.Printf("❌ Failed to load image: %v\n", err)
		return
	}
	green.Printf("🖼️  Image loaded: %s (%s)\n", image.Path, api.FormatImageSize(image.Size))

	// XELYON.md読み込み
	if config := loadProjectConfig(); config != "" {
		agent.SystemPrompt += "\n\n## Project Context:\n" + config
		green.Println("📋 XELYON.md loaded")
	}

	// Repo Map 生成（キャッシュあり）
	cwd, err := os.Getwd()
	if err != nil {
		yellow.Printf("Warning: Could not get current directory: %v\n", err)
		cwd = "."
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

	// デフォルトメッセージ
	if query == "" {
		query = "Please analyze this image."
	}

	// 画像付きで会話
	agent.chatWithImage(query, image)

	// 対話ループに入る
	mlReader := ui.NewMultilineReader(os.Stdin)
	mlReader.EnableBracketedPaste()
	defer mlReader.DisableBracketedPaste()
	agent.mlReader = mlReader // ペーストモードで共有するため

	for {
		mlReader.FlushInput()
		agent.PrintStatusFooter()

		input, err := mlReader.ReadInput("\n> ")
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

		// 通常の会話
		agent.chat(input)
	}
}
