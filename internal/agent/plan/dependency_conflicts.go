package plan

import (
	"fmt"
	"strings"
)

// DetectConflicts は並列実行時の競合を検出する。
func (da *DependencyAnalyzer) DetectConflicts(parallelStepIDs []int, steps []PlanStep) []Conflict {
	var conflicts []Conflict

	// ステップIDからステップを取得
	stepMap := make(map[int]*PlanStep)
	for i := range steps {
		stepMap[steps[i].ID] = &steps[i]
	}

	// 並列実行するステップのファイルアクセスを収集
	parallelReads := make(map[string][]int)  // file -> stepIDs that read
	parallelWrites := make(map[string][]int) // file -> stepIDs that write

	for _, stepID := range parallelStepIDs {
		step := stepMap[stepID]
		if step == nil {
			continue
		}

		for _, f := range step.ReadFiles {
			parallelReads[f] = append(parallelReads[f], stepID)
		}
		for _, f := range step.WriteFiles {
			parallelWrites[f] = append(parallelWrites[f], stepID)
		}
	}

	// Write-Write 競合検出
	for file, writerIDs := range parallelWrites {
		if len(writerIDs) > 1 {
			conflicts = append(conflicts, Conflict{
				StepIDs:      writerIDs,
				ConflictType: "write-write",
				Files:        []string{file},
				Message:      "Multiple steps write to the same file",
			})
		}
	}

	// Read-Write 競合検出
	for file, writerIDs := range parallelWrites {
		if readerIDs, ok := parallelReads[file]; ok {
			// 書き込みと読み取りが同時に発生
			allIDs := make(map[int]bool)
			for _, id := range writerIDs {
				allIDs[id] = true
			}
			for _, id := range readerIDs {
				allIDs[id] = true
			}

			// 異なるステップで読み書きが発生している場合
			if len(allIDs) > 1 {
				var ids []int
				for id := range allIDs {
					ids = append(ids, id)
				}
				conflicts = append(conflicts, Conflict{
					StepIDs:      ids,
					ConflictType: "read-write",
					Files:        []string{file},
					Message:      "Concurrent read and write to the same file",
				})
			}
		}
	}

	return conflicts
}

// FormatConflictWarning は競合情報を警告文字列に整形する。
func FormatConflictWarning(c Conflict) string {
	return "Conflict detected: " + c.Message +
		" (steps: " + formatIntSlice(c.StepIDs) +
		", files: " + strings.Join(c.Files, ", ") + ")"
}

// formatIntSlice は int スライスを文字列に整形する。
func formatIntSlice(ids []int) string {
	var strs []string
	for _, id := range ids {
		strs = append(strs, fmt.Sprintf("%d", id))
	}
	return strings.Join(strs, ", ")
}
