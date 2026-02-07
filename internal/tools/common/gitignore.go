package common

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

// EnsureGitignore は .gitignore に *.bak* パターンを追加（初回のみ）
func EnsureGitignore(dir string) error {
	gitignoreCheckedLock.Lock()
	defer gitignoreCheckedLock.Unlock()

	// すでにこのディレクトリで確認済みの場合はスキップ
	if gitignoreAddedFlag[dir] {
		return nil
	}

	// .gitignore のパスを決定（リポジトリルートまたはカレントディレクトリ）
	gitignorePath := FindGitignorePath(dir)
	if gitignorePath == "" {
		// .gitがない、または.gitignoreを作成する場所が不明
		gitignoreAddedFlag[dir] = true
		return nil
	}

	// .gitignore が存在するか確認
	exists := FileExists(gitignorePath)

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

	// CI/テスト実行では .gitignore 追記の対話プロンプトを出さない
	if v := strings.TrimSpace(os.Getenv("XELYON_SKIP_GITIGNORE_PROMPT")); v != "" && v != "0" {
		gitignoreAddedFlag[dir] = true
		return nil
	}

	fmt.Println()
	Yellow.Println("--------------------------------------------")
	Yellow.Printf("Add backup files to .gitignore\n")
	Yellow.Println("--------------------------------------------")
	fmt.Printf("Add backup files (*.bak*) to .gitignore?\n")
	fmt.Printf("Location: %s\n", gitignorePath)
	Yellow.Print("\nAdd to .gitignore? (y/n): ")

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return err
	}

	input = stripBracketedPaste(input)
	input = strings.TrimSpace(strings.ToLower(input))
	if input != "y" && input != "yes" {
		Yellow.Println("Skipped")
		gitignoreAddedFlag[dir] = true // 拒否されたので再度聞かない
		return nil
	}

	// .gitignore に追加
	if err := addBackupPatternsToGitignore(gitignorePath); err != nil {
		return err
	}

	Green.Printf("Added *.bak* pattern to .gitignore\n")
	Yellow.Println("--------------------------------------------")
	fmt.Println()

	gitignoreAddedFlag[dir] = true
	return nil
}

// FindGitignorePath は .gitignore のパスを検索（リポジトリルートから）
func FindGitignorePath(startDir string) string {
	// リポジトリルートを探す
	repoRoot := FindGitRoot(startDir)
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

// FindGitRoot は .git ディレクトリのあるルートを探す
func FindGitRoot(startDir string) string {
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
	exists := FileExists(gitignorePath)

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
