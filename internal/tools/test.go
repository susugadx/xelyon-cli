package tools

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/susugadx/xelyon-cli/internal/ui"
)

// executeRunTest はテストを自動検出して実行
func executeRunTest(path string) string {
	if path == "" {
		path = "."
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}

	framework, command := detectTestFramework(absPath)

	if framework == "" {
		return `No test framework detected.

Supported frameworks:
  - Go: go.mod → go test ./...
  - JavaScript/TypeScript: package.json → npm test
  - Python: pytest.ini/setup.py → pytest
  - Rust: Cargo.toml → cargo test

Please ensure your project has the appropriate configuration files.`
	}

	cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	cyan.Printf("🧪 Running Tests / テスト実行\n")
	cyan.Printf("📂 Path / パス: %s\n", path)
	cyan.Printf("🔧 Framework / フレームワーク: %s\n", framework)
	cyan.Printf("⚙️  Command / コマンド: %s\n", command)
	cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	green.Printf("▶ Running: %s\n", command)

	spinner := ui.NewSpinnerWithWriter(os.Stderr)
	spinner.Start("Running tests...")
	defer spinner.Stop()

	cmd := exec.Command("bash", "-c", command)
	cmd.Dir = absPath
	output, err := cmd.CombinedOutput()

	result := string(output)

	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		}
	}

	if exitCode == 0 {
		green.Printf("\n✅ Tests passed (exit code: %d)\n", exitCode)
	} else {
		red.Printf("\n❌ Tests failed (exit code: %d)\n", exitCode)
	}

	return fmt.Sprintf("%s\n\nExit code: %d", result, exitCode)
}
