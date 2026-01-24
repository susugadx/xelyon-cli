package dev

import (
	"fmt"
	"os/exec"
	"path/filepath"
)

// ExecuteFormat runs formatter with auto-detection
func ExecuteFormat(path string) (string, string, error) {
	if path == "" {
		path = "."
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Sprintf("Error: %v", err), "", nil
	}

	formatter, command := DetectFormatter(absPath)

	if formatter == "" {
		return `No formatter detected.

Supported formatters:
  - Go: *.go files → go fmt ./...
  - JavaScript/TypeScript: .prettierrc → prettier --write .
  - Python: *.py files → black . or autopep8
  - Rust: Cargo.toml → cargo fmt

Please ensure your project has the appropriate files or configuration.`, "", nil
	}

	cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	cyan.Printf("✨ Formatting Code / コード整形\n")
	cyan.Printf("📂 Path / パス: %s\n", path)
	cyan.Printf("🔧 Formatter / フォーマッター: %s\n", formatter)
	cyan.Printf("⚙️  Command / コマンド: %s\n", command)
	cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	green.Printf("▶ Running: %s\n", command)

	cmd := exec.Command("bash", "-c", command)
	cmd.Dir = absPath
	output, err := cmd.CombinedOutput()

	result := string(output)
	if len(result) > 2000 {
		result = result[:2000] + "\n... (truncated)"
	}

	if err != nil {
		return fmt.Sprintf("Formatter failed:\n%s\n\nError: %v", result, err), "", nil
	}

	green.Println("\n✅ Formatting completed")

	return fmt.Sprintf("Successfully formatted code:\n%s", result), "", nil
}
