package tools

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// GrepReplaceResult は一括置換の結果
type GrepReplaceResult struct {
	FilesModified int
	TotalMatches  int
	FileResults   []FileReplaceResult
	BackupPaths   map[string]string // filepath -> backupPath
}

// FileReplaceResult は1ファイルの置換結果
type FileReplaceResult struct {
	FilePath   string
	MatchCount int
	BackupPath string
}

// executeGrepReplace は複数ファイルで一括置換を実行
// pattern: 検索パターン（正規表現）
// replacement: 置換文字列
// path: 検索対象ディレクトリ（デフォルト: "."）
// filePattern: ファイル名パターン（例: "*.go", デフォルト: 全ファイル）
// dryRun: trueの場合、実際の置換は行わずプレビューのみ
func executeGrepReplace(pattern, replacement, path, filePattern string, dryRun bool) (string, []FileReplaceResult, error) {
	if pattern == "" {
		return "", nil, fmt.Errorf("pattern is required")
	}
	if replacement == "" && !dryRun {
		// 空文字への置換は許可（削除として機能）
	}

	if path == "" {
		path = "."
	}
	if filePattern == "" {
		filePattern = "*"
	}

	// 正規表現をコンパイル
	re, err := regexp.Compile(pattern)
	if err != nil {
		return "", nil, fmt.Errorf("invalid regex pattern: %w", err)
	}

	// 対象ファイルを収集
	var targetFiles []string
	err = filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // エラーがあってもスキップして続行
		}

		// ディレクトリはスキップ
		if info.IsDir() {
			// .git, node_modules, vendor などはスキップ
			base := filepath.Base(filePath)
			if base == ".git" || base == "node_modules" || base == "vendor" || base == ".bak" {
				return filepath.SkipDir
			}
			return nil
		}

		// バイナリファイル、バックアップファイルはスキップ
		if strings.Contains(filePath, ".bak.") {
			return nil
		}

		// ファイルパターンにマッチするか確認
		matched, _ := filepath.Match(filePattern, filepath.Base(filePath))
		if !matched && filePattern != "*" {
			return nil
		}

		// コードファイルのみ対象
		ext := strings.ToLower(filepath.Ext(filePath))
		codeExts := map[string]bool{
			".go": true, ".js": true, ".ts": true, ".tsx": true, ".jsx": true,
			".py": true, ".rb": true, ".java": true, ".c": true, ".cpp": true,
			".h": true, ".hpp": true, ".rs": true, ".swift": true, ".kt": true,
			".md": true, ".json": true, ".yaml": true, ".yml": true, ".toml": true,
			".html": true, ".css": true, ".scss": true, ".sql": true, ".sh": true,
			".txt": true, ".xml": true, ".vue": true, ".svelte": true,
		}
		if !codeExts[ext] && filePattern == "*" {
			return nil
		}

		targetFiles = append(targetFiles, filePath)
		return nil
	})

	if err != nil {
		return "", nil, fmt.Errorf("failed to walk directory: %w", err)
	}

	if len(targetFiles) == 0 {
		return "No files found matching the criteria", nil, nil
	}

	// 各ファイルを処理
	var results []FileReplaceResult
	totalMatches := 0
	filesModified := 0

	for _, filePath := range targetFiles {
		content, err := os.ReadFile(filePath)
		if err != nil {
			continue // 読み取りエラーはスキップ
		}

		// マッチ数をカウント
		matches := re.FindAllStringIndex(string(content), -1)
		if len(matches) == 0 {
			continue
		}

		totalMatches += len(matches)

		if dryRun {
			// ドライランの場合はプレビューのみ
			results = append(results, FileReplaceResult{
				FilePath:   filePath,
				MatchCount: len(matches),
			})
			continue
		}

		// バックアップを作成
		backupPath, err := createBackup(filePath)
		if err != nil {
			yellow.Printf("Warning: Failed to create backup for %s: %v\n", filePath, err)
			continue
		}

		// 置換を実行
		newContent := re.ReplaceAllString(string(content), replacement)

		// ファイルに書き込み
		if err := os.WriteFile(filePath, []byte(newContent), 0644); err != nil {
			red.Printf("Error: Failed to write %s: %v\n", filePath, err)
			continue
		}

		filesModified++
		results = append(results, FileReplaceResult{
			FilePath:   filePath,
			MatchCount: len(matches),
			BackupPath: backupPath,
		})
	}

	// 結果サマリーを生成
	var sb strings.Builder

	if dryRun {
		sb.WriteString("🔍 Dry Run - Preview of changes:\n")
	} else {
		sb.WriteString("✅ Replacement completed:\n")
	}

	sb.WriteString(fmt.Sprintf("  Pattern: %s\n", pattern))
	sb.WriteString(fmt.Sprintf("  Replacement: %s\n", replacement))
	sb.WriteString(fmt.Sprintf("  Files: %d\n", len(results)))
	sb.WriteString(fmt.Sprintf("  Total matches: %d\n", totalMatches))
	sb.WriteString("\n")

	// ファイルごとの結果（最大20件）
	displayCount := len(results)
	if displayCount > 20 {
		displayCount = 20
	}

	for i := 0; i < displayCount; i++ {
		r := results[i]
		sb.WriteString(fmt.Sprintf("  %s (%d matches)\n", r.FilePath, r.MatchCount))
	}

	if len(results) > 20 {
		sb.WriteString(fmt.Sprintf("  ... and %d more files\n", len(results)-20))
	}

	return sb.String(), results, nil
}
