package agent

import (
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
