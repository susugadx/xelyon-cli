//go:build ignore
// +build ignore

package main

import (
	"fmt"

	"github.com/susugadx/xelyon-cli/internal/tools/common"
)

// このデモはinteractive confirm機能を示します
// 実行方法:
//   go run examples/interactive_confirm_demo.go

func main() {
	fmt.Println("=== Interactive Confirm Demo ===")
	fmt.Println()
	fmt.Println("This demonstrates the new y/n/c confirmation system:")
	fmt.Println("  y - Execute")
	fmt.Println("  n - Cancel")
	fmt.Println("  c - Comment (provide feedback for revision)")
	fmt.Println()

	// デモ：対話的確認
	result := common.ConfirmInteractive("Would you like to proceed with this demo?")

	switch result.Action {
	case "yes":
		fmt.Println("\n✅ User approved!")

	case "no":
		fmt.Println("\n❌ User cancelled.")

	case "comment":
		fmt.Println("\n💬 User provided feedback:")
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Println(result.Comment)
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Println()
		fmt.Println("In a real scenario, this feedback would be sent to the AI")
		fmt.Println("for revision.")
	}
}
