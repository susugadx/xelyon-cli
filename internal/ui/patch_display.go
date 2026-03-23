package ui

import (
	"fmt"
	"io"
	"strings"

	"github.com/fatih/color"
)

// PatchFileDisplay はファイルごとの変更表示情報。
type PatchFileDisplay struct {
	Path       string
	Action     string // "created" | "modified" | "deleted" | "moved"
	OldContent string
	NewContent string
}

// ShowCodexStyleDiff は Codex CLI 風の差分表示を出力する。
func ShowCodexStyleDiff(out io.Writer, files []PatchFileDisplay) {
	bold := color.New(color.Bold)
	green := color.New(color.FgGreen)
	red := color.New(color.FgRed)
	dim := color.New(color.Faint)

	for _, f := range files {
		switch f.Action {
		case "created":
			showAddedFile(out, bold, green, dim, f)
		case "deleted":
			showDeletedFile(out, bold, red, dim, f)
		case "modified", "moved":
			showModifiedFile(out, bold, green, red, dim, f)
		}
	}
}

func showAddedFile(out io.Writer, bold, green, dim *color.Color, f PatchFileDisplay) {
	lines := strings.Split(f.NewContent, "\n")
	added := len(lines)
	if added > 0 && lines[added-1] == "" {
		added--
	}
	showPatchFileHeader(out, bold, dim, f.Action, f.Path, "", added, 0)

	previewLines := make([]PatchPreviewLine, 0, added)
	for i, line := range lines[:added] {
		previewLines = append(previewLines, PatchPreviewLine{
			Type:    '+',
			LineNum: i + 1,
			Text:    line,
		})
	}
	renderPatchLines(out, green, color.New(color.FgRed), dim, previewLines)
	fmt.Fprintln(out)
}

func showDeletedFile(out io.Writer, bold, red, dim *color.Color, f PatchFileDisplay) {
	_ = red
	lines := strings.Split(f.OldContent, "\n")
	deleted := len(lines)
	if deleted > 0 && lines[deleted-1] == "" {
		deleted--
	}
	showPatchFileHeader(out, bold, dim, f.Action, f.Path, "", 0, deleted)
	fmt.Fprintln(out)
}

func showModifiedFile(out io.Writer, bold, green, red, dim *color.Color, f PatchFileDisplay) {
	oldLines := strings.Split(f.OldContent, "\n")
	newLines := strings.Split(f.NewContent, "\n")

	// 追加/削除行数を計算
	added, removed := countDiffLines(oldLines, newLines)
	showPatchFileHeader(out, bold, dim, f.Action, f.Path, "", added, removed)

	// unified diff を生成して表示
	hunks := computeHunks(oldLines, newLines, 3) // コンテキスト3行
	for i, hunk := range hunks {
		if i > 0 {
			dim.Fprintf(out, "     :\n") // hunk間の省略
		}
		previewLines := make([]PatchPreviewLine, 0, len(hunk.Lines))
		for _, dl := range hunk.Lines {
			switch dl.Type {
			case diffContext:
				previewLines = append(previewLines, PatchPreviewLine{Type: ' ', LineNum: dl.NewLineNum, Text: dl.Text})
			case diffRemoved:
				previewLines = append(previewLines, PatchPreviewLine{Type: '-', LineNum: dl.OldLineNum, Text: dl.Text})
			case diffAdded:
				previewLines = append(previewLines, PatchPreviewLine{Type: '+', LineNum: dl.NewLineNum, Text: dl.Text})
			}
		}
		renderPatchLines(out, green, red, dim, previewLines)
	}
	fmt.Fprintln(out)
}

// --- diff 計算 ---

type diffLineType int

const (
	diffContext diffLineType = iota
	diffAdded
	diffRemoved
)

type diffLine struct {
	Type       diffLineType
	OldLineNum int
	NewLineNum int
	Text       string
}

type diffHunk struct {
	Lines []diffLine
}

func computeHunks(oldLines, newLines []string, contextLines int) []diffHunk {
	ops := computeLCS(oldLines, newLines)
	return groupIntoHunks(ops, oldLines, newLines, contextLines)
}

func countDiffLines(oldLines, newLines []string) (added, removed int) {
	ops := computeLCS(oldLines, newLines)
	for _, op := range ops {
		switch op {
		case opInsert:
			added++
		case opDelete:
			removed++
		}
	}
	return
}

type editOp int

const (
	opEqual editOp = iota
	opInsert
	opDelete
)

func computeLCS(oldLines, newLines []string) []editOp {
	m, n := len(oldLines), len(newLines)
	dp := make([][]int, m+1)
	for i := range dp {
		dp[i] = make([]int, n+1)
	}
	for i := 1; i <= m; i++ {
		for j := 1; j <= n; j++ {
			if oldLines[i-1] == newLines[j-1] {
				dp[i][j] = dp[i-1][j-1] + 1
			} else if dp[i-1][j] >= dp[i][j-1] {
				dp[i][j] = dp[i-1][j]
			} else {
				dp[i][j] = dp[i][j-1]
			}
		}
	}

	var ops []editOp
	i, j := m, n
	for i > 0 || j > 0 {
		if i > 0 && j > 0 && oldLines[i-1] == newLines[j-1] {
			ops = append(ops, opEqual)
			i--
			j--
		} else if j > 0 && (i == 0 || dp[i][j-1] >= dp[i-1][j]) {
			ops = append(ops, opInsert)
			j--
		} else {
			ops = append(ops, opDelete)
			i--
		}
	}

	for l, r := 0, len(ops)-1; l < r; l, r = l+1, r-1 {
		ops[l], ops[r] = ops[r], ops[l]
	}
	return ops
}

func groupIntoHunks(ops []editOp, oldLines, newLines []string, ctxLines int) []diffHunk {
	var allLines []diffLine
	oldIdx, newIdx := 0, 0
	for _, op := range ops {
		switch op {
		case opEqual:
			allLines = append(allLines, diffLine{
				Type:       diffContext,
				OldLineNum: oldIdx + 1,
				NewLineNum: newIdx + 1,
				Text:       oldLines[oldIdx],
			})
			oldIdx++
			newIdx++
		case opDelete:
			allLines = append(allLines, diffLine{
				Type:       diffRemoved,
				OldLineNum: oldIdx + 1,
				Text:       oldLines[oldIdx],
			})
			oldIdx++
		case opInsert:
			allLines = append(allLines, diffLine{
				Type:       diffAdded,
				NewLineNum: newIdx + 1,
				Text:       newLines[newIdx],
			})
			newIdx++
		}
	}

	var changeIndices []int
	for i, dl := range allLines {
		if dl.Type != diffContext {
			changeIndices = append(changeIndices, i)
		}
	}

	if len(changeIndices) == 0 {
		return nil
	}

	var hunks []diffHunk
	var currentHunk diffHunk
	hunkStart := -1

	for i, dl := range allLines {
		isNearChange := false
		for _, ci := range changeIndices {
			if i >= ci-ctxLines && i <= ci+ctxLines {
				isNearChange = true
				break
			}
		}

		if isNearChange {
			if hunkStart == -1 {
				hunkStart = i
			}
			currentHunk.Lines = append(currentHunk.Lines, dl)
		} else if hunkStart != -1 {
			hunks = append(hunks, currentHunk)
			currentHunk = diffHunk{}
			hunkStart = -1
		}
	}

	if hunkStart != -1 {
		hunks = append(hunks, currentHunk)
	}

	return hunks
}
