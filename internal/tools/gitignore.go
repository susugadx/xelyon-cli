package tools

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

var (
	gitignoreCheckedLock sync.Mutex              // スレッドセーフ
	gitignoreAddedFlag   = make(map[string]bool) // ディレクトリごとの追加済みフラグ
)

// ensureGitignore は .gitignore に *.bak* パターンを追加（初回のみ）
func ensureGitignore(dir string) error {
	gitignoreCheckedLock.Lock()
	defer gitignoreCheckedLock.Unlock()

	// すでにこのディレクトリで確認済みの場合はスキップ
	if gitignoreAddedFlag[dir] {
		return nil
	}

	// .gitignore のパスを決定（リポジトリルートまたはカレントディレクトリ）
	gitignorePath := findGitignorePath(dir)
	if gitignorePath == "" {
		// .gitがない、または.gitignoreを作成する場所が不明
		gitignoreAddedFlag[dir] = true
		return nil
	}

	// .gitignore が存在するか確認
	exists := fileExists(gitignorePath)

	// .gitignore にすでに *.bak* パターンが含まれているかチェック
	if exists {
		hasPattern, err := gitignoreHasBackupPattern(gitignorePath)
		if err != nil {
			return err
		}
		if hasPattern {
			// すでに追加済み
			gitignoreAddedFlag[dir] = true
			return nil
		}
	}

	// ユーザーに確認
	fmt.Println()
	yellow.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	yellow.Printf("📝 .gitignore にバックアップファイルを追加\n")
	yellow.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("バックアップファイル (*.bak*) を .gitignore に追加しますか？\n")
	fmt.Printf("場所: %s\n", gitignorePath)
	yellow.Print("\nAdd to .gitignore? (y/n): ")

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return err
	}

	input = strings.TrimSpace(strings.ToLower(input))
	if input != "y" && input != "yes" {
		yellow.Println("Skipped")
		gitignoreAddedFlag[dir] = true // 拒否されたので再度聞かない
		return nil
	}

	// .gitignore に追加
	if err := addBackupPatternsToGitignore(gitignorePath); err != nil {
		return err
	}

	green.Printf("✅ .gitignore に *.bak* パターンを追加しました\n")
	yellow.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	gitignoreAddedFlag[dir] = true
	return nil
}

// findGitignorePath は .gitignore のパスを検索（リポジトリルートから）
func findGitignorePath(startDir string) string {
	// リポジトリルートを探す
	repoRoot := findGitRoot(startDir)
	if repoRoot != "" {
		return filepath.Join(repoRoot, ".gitignore")
	}

	// .git が見つからない場合はカレントディレクトリ
	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}
	return filepath.Join(cwd, ".gitignore")
}

// findGitRoot は .git ディレクトリのあるルートを探す
func findGitRoot(startDir string) string {
	dir := startDir
	for {
		gitPath := filepath.Join(dir, ".git")
		if info, err := os.Stat(gitPath); err == nil && info.IsDir() {
			return dir
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			// ルートディレクトリに到達
			break
		}
		dir = parent
	}
	return ""
}

// gitignoreHasBackupPattern は .gitignore に *.bak* パターンが含まれているかチェック
func gitignoreHasBackupPattern(gitignorePath string) (bool, error) {
	file, err := os.Open(gitignorePath)
	if err != nil {
		return false, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// コメント行はスキップ
		if strings.HasPrefix(line, "#") {
			continue
		}
		// *.bak または *.bak.* パターンをチェック
		if line == "*.bak" || line == "*.bak.*" || strings.Contains(line, "*.bak") {
			return true, nil
		}
	}

	return false, scanner.Err()
}

// addBackupPatternsToGitignore は .gitignore に *.bak* パターンを追加
func addBackupPatternsToGitignore(gitignorePath string) error {
	// ファイルが存在しない場合は新規作成
	exists := fileExists(gitignorePath)

	var content string
	if exists {
		data, err := os.ReadFile(gitignorePath)
		if err != nil {
			return err
		}
		content = string(data)

		// 末尾に改行がない場合は追加
		if !strings.HasSuffix(content, "\n") {
			content += "\n"
		}
	}

	// パターンを追加
	content += "\n# XELYON CLI backup files\n*.bak\n*.bak.*\n"

	// ファイルに書き込み
	if err := os.WriteFile(gitignorePath, []byte(content), 0644); err != nil {
		return err
	}

	return nil
}
