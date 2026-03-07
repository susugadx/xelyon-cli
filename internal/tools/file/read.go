package file

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/tools/common"
)

// formatFileSize はバイト数を人間が読みやすい形式に変換
func formatFileSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// MaxReadLines はデフォルトの最大読み込み行数
const MaxReadLines = 300

// LargeFileThreshold は「大容量ファイル」と判定するサイズ（1MB）
const LargeFileThreshold = 1024 * 1024

// ExecuteReadFile はファイルを読み込む（行範囲指定対応）
// startLine, endLine が指定されている場合はその範囲のみ返す
// 指定がない場合は最初のMaxReadLines行を返す
func ExecuteReadFile(path string, startLine, endLine int) string {
	if path == "" {
		return "Error: path is empty"
	}

	// パストラバーサル防止
	absPath, err := common.ValidatePath(path)
	if err != nil {
		common.Red.Printf("🚫 Security: %v\n", err)
		return fmt.Sprintf("Error: %v", err)
	}

	// 設定読み込み（ファイル情報表示用）
	cfg, _ := config.LoadConfig()
	showFileInfo := cfg != nil && cfg.Streaming.ShowFileInfo

	// ファイル情報を取得（サイズ表示用）
	var fileInfo os.FileInfo
	var fileSize int64
	if showFileInfo {
		if info, err := os.Stat(absPath); err == nil {
			fileInfo = info
			fileSize = info.Size()
		}
	}

	var contentStr string

	// キャッシュチェック（行範囲指定なしの場合のみ）
	if startLine == 0 && endLine == 0 && tools.GlobalToolCache != nil {
		if cached, hit := tools.GlobalToolCache.GetFile(absPath); hit {
			contentStr = cached
		}
	}

	// キャッシュミスまたは行範囲指定ありの場合はファイルを読む
	if contentStr == "" {
		if fileInfo == nil {
			info, err := os.Stat(absPath)
			if err != nil {
				return fmt.Sprintf("Error reading file: %v", err)
			}
			fileInfo = info
			if showFileInfo {
				fileSize = info.Size()
			}
		}

		f, err := os.Open(absPath)
		if err != nil {
			return fmt.Sprintf("Error reading file: %v", err)
		}
		defer f.Close()

		// 先頭 512 バイトでバイナリ判定
		header := make([]byte, 512)
		n, _ := f.Read(header)
		if strings.Contains(string(header[:n]), "\x00") {
			return fmt.Sprintf("Error: %s appears to be a binary file (contains null bytes). Use 'file %s' or 'xxd %s | head' for binary inspection.", path, path, path)
		}

		// 行範囲指定なし + 大容量ファイルはストリーミング読み込み
		if startLine == 0 && endLine == 0 && fileInfo.Size() > LargeFileThreshold {
			if _, err := f.Seek(0, 0); err != nil {
				return fmt.Sprintf("Error reading file: %v", err)
			}
			lines, totalRead, hasMore, err := readFirstNLines(f, MaxReadLines)
			if err != nil {
				return fmt.Sprintf("Error reading file: %v", err)
			}

			if hasMore {
				// 推定総行数
				totalLines := totalRead
				if totalRead > 0 {
					avgLineLen := fileInfo.Size() / int64(totalRead)
					if avgLineLen > 0 {
						totalLines = int(fileInfo.Size() / avgLineLen)
					}
				}
				// outline-first モード（先頭300行分の content で BuildBlockMap）
				result := formatOutline(absPath, lines, totalLines)
				if showFileInfo && fileSize > 0 {
					common.Green.Printf("📄 Read: %s (%s, outline of ~%d lines)\n", path, formatFileSize(fileSize), totalLines)
				} else {
					common.Green.Printf("📄 Read: %s (outline of ~%d lines)\n", path, totalLines)
				}
				return result
			}
			// 300行以下に収まった場合は全行表示
			result := formatLinesWithNumbers(lines, 1)
			if showFileInfo && fileSize > 0 {
				common.Green.Printf("📄 Read: %s (%s, %d lines)\n", path, formatFileSize(fileSize), len(lines))
			} else {
				common.Green.Printf("📄 Read: %s (%d lines)\n", path, len(lines))
			}
			return result
		}

		if _, err := f.Seek(0, 0); err != nil {
			return fmt.Sprintf("Error reading file: %v", err)
		}
		content, err := io.ReadAll(f)
		if err != nil {
			return fmt.Sprintf("Error reading file: %v", err)
		}
		contentStr = string(content)

		// キャッシュに保存（行範囲指定なしの場合のみ）
		if startLine == 0 && endLine == 0 && tools.GlobalToolCache != nil {
			tools.GlobalToolCache.SetFile(absPath, contentStr)
		}
	}

	// バイナリファイル検出（先頭 512 バイトに NUL が含まれる場合）
	checkLen := len(contentStr)
	if checkLen > 512 {
		checkLen = 512
	}
	if strings.Contains(contentStr[:checkLen], "\x00") {
		return fmt.Sprintf("Error: %s appears to be a binary file (contains null bytes). Use 'file %s' or 'xxd %s | head' for binary inspection.", path, path, path)
	}

	lines := strings.Split(contentStr, "\n")
	totalLines := len(lines)

	// 行範囲が指定されている場合（start_line のみ、end_line のみ、両方指定 に対応）
	if startLine > 0 || endLine > 0 {
		// 片方のみ指定時のデフォルト補完
		if startLine <= 0 {
			startLine = 1
		}
		if endLine <= 0 {
			endLine = startLine + MaxReadLines - 1
		}

		// 範囲調整
		if startLine > totalLines {
			return fmt.Sprintf("Error: start_line %d exceeds total lines %d", startLine, totalLines)
		}
		if endLine > totalLines {
			endLine = totalLines
		}
		if startLine > endLine {
			return fmt.Sprintf("Error: start_line %d is greater than end_line %d", startLine, endLine)
		}

		// 1-indexed to 0-indexed
		selectedLines := lines[startLine-1 : endLine]
		result := formatLinesWithNumbers(selectedLines, startLine)
		if showFileInfo && fileSize > 0 {
			common.Green.Printf("📄 Read: %s (%s, lines %d-%d of %d)\n", path, formatFileSize(fileSize), startLine, endLine, totalLines)
		} else {
			common.Green.Printf("📄 Read: %s (lines %d-%d of %d)\n", path, startLine, endLine, totalLines)
		}
		return result
	}

	// 行範囲指定なし: デフォルトで最初のMaxReadLines行
	if totalLines <= MaxReadLines {
		// 全行表示
		result := formatLinesWithNumbers(lines, 1)
		if showFileInfo && fileSize > 0 {
			common.Green.Printf("📄 Read: %s (%s, %d lines)\n", path, formatFileSize(fileSize), totalLines)
		} else {
			common.Green.Printf("📄 Read: %s (%d lines)\n", path, totalLines)
		}
		return result
	}

	// outline-first モード: 先頭 + シグネチャ一覧 + 末尾
	result := formatOutline(absPath, lines, totalLines)
	if showFileInfo && fileSize > 0 {
		common.Green.Printf("📄 Read: %s (%s, outline of %d lines)\n", path, formatFileSize(fileSize), totalLines)
	} else {
		common.Green.Printf("📄 Read: %s (outline of %d lines)\n", path, totalLines)
	}
	return result
}

// outlineHeadLines はアウトラインモードで表示する先頭行数
const outlineHeadLines = 30

// outlineTailLines はアウトラインモードで表示する末尾行数
const outlineTailLines = 10

// formatOutline はファイルのアウトラインを生成する。
// 先頭30行 + 関数/メソッドシグネチャ一覧 + 末尾10行 を返す。
func formatOutline(filePath string, lines []string, totalLines int) string {
	var sb strings.Builder

	// 1. 先頭 outlineHeadLines 行
	headEnd := outlineHeadLines
	if headEnd > totalLines {
		headEnd = totalLines
	}
	sb.WriteString(formatLinesWithNumbers(lines[:headEnd], 1))

	// 2. シグネチャ一覧
	content := strings.Join(lines, "\n")
	isBrace := common.IsBraceLanguage(filePath)
	blocks := common.BuildBlockMap(content, isBrace)

	// headEnd 行より後のブロックのみ抽出（先頭部分と重複しないように）
	var signatures []string
	for _, b := range blocks {
		if b.StartLine > headEnd && b.StartLine <= totalLines-outlineTailLines {
			signatures = append(signatures, fmt.Sprintf("  L%-4d %s", b.StartLine, b.Name))
		}
	}

	if len(signatures) > 0 {
		sb.WriteString("\n--- Signatures ---\n")
		for _, sig := range signatures {
			sb.WriteString(sig)
			sb.WriteString("\n")
		}
	}

	// 3. 末尾 outlineTailLines 行
	tailStart := totalLines - outlineTailLines
	if tailStart < headEnd {
		tailStart = headEnd
	}
	if tailStart < totalLines {
		sb.WriteString("\n--- Last lines ---\n")
		sb.WriteString(formatLinesWithNumbers(lines[tailStart:], tailStart+1))
	}

	// 4. ガイドメッセージ
	fmt.Fprintf(&sb, "\n(%d lines total. Use start_line/end_line to read function body)\n", totalLines)

	return sb.String()
}

// formatLinesWithNumbers は行番号付きでフォーマット
func formatLinesWithNumbers(lines []string, startNum int) string {
	var sb strings.Builder
	for i, line := range lines {
		fmt.Fprintf(&sb, "%d: %s\n", startNum+i, line)
	}
	return sb.String()
}

// readFirstNLines は io.Reader から最初の n 行を読み、残りがあるかを返す
func readFirstNLines(r io.Reader, n int) (lines []string, totalRead int, hasMore bool, err error) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		totalRead++
		if totalRead <= n {
			lines = append(lines, scanner.Text())
			continue
		}
		hasMore = true
		return lines, totalRead, hasMore, nil
	}
	if scanErr := scanner.Err(); scanErr != nil {
		return nil, 0, false, scanErr
	}
	return lines, totalRead, false, nil
}
