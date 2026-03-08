package agent

import (
	"fmt"
	"os"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/lsp"
)

// handleLSPCommand は LSP サーバーの状態を表示・管理
func handleLSPCommand(agent *Agent, args []string) bool {
	out := agent.output()
	lspClient := agent.GetLSPClient()
	if lspClient == nil {
		yellow.Fprintln(out, "LSP is not enabled.")
		yellow.Fprintln(out, "To enable, set 'lsp.enabled: true' in ~/.xelyon/config.yaml")
		return true
	}

	// サブコマンド処理（デフォルトはstatus）
	subCmd := "status"
	if len(args) > 0 {
		subCmd = args[0]
	}

	switch subCmd {
	case "status":
		return handleLSPStatus(agent, lspClient)

	case "detect":
		return handleLSPDetect(agent, lspClient)

	case "install":
		return handleLSPInstall(agent, lspClient, args[1:])

	default:
		yellow.Fprintf(out, "Unknown subcommand: %s\n", subCmd)
		yellow.Fprintln(out, "Usage: /lsp [status|detect|install <language|all>]")
		return true
	}
}

// handleLSPStatus は LSP サーバーの状態を表示
func handleLSPStatus(agent *Agent, lspClient *lsp.Client) bool {
	out := agent.output()
	printCommandHeaderToWriter(out, "LSP Server Status")

	status := lspClient.Status()
	if len(status) == 0 {
		yellow.Fprintln(out, "  No LSP servers configured")
		return true
	}

	// 未インストールサーバーを収集
	var notInstalled []string

	for lang, state := range status {
		var icon string
		switch {
		case state == "running":
			icon = "🟢"
		case state == "disabled":
			icon = "🔴"
		case strings.HasPrefix(state, "not installed"):
			icon = "⚠️ "
			notInstalled = append(notInstalled, lang)
		default:
			icon = "⚪"
		}
		_, _ = fmt.Fprintf(out, "  %s %-12s : %s\n", icon, lang, state)
	}

	// プロジェクト内の言語を検出
	cwd, _ := os.Getwd()
	detectedLangs, _ := lsp.DetectProjectLanguages(cwd)
	if len(detectedLangs) > 0 {
		_, _ = fmt.Fprintln(out)
		var langNames []string
		for _, info := range detectedLangs {
			langNames = append(langNames, info.ServerKey)
		}
		cyan.Fprintf(out, "🔍 Detected languages in project: %s\n", strings.Join(langNames, ", "))
	}

	// 未インストールサーバーのインストール提案
	if len(notInstalled) > 0 {
		_, _ = fmt.Fprintln(out)
		yellow.Fprintln(out, "💡 Missing LSP servers:")
		for _, lang := range notInstalled {
			if info, ok := lsp.GetInstallInfo(lang); ok {
				_, _ = fmt.Fprintf(out, "   %s: %s\n", lang, info.Commands[0])
			}
		}
		_, _ = fmt.Fprintln(out)
		yellow.Fprintln(out, "   Run '/lsp install <language>' to install")
		yellow.Fprintln(out, "   Run '/lsp install all' to install all missing servers")
	} else {
		_, _ = fmt.Fprintln(out)
		green.Fprintln(out, "Tip: LSP servers start lazily when first used.")
	}

	return true
}

// handleLSPDetect はプロジェクト内の言語を検出して表示
func handleLSPDetect(agent *Agent, lspClient *lsp.Client) bool {
	out := agent.output()
	cyan.Fprintln(out, "🔍 Detecting languages in project...")
	_, _ = fmt.Fprintln(out)

	cwd, err := os.Getwd()
	if err != nil {
		red.Fprintf(out, "Error getting current directory: %v\n", err)
		return true
	}

	languages, err := lsp.DetectProjectLanguages(cwd)
	if err != nil {
		red.Fprintf(out, "Error detecting languages: %v\n", err)
		return true
	}

	if len(languages) == 0 {
		yellow.Fprintln(out, "No supported languages detected in this project.")
		return true
	}

	cyan.Fprintln(out, "Found:")
	for _, info := range languages {
		exts := strings.Join(info.Extensions, ", ")
		_, _ = fmt.Fprintf(out, "  - %s (%s): %d files\n", info.ServerKey, exts, info.FileCount)
	}

	// LSPサーバーの状態を確認
	status := lspClient.Status()
	_, _ = fmt.Fprintln(out)
	cyan.Fprintln(out, "LSP Status:")

	var notInstalled []string
	for _, info := range languages {
		serverKey := info.ServerKey
		state, ok := status[serverKey]

		if !ok {
			_, _ = fmt.Fprintf(out, "  ❓ %s: not configured\n", serverKey)
			continue
		}

		if strings.HasPrefix(state, "not installed") {
			_, _ = fmt.Fprintf(out, "  ❌ %s: not installed\n", serverKey)
			notInstalled = append(notInstalled, serverKey)
		} else if state == "disabled" {
			_, _ = fmt.Fprintf(out, "  🔴 %s: disabled\n", serverKey)
		} else {
			installInfo, _ := lsp.GetInstallInfo(serverKey)
			_, _ = fmt.Fprintf(out, "  ✅ %s: installed (%s)\n", serverKey, installInfo.PackageName)
		}
	}

	if len(notInstalled) > 0 {
		_, _ = fmt.Fprintln(out)
		yellow.Fprintf(out, "Run '/lsp install %s' to install missing servers.\n", notInstalled[0])
		if len(notInstalled) > 1 {
			yellow.Fprintln(out, "Run '/lsp install all' to install all missing servers.")
		}
	}

	return true
}

// handleLSPInstall は LSP サーバーをインストール
func handleLSPInstall(agent *Agent, lspClient *lsp.Client, args []string) bool {
	out := agent.output()
	if len(args) == 0 {
		yellow.Fprintln(out, "Usage: /lsp install <language|all>")
		yellow.Fprintln(out, "Available languages: go, typescript, python, rust")
		return true
	}

	target := args[0]

	if target == "all" {
		return handleLSPInstallAll(agent, lspClient)
	}

	// 単一言語のインストール
	info, ok := lsp.GetInstallInfo(target)
	if !ok {
		red.Fprintf(out, "Unknown language: %s\n", target)
		yellow.Fprintln(out, "Available languages: go, typescript, python, rust")
		return true
	}

	// インストール済みチェック
	status := lspClient.Status()
	if state, exists := status[target]; exists && !strings.HasPrefix(state, "not installed") {
		green.Fprintf(out, "✅ %s is already installed (%s)\n", target, info.PackageName)
		return true
	}

	// インストール実行
	cyan.Fprintf(out, "📦 Installing %s LSP server (%s)...\n", target, info.PackageName)
	_, _ = fmt.Fprintln(out)
	cyan.Fprintf(out, "Running: %s\n", info.Commands[0])
	_, _ = fmt.Fprintln(out)

	if err := lsp.RunInstallWithIO(target, agent.ui().Input(), agent.ui().Output(), agent.ui().ErrorOutput()); err != nil {
		red.Fprintf(out, "❌ Installation failed: %v\n", err)
		_, _ = fmt.Fprintln(out)
		yellow.Fprintln(out, "Try installing manually:")
		for _, cmd := range info.Commands {
			_, _ = fmt.Fprintf(out, "   %s\n", cmd)
		}
		return true
	}

	_, _ = fmt.Fprintln(out)
	green.Fprintf(out, "✅ %s installed successfully!\n", info.PackageName)
	_, _ = fmt.Fprintln(out)
	yellow.Fprintln(out, "Tip: LSP server will start automatically when you use LSP tools.")
	return true
}

// handleLSPInstallAll は未インストールの全LSPサーバーをインストール
func handleLSPInstallAll(agent *Agent, lspClient *lsp.Client) bool {
	out := agent.output()
	status := lspClient.Status()

	// 未インストールのサーバーを収集
	var toInstall []string
	for lang, state := range status {
		if strings.HasPrefix(state, "not installed") {
			toInstall = append(toInstall, lang)
		}
	}

	if len(toInstall) == 0 {
		green.Fprintln(out, "✅ All configured LSP servers are already installed!")
		return true
	}

	cyan.Fprintln(out, "📦 Installing missing LSP servers...")
	_, _ = fmt.Fprintln(out)

	successCount := 0
	for i, lang := range toInstall {
		info, ok := lsp.GetInstallInfo(lang)
		if !ok {
			continue
		}

		_, _ = fmt.Fprintf(out, "[%d/%d] Installing %s (%s)...\n", i+1, len(toInstall), lang, info.PackageName)
		_, _ = fmt.Fprintf(out, "      %s\n", info.Commands[0])

		if err := lsp.RunInstallWithIO(lang, agent.ui().Input(), agent.ui().Output(), agent.ui().ErrorOutput()); err != nil {
			red.Fprintf(out, "      ❌ Failed: %v\n", err)
		} else {
			green.Fprintln(out, "      ✅ Success")
			successCount++
		}
		_, _ = fmt.Fprintln(out)
	}

	if successCount == len(toInstall) {
		green.Fprintln(out, "All servers installed!")
	} else {
		yellow.Fprintf(out, "%d of %d servers installed successfully.\n", successCount, len(toInstall))
	}

	return true
}
