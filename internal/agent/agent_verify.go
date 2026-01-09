package agent

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// VerifyResult は検証結果
type VerifyResult struct {
	NeedsVerify  bool     // 検証が必要か
	FileType     string   // "go", "js", "py" など
	ChangedFiles []string // 変更されたファイル
	FmtResult    string   // go fmt の結果
	TestResult   string   // go test の結果
	TestPassed   bool     // テスト成功したか
}

// ShouldVerify は変更されたファイルが検証対象かを判定
func ShouldVerify(filePath string) *VerifyResult {
	ext := strings.ToLower(filepath.Ext(filePath))
	result := &VerifyResult{
		ChangedFiles: []string{filePath},
	}

	switch ext {
	case ".go":
		result.NeedsVerify = true
		result.FileType = "go"
	// 将来的に .js, .py なども追加可能
	default:
		result.NeedsVerify = false
	}
	return result
}

// RunGoFmt は go fmt を実行
func RunGoFmt(filePath string) (string, error) {
	cmd := exec.Command("go", "fmt", filePath)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

// RunGoTest は go test を実行（該当パッケージのみ）
func RunGoTest(filePath string) (string, bool, error) {
	dir := filepath.Dir(filePath)
	cmd := exec.Command("go", "test", "-v", "./"+dir+"/...")
	output, err := cmd.CombinedOutput()

	// exit code 0 なら成功
	passed := err == nil
	return string(output), passed, nil // エラーはテスト失敗も含むので nil 返す
}

// CheckGoModExists は go.mod があるか確認
func CheckGoModExists() bool {
	_, err := exec.Command("go", "list", "-m").CombinedOutput()
	return err == nil
}

// suggestVerification はGoファイル変更後の検証を提案・実行
func (a *Agent) suggestVerification(filePath string, vr *VerifyResult) {
	if vr.FileType != "go" {
		return
	}

	// go.mod存在チェック
	if !CheckGoModExists() {
		return // Goプロジェクトじゃない
	}

	fmt.Println()
	yellow.Println("🔍 Go file changed. Run verification?")
	fmt.Printf("   File: %s\n", filePath)
	yellow.Print("   Run go fmt + go test? (y/n/f=fmt only): ")

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return
	}

	input = strings.TrimSpace(strings.ToLower(input))

	switch input {
	case "y", "yes":
		a.runVerification(filePath, true, true)
	case "f", "fmt":
		a.runVerification(filePath, true, false)
	default:
		yellow.Println("   Skipped verification")
	}
}

// runVerification は実際に検証を実行
func (a *Agent) runVerification(filePath string, runFmt, runTest bool) {
	if runFmt {
		cyan.Println("\n📝 Running go fmt...")
		output, err := RunGoFmt(filePath)
		if err != nil {
			red.Printf("   go fmt failed: %v\n", err)
		} else if output == "" {
			green.Println("   ✅ Already formatted")
		} else {
			green.Printf("   ✅ Formatted: %s\n", output)
		}
	}

	if runTest {
		cyan.Println("\n🧪 Running go test...")
		output, passed, _ := RunGoTest(filePath)

		// 出力が長い場合は省略
		lines := strings.Split(output, "\n")
		if len(lines) > 20 {
			for _, line := range lines[:10] {
				fmt.Println("   " + line)
			}
			yellow.Printf("   ... (%d lines omitted)\n", len(lines)-20)
			for _, line := range lines[len(lines)-10:] {
				fmt.Println("   " + line)
			}
		} else {
			for _, line := range lines {
				if line != "" {
					fmt.Println("   " + line)
				}
			}
		}

		if passed {
			green.Println("   ✅ All tests passed")
		} else {
			red.Println("   ❌ Tests failed")
			a.suggestRollback()
		}
	}
}

// suggestRollback はテスト失敗時にrollbackを提案
func (a *Agent) suggestRollback() {
	if len(a.changeStack) == 0 {
		return
	}

	yellow.Print("\n   Rollback the change? (y/n): ")

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return
	}

	input = strings.TrimSpace(strings.ToLower(input))
	if input == "y" || input == "yes" {
		handleUndoCommand(a)
	}
}
