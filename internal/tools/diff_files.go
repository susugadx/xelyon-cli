package tools

import (
	"fmt"
	"os"
	"strings"

	"github.com/sergi/go-diff/diffmatchpatch"
)

// executeDiffFiles は2つのファイルの差分を表示
// file1: 比較元ファイル
// file2: 比較先ファイル
// context: 差分前後の表示行数（デフォルト: 3）
func executeDiffFiles(file1, file2 string, contextLines int) (string, error) {
	if file1 == "" || file2 == "" {
		return "", fmt.Errorf("both file1 and file2 are required")
	}

	// ファイル1を読み込み
	content1, err := os.ReadFile(file1)
	if err != nil {
		return "", fmt.Errorf("failed to read file1 '%s': %w", file1, err)
	}

	// ファイル2を読み込み
	content2, err := os.ReadFile(file2)
	if err != nil {
		return "", fmt.Errorf("failed to read file2 '%s': %w", file2, err)
	}

	text1 := string(content1)
	text2 := string(content2)

	// 同じ場合
	if text1 == text2 {
		return fmt.Sprintf("Files '%s' and '%s' are identical", file1, file2), nil
	}

	// 差分を計算
	dmp := diffmatchpatch.New()
	diffs := dmp.DiffMain(text1, text2, true)

	// 人間が読みやすい形式に整形
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("--- %s\n", file1))
	sb.WriteString(fmt.Sprintf("+++ %s\n", file2))
	sb.WriteString("\n")

	// unified diff 形式で出力
	result := generateUnifiedDiff(text1, text2, file1, file2, contextLines)
	if result == "" {
		// フォールバック: シンプルな差分表示
		for _, diff := range diffs {
			switch diff.Type {
			case diffmatchpatch.DiffDelete:
				for _, line := range strings.Split(diff.Text, "\n") {
					if line != "" {
						sb.WriteString(fmt.Sprintf("- %s\n", line))
					}
				}
			case diffmatchpatch.DiffInsert:
				for _, line := range strings.Split(diff.Text, "\n") {
					if line != "" {
						sb.WriteString(fmt.Sprintf("+ %s\n", line))
					}
				}
			case diffmatchpatch.DiffEqual:
				// コンテキスト行は省略（長い場合）
				lines := strings.Split(diff.Text, "\n")
				if len(lines) > contextLines*2+1 {
					for i := 0; i < contextLines && i < len(lines); i++ {
						if lines[i] != "" {
							sb.WriteString(fmt.Sprintf("  %s\n", lines[i]))
						}
					}
					sb.WriteString(fmt.Sprintf("  ... (%d lines omitted) ...\n", len(lines)-contextLines*2))
					for i := len(lines) - contextLines; i < len(lines); i++ {
						if i >= 0 && lines[i] != "" {
							sb.WriteString(fmt.Sprintf("  %s\n", lines[i]))
						}
					}
				} else {
					for _, line := range lines {
						if line != "" {
							sb.WriteString(fmt.Sprintf("  %s\n", line))
						}
					}
				}
			}
		}
		return sb.String(), nil
	}

	return result, nil
}

// generateUnifiedDiff は unified diff 形式の差分を生成
func generateUnifiedDiff(text1, text2, file1, file2 string, contextLines int) string {
	if contextLines <= 0 {
		contextLines = 3
	}

	lines1 := strings.Split(text1, "\n")
	lines2 := strings.Split(text2, "\n")

	// LCS (Longest Common Subsequence) ベースの差分
	// 簡易実装: 行単位で比較
	var result strings.Builder
	result.WriteString(fmt.Sprintf("--- %s\n", file1))
	result.WriteString(fmt.Sprintf("+++ %s\n", file2))

	// 差分のある行を見つける
	type change struct {
		line1Start int
		line1End   int
		line2Start int
		line2End   int
	}

	var changes []change
	i, j := 0, 0
	for i < len(lines1) || j < len(lines2) {
		// 同じ行をスキップ
		for i < len(lines1) && j < len(lines2) && lines1[i] == lines2[j] {
			i++
			j++
		}

		if i >= len(lines1) && j >= len(lines2) {
			break
		}

		// 差分の開始
		startI, startJ := i, j

		// 次の一致点を探す
		found := false
		for lookAhead := 1; lookAhead < 100 && !found; lookAhead++ {
			// lines1を進める
			if i+lookAhead < len(lines1) {
				for k := j; k < len(lines2) && k < j+lookAhead+1; k++ {
					if lines1[i+lookAhead] == lines2[k] {
						i += lookAhead
						j = k
						found = true
						break
					}
				}
			}
			// lines2を進める
			if !found && j+lookAhead < len(lines2) {
				for k := i; k < len(lines1) && k < i+lookAhead+1; k++ {
					if lines1[k] == lines2[j+lookAhead] {
						i = k
						j += lookAhead
						found = true
						break
					}
				}
			}
		}

		if !found {
			// 残り全部が差分
			i = len(lines1)
			j = len(lines2)
		}

		if startI < i || startJ < j {
			changes = append(changes, change{startI, i, startJ, j})
		}
	}

	// 変更をまとめて出力
	for _, c := range changes {
		// ハンク ヘッダー
		hunkStart1 := c.line1Start - contextLines
		if hunkStart1 < 0 {
			hunkStart1 = 0
		}
		hunkEnd1 := c.line1End + contextLines
		if hunkEnd1 > len(lines1) {
			hunkEnd1 = len(lines1)
		}

		hunkStart2 := c.line2Start - contextLines
		if hunkStart2 < 0 {
			hunkStart2 = 0
		}
		hunkEnd2 := c.line2End + contextLines
		if hunkEnd2 > len(lines2) {
			hunkEnd2 = len(lines2)
		}

		result.WriteString(fmt.Sprintf("@@ -%d,%d +%d,%d @@\n",
			hunkStart1+1, hunkEnd1-hunkStart1,
			hunkStart2+1, hunkEnd2-hunkStart2))

		// コンテキスト（前）
		for k := hunkStart1; k < c.line1Start && k < len(lines1); k++ {
			result.WriteString(fmt.Sprintf(" %s\n", lines1[k]))
		}

		// 削除行
		for k := c.line1Start; k < c.line1End && k < len(lines1); k++ {
			result.WriteString(fmt.Sprintf("-%s\n", lines1[k]))
		}

		// 追加行
		for k := c.line2Start; k < c.line2End && k < len(lines2); k++ {
			result.WriteString(fmt.Sprintf("+%s\n", lines2[k]))
		}

		// コンテキスト（後）
		for k := c.line1End; k < hunkEnd1 && k < len(lines1); k++ {
			result.WriteString(fmt.Sprintf(" %s\n", lines1[k]))
		}
	}

	if len(changes) == 0 {
		return ""
	}

	return result.String()
}
