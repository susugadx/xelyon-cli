package dev

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/tools/common"
)

const (
	artifactsDir   = ".xelyon/artifacts"
	tailLineCount  = 30
	maxArtifactAge = 24 * time.Hour
)

// TruncateWithFile は長い出力をファイルに保存し、末尾30行を返す。
// OutputTruncateLen 以下の場合はそのまま返す。
// 超過時は .xelyon/artifacts/output_<unixnano>.txt に全文を保存する。
func TruncateWithFile(output string) string {
	if len(output) <= config.OutputTruncateLen {
		return output
	}

	// .xelyon/artifacts/ ディレクトリを作成
	if err := os.MkdirAll(artifactsDir, 0755); err != nil {
		// ディレクトリ作成に失敗した場合はフォールバック
		return output[:config.OutputTruncateLen] + "\n... (truncated)"
	}

	// ファイルに全文保存
	filename := fmt.Sprintf("output_%d.txt", time.Now().UnixNano())
	relPath := filepath.Join(artifactsDir, filename)
	if err := os.WriteFile(relPath, []byte(output), 0600); err != nil {
		return output[:config.OutputTruncateLen] + "\n... (truncated)"
	}

	// 行数とサイズを計算
	lineCount := strings.Count(output, "\n") + 1
	sizeKB := len(output) / 1024

	// 末尾30行を取得
	tail := lastNLines(output, tailLineCount)

	return fmt.Sprintf("Output saved: %s (%d lines, %dKB)\nLast %d lines:\n%s\n\nUse `read_file(paths=[%q])` to read specific sections.",
		relPath, lineCount, sizeKB, tailLineCount, tail, relPath+":1-50")
}

// CleanupArtifacts は24時間以上前のartifactファイルを削除する。
// 起動時に呼び出す。artifactsディレクトリが存在する場合のみ .gitignore もチェックする。
func CleanupArtifacts() {
	CleanupArtifactsWithWriter(common.DefaultOutput().StdoutWriter())
}

// CleanupArtifactsWithWriter は警告出力先を指定して古いartifactファイルを削除する。
func CleanupArtifactsWithWriter(w io.Writer) {
	if w == nil {
		w = io.Discard
	}
	entries, err := os.ReadDir(artifactsDir)
	if err != nil {
		return // ディレクトリがなければ何もしない
	}

	// artifactsディレクトリが存在する場合のみ .gitignore チェック
	checkGitignoreWithWriter(w)

	now := time.Now()
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		if now.Sub(info.ModTime()) > maxArtifactAge {
			path := filepath.Join(artifactsDir, entry.Name())
			if err := os.Remove(path); err != nil {
				_, _ = fmt.Fprintf(w, "Warning: failed to remove old artifact %s: %v\n", path, err)
			}
		}
	}
}

// checkGitignore は .gitignore に .xelyon/ が含まれているかチェックし、
// なければ警告を出力する。
func checkGitignore() {
	checkGitignoreWithWriter(common.DefaultOutput().StdoutWriter())
}

func checkGitignoreWithWriter(w io.Writer) {
	if w == nil {
		w = io.Discard
	}
	f, err := os.Open(".gitignore")
	if err != nil {
		_, _ = fmt.Fprintln(w, "Warning: .xelyon/ is not in .gitignore. Add it to avoid committing artifacts.")
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == ".xelyon/" || line == ".xelyon" {
			return
		}
	}
	_, _ = fmt.Fprintln(w, "Warning: .xelyon/ is not in .gitignore. Add it to avoid committing artifacts.")
}

// lastNLines は文字列の末尾N行を返す。
func lastNLines(s string, n int) string {
	lines := strings.Split(s, "\n")
	if len(lines) <= n {
		return s
	}
	return strings.Join(lines[len(lines)-n:], "\n")
}
