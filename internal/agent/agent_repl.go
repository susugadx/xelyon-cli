package agent

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/audit"
	"github.com/susugadx/xelyon-cli/internal/history"
	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

// RunInteractive はインタラクティブモードでエージェントを実行
func RunInteractive(model string, provider api.Provider, autoApprove, planMode bool) {
	// 監査ログ初期化（環境変数で制御: XELYON_AUDIT_LOG=1 で有効化）
	auditEnabled := os.Getenv("XELYON_AUDIT_LOG") == "1"
	if err := audit.Init(auditEnabled); err != nil {
		yellow.Printf("Warning: Failed to initialize audit log: %v\n", err)
	}
	if auditEnabled {
		green.Println("📝 Audit logging enabled")
	}

	agent := NewAgent(model, provider)
	agent.AutoApprove = autoApprove
	agent.PlanMode = planMode         // Plan Modeフラグを設定
	tools.SetAutoApprove(autoApprove) // ツールに --auto-approve 設定を伝える
	defer agent.Cleanup()             // グレースフルシャットダウン

	// シグナルハンドリング（Ctrl+C 2回で終了、1回目はAI応答中断）
	setupSignalHandler(agent)

	// ヘッダー表示
	printHeader(model, provider)
	printModeInfo(planMode, autoApprove, false)

	// XELYON.md読み込み
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

	// REPLループ（複数行入力対応）
	mlReader := ui.NewMultilineReader(os.Stdin)
	mlReader.EnableBracketedPaste()
	defer mlReader.DisableBracketedPaste()

	for {
		// AI出力後に溜まった入力をクリア（出力中のEnter押下を無視）
		mlReader.FlushInput()

		// Status / 状態表示（常にプロンプト直前に表示）
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

		// 画像入力チェック: image:/path/to/file.png 形式を検出
		if strings.Contains(input, "image:") {
			textPart, image := parseImageInput(input)
			if image != nil {
				agent.chatWithImage(textPart, image)
				continue
			}
		}

		// AIに送信
		agent.chat(input)
	}
}

// RunInteractiveWithResume は前回のセッションを再開してインタラクティブモードを実行
func RunInteractiveWithResume(model string, provider api.Provider, autoApprove, planMode bool) {
	storage, err := history.NewStorage()
	if err != nil {
		red.Printf("Failed to initialize storage: %v\n", err)
		RunInteractive(model, provider, autoApprove, planMode)
		return
	}

	sessionID, err := storage.GetLastSession()
	if err != nil {
		yellow.Println("No previous session found, starting new session")
		RunInteractive(model, provider, autoApprove, planMode)
		return
	}

	session, err := storage.Load(sessionID)
	if err != nil {
		red.Printf("Failed to load session: %v\n", err)
		RunInteractive(model, provider, autoApprove, planMode)
		return
	}

	// ロード済みセッションでAgent作成
	agent := NewAgent(model, provider)
	agent.AutoApprove = autoApprove
	agent.PlanMode = planMode
	tools.SetAutoApprove(autoApprove) // ツールに --auto-approve 設定を伝える
	agent.session = session
	agent.History = session.ToAPIMessages()
	defer agent.Cleanup() // グレースフルシャットダウン

	// シグナルハンドリング（Ctrl+C 2回で終了、1回目はAI応答中断）
	setupSignalHandler(agent)

	printHeader(model, provider)
	printModeInfo(planMode, autoApprove, false)
	green.Printf("📂 Resumed session %s (%d messages)\n", sessionID, len(session.Messages))

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

	// REPLループ（複数行入力対応）
	mlReader := ui.NewMultilineReader(os.Stdin)
	mlReader.EnableBracketedPaste()
	defer mlReader.DisableBracketedPaste()

	for {
		// Status / 状態表示（常にプロンプト直前に表示）
		agent.PrintStatusFooter()

		input, err := mlReader.ReadInput("\n> ")
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

		// 画像入力チェック
		if strings.Contains(input, "image:") {
			textPart, image := parseImageInput(input)
			if image != nil {
				agent.chatWithImage(textPart, image)
				continue
			}
		}

		agent.chat(input)
	}
}

// setupSignalHandler はシグナルハンドラーを設定する
func setupSignalHandler(agent *Agent) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	var lastInterrupt time.Time
	var interruptMu sync.Mutex
	go func() {
		for range sigChan {
			interruptMu.Lock()
			now := time.Now()

			if now.Sub(lastInterrupt) < 3*time.Second {
				// 2回目（3秒以内）: アプリ終了
				interruptMu.Unlock()
				fmt.Println("\n\n👋 Gracefully shutting down...")
				agent.Cleanup()
				os.Exit(0)
			}

			lastInterrupt = now
			interruptMu.Unlock()

			// 1回目: 中断メッセージ
			fmt.Println("\n\n⚠️  Interrupted. Press Ctrl+C again within 3 seconds to exit.")

			// 現在のAPI呼び出しをキャンセル
			if agent.cancelFunc != nil {
				agent.cancelFunc()
			}
		}
	}()
}
