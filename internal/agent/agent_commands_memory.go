package agent

import (
	"fmt"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/memory"
)

// handleMemoryCommand は記憶の管理を処理
func handleMemoryCommand(args []string) bool {
	store, err := memory.NewMemoryStore()
	if err != nil {
		red.Printf("Failed to initialize memory store: %v\n", err)
		return true
	}

	// 引数なし、または "list" → 一覧表示
	if len(args) == 0 || (len(args) == 1 && args[0] == "list") {
		memories := store.List()
		if len(memories) == 0 {
			yellow.Println("No memories stored")
			yellow.Println("\nUsage: /memory <text>  または  /memory add <text>")
			return true
		}

		cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		cyan.Printf("🧠 Memories / 記憶 (%d 件)\n", len(memories))
		cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Println()

		for i, m := range memories {
			// プロジェクト別かグローバルかを表示
			scope := "Global"
			if m.Project != "" {
				scope = fmt.Sprintf("Project: %s", m.Project)
			}

			// タイムスタンプ
			timeStr := m.CreatedAt.Format("2006-01-02 15:04")

			fmt.Printf("  [%s] %s\n", m.ID, m.Content)
			fmt.Printf("      %s | %s\n", scope, timeStr)

			if i < len(memories)-1 {
				fmt.Println()
			}
		}

		fmt.Println()
		cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		yellow.Println("使い方:")
		yellow.Println("  /memory <text>         - 記憶を追加（グローバル）")
		yellow.Println("  /memory add <text>     - 記憶を追加（グローバル）")
		yellow.Println("  /memory project <text> - 記憶を追加（プロジェクト別）")
		yellow.Println("  /memory list           - 記憶一覧")
		yellow.Println("  /memory delete <id>    - 記憶削除")
		yellow.Println("  /memory clear          - 全記憶削除")
		cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

		return true
	}

	subcommand := args[0]

	// /memory add <text> → グローバル記憶追加
	if subcommand == "add" {
		if len(args) < 2 {
			red.Println("Usage: /memory add <text>")
			return true
		}

		content := strings.Join(args[1:], " ")
		m, err := store.Add(content, false)
		if err != nil {
			red.Printf("Failed to add memory: %v\n", err)
			return true
		}

		green.Printf("✅ Memory added (ID: %s)\n", m.ID)
		green.Printf("   Scope: Global\n")
		green.Printf("   Content: %s\n", m.Content)
		return true
	}

	// /memory project <text> → プロジェクト別記憶追加
	if subcommand == "project" {
		if len(args) < 2 {
			red.Println("Usage: /memory project <text>")
			return true
		}

		if store.ProjectPath == "" {
			red.Println("Not in a project directory (.xelyon not found)")
			yellow.Println("Hint: Create .xelyon directory in project root, or use /memory add for global memory")
			return true
		}

		content := strings.Join(args[1:], " ")
		m, err := store.Add(content, true)
		if err != nil {
			red.Printf("Failed to add memory: %v\n", err)
			return true
		}

		green.Printf("✅ Memory added (ID: %s)\n", m.ID)
		green.Printf("   Scope: Project (%s)\n", m.Project)
		green.Printf("   Content: %s\n", m.Content)
		return true
	}

	// /memory delete <id> → 記憶削除
	if subcommand == "delete" {
		if len(args) < 2 {
			red.Println("Usage: /memory delete <id>")
			return true
		}

		id := args[1]
		if err := store.Delete(id); err != nil {
			red.Printf("Failed to delete memory: %v\n", err)
			return true
		}

		green.Printf("✅ Memory deleted (ID: %s)\n", id)
		return true
	}

	// /memory clear → 全記憶削除
	if subcommand == "clear" {
		// 確認プロンプト
		cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		cyan.Printf("⚠️  Clear All Memories / 全記憶削除\n")
		cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Printf("削除する記憶数: %d 件\n", len(store.List()))
		yellow.Println("\n⚠️  Warning: すべての記憶が削除されます（復元不可）")

		// 確認
		if !promptConfirm("\nContinue? (y/n): ") {
			yellow.Println("Cancelled")
			return true
		}

		// 全削除実行
		if err := store.Clear(false); err != nil {
			red.Printf("Failed to clear memories: %v\n", err)
			return true
		}

		green.Println("✅ All memories cleared")
		return true
	}

	// /memory <text> → グローバル記憶追加（ショートカット）
	content := strings.Join(args, " ")
	m, err := store.Add(content, false)
	if err != nil {
		red.Printf("Failed to add memory: %v\n", err)
		return true
	}

	green.Printf("✅ Memory added (ID: %s)\n", m.ID)
	green.Printf("   Scope: Global\n")
	green.Printf("   Content: %s\n", m.Content)
	return true
}
