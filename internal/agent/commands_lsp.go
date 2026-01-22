package agent

import (
	"fmt"
	"os"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/lsp"
)

// handleLSPCommand は LSP サーバーの状態を表示・管理
func handleLSPCommand(agent *Agent, args []string) bool {
	lspClient := agent.GetLSPClient()
	if lspClient == nil {
		yellow.Println("LSP is not enabled.")
		yellow.Println("To enable, set 'lsp.enabled: true' in ~/.xelyon/config.yaml")
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
		yellow.Printf("Unknown subcommand: %s\n", subCmd)
		yellow.Println("Usage: /lsp [status|detect|install <language|all>]")
		return true
	}
}

// handleLSPStatus は LSP サーバーの状態を表示
func handleLSPStatus(agent *Agent, lspClient *lsp.Client) bool {
	printCommandHeader("LSP Server Status")

	status := lspClient.Status()
	if len(status) == 0 {
		yellow.Println("  No LSP servers configured")
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
		fmt.Printf("  %s %-12s : %s\n", icon, lang, state)
	}

	// プロジェクト内の言語を検出
	cwd, _ := os.Getwd()
	detectedLangs, _ := lsp.DetectProjectLanguages(cwd)
	if len(detectedLangs) > 0 {
		fmt.Println()
		var langNames []string
		for _, info := range detectedLangs {
			langNames = append(langNames, info.ServerKey)
		}
		cyan.Printf("🔍 Detected languages in project: %s\n", strings.Join(langNames, ", "))
	}

	// 未インストールサーバーのインストール提案
	if len(notInstalled) > 0 {
		fmt.Println()
		yellow.Println("💡 Missing LSP servers:")
		for _, lang := range notInstalled {
			if info, ok := lsp.GetInstallInfo(lang); ok {
				fmt.Printf("   %s: %s\n", lang, info.Commands[0])
			}
		}
		fmt.Println()
		yellow.Printf("   Run '/lsp install <language>' to install\n")
		yellow.Printf("   Run '/lsp install all' to install all missing servers\n")
	} else {
		fmt.Println()
		green.Println("Tip: LSP servers start lazily when first used.")
	}

	return true
}

// handleLSPDetect はプロジェクト内の言語を検出して表示
func handleLSPDetect(agent *Agent, lspClient *lsp.Client) bool {
	cyan.Println("🔍 Detecting languages in project...")
	fmt.Println()

	cwd, err := os.Getwd()
	if err != nil {
		red.Printf("Error getting current directory: %v\n", err)
		return true
	}

	languages, err := lsp.DetectProjectLanguages(cwd)
	if err != nil {
		red.Printf("Error detecting languages: %v\n", err)
		return true
	}

	if len(languages) == 0 {
		yellow.Println("No supported languages detected in this project.")
		return true
	}

	cyan.Println("Found:")
	for _, info := range languages {
		exts := strings.Join(info.Extensions, ", ")
		fmt.Printf("  - %s (%s): %d files\n", info.ServerKey, exts, info.FileCount)
	}

	// LSPサーバーの状態を確認
	status := lspClient.Status()
	fmt.Println()
	cyan.Println("LSP Status:")

	var notInstalled []string
	for _, info := range languages {
		serverKey := info.ServerKey
		state, ok := status[serverKey]

		if !ok {
			fmt.Printf("  ❓ %s: not configured\n", serverKey)
			continue
		}

		if strings.HasPrefix(state, "not installed") {
			fmt.Printf("  ❌ %s: not installed\n", serverKey)
			notInstalled = append(notInstalled, serverKey)
		} else if state == "disabled" {
			fmt.Printf("  🔴 %s: disabled\n", serverKey)
		} else {
			installInfo, _ := lsp.GetInstallInfo(serverKey)
			fmt.Printf("  ✅ %s: installed (%s)\n", serverKey, installInfo.PackageName)
		}
	}

	if len(notInstalled) > 0 {
		fmt.Println()
		yellow.Printf("Run '/lsp install %s' to install missing servers.\n", notInstalled[0])
		if len(notInstalled) > 1 {
			yellow.Println("Run '/lsp install all' to install all missing servers.")
		}
	}

	return true
}

// handleLSPInstall は LSP サーバーをインストール
func handleLSPInstall(agent *Agent, lspClient *lsp.Client, args []string) bool {
	if len(args) == 0 {
		yellow.Println("Usage: /lsp install <language|all>")
		yellow.Println("Available languages: go, typescript, python, rust")
		return true
	}

	target := args[0]

	if target == "all" {
		return handleLSPInstallAll(agent, lspClient)
	}

	// 単一言語のインストール
	info, ok := lsp.GetInstallInfo(target)
	if !ok {
		red.Printf("Unknown language: %s\n", target)
		yellow.Println("Available languages: go, typescript, python, rust")
		return true
	}

	// インストール済みチェック
	status := lspClient.Status()
	if state, exists := status[target]; exists && !strings.HasPrefix(state, "not installed") {
		green.Printf("✅ %s is already installed (%s)\n", target, info.PackageName)
		return true
	}

	// インストール実行
	cyan.Printf("📦 Installing %s LSP server (%s)...\n", target, info.PackageName)
	fmt.Println()
	cyan.Printf("Running: %s\n", info.Commands[0])
	fmt.Println()

	if err := lsp.RunInstall(target); err != nil {
		red.Printf("❌ Installation failed: %v\n", err)
		fmt.Println()
		yellow.Println("Try installing manually:")
		for _, cmd := range info.Commands {
			fmt.Printf("   %s\n", cmd)
		}
		return true
	}

	fmt.Println()
	green.Printf("✅ %s installed successfully!\n", info.PackageName)
	fmt.Println()
	yellow.Println("Tip: LSP server will start automatically when you use LSP tools.")
	return true
}

// handleLSPInstallAll は未インストールの全LSPサーバーをインストール
func handleLSPInstallAll(agent *Agent, lspClient *lsp.Client) bool {
	status := lspClient.Status()

	// 未インストールのサーバーを収集
	var toInstall []string
	for lang, state := range status {
		if strings.HasPrefix(state, "not installed") {
			toInstall = append(toInstall, lang)
		}
	}

	if len(toInstall) == 0 {
		green.Println("✅ All configured LSP servers are already installed!")
		return true
	}

	cyan.Println("📦 Installing missing LSP servers...")
	fmt.Println()

	successCount := 0
	for i, lang := range toInstall {
		info, ok := lsp.GetInstallInfo(lang)
		if !ok {
			continue
		}

		fmt.Printf("[%d/%d] Installing %s (%s)...\n", i+1, len(toInstall), lang, info.PackageName)
		fmt.Printf("      %s\n", info.Commands[0])

		if err := lsp.RunInstall(lang); err != nil {
			red.Printf("      ❌ Failed: %v\n", err)
		} else {
			green.Println("      ✅ Success")
			successCount++
		}
		fmt.Println()
	}

	if successCount == len(toInstall) {
		green.Println("All servers installed!")
	} else {
		yellow.Printf("%d of %d servers installed successfully.\n", successCount, len(toInstall))
	}

	return true
}
