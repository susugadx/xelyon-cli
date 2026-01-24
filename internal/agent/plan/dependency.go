package plan

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/susugadx/xelyon-cli/internal/lsp"
)

// DependencyAnalyzer はプランステップ間の依存関係を解析
type DependencyAnalyzer struct {
	fileWriters map[string][]int // file -> 書き込むステップID
	fileReaders map[string][]int // file -> 読み取るステップID
	lspClient   *lsp.Client      // LSP連携（optional）
}

// Conflict は並列実行時の競合情報
type Conflict struct {
	StepIDs      []int    // 競合するステップID
	ConflictType string   // "write-write", "read-write", "write-read"
	Files        []string // 競合ファイル
	Message      string   // 詳細メッセージ
}

// DependencyResult は依存関係解析の結果
type DependencyResult struct {
	Steps     []PlanStep // 依存関係が補完されたステップ
	Conflicts []Conflict // 検出された競合
	Warnings  []string   // 警告メッセージ
}

// ツール分類（読み取り専用）
var readTools = map[string]bool{
	"read_file":       true,
	"search_code":     true,
	"search_file":     true,
	"list_dir":        true,
	"git_status":      true,
	"git_diff":        true,
	"git_log":         true,
	"lsp_references":  true,
	"lsp_definition":  true,
	"lsp_hover":       true,
	"lsp_diagnostics": true,
}

// ツール分類（書き込み）
var writeTools = map[string]bool{
	"write_file":    true,
	"str_replace":   true,
	"delete_file":   true,
	"append_file":   true,
	"prepend_file":  true,
	"insert_after":  true,
	"insert_before": true,
	"delete_lines":  true,
	"move_file":     true,
	"copy_file":     true,
	"create_dir":    true,
	"lsp_rename":    true, // 実際にはプレビューのみだが、将来の実装に備えてwrite扱い
}

// ファイルパス抽出用の正規表現パターン
var filePathPatterns = []*regexp.Regexp{
	// 引用符で囲まれたパス
	regexp.MustCompile(`["']([^"']+\.[a-zA-Z0-9]+)["']`),
	// パス風文字列（スペースなし）
	regexp.MustCompile(`\b((?:[\w.-]+/)*[\w.-]+\.[a-zA-Z0-9]{1,10})\b`),
	// 絶対パス
	regexp.MustCompile(`(/[^\s]+\.[a-zA-Z0-9]+)`),
}

// NewDependencyAnalyzer は DependencyAnalyzer を作成
func NewDependencyAnalyzer(lspClient *lsp.Client) *DependencyAnalyzer {
	return &DependencyAnalyzer{
		fileWriters: make(map[string][]int),
		fileReaders: make(map[string][]int),
		lspClient:   lspClient,
	}
}

// Analyze はステップ間の依存関係を解析
func (da *DependencyAnalyzer) Analyze(steps []PlanStep) *DependencyResult {
	result := &DependencyResult{
		Steps:     make([]PlanStep, len(steps)),
		Conflicts: []Conflict{},
		Warnings:  []string{},
	}

	// ステップをコピー
	copy(result.Steps, steps)

	// 各ステップからファイルを抽出
	for i := range result.Steps {
		readFiles, writeFiles := da.ExtractFilesFromStep(&result.Steps[i])
		result.Steps[i].ReadFiles = readFiles
		result.Steps[i].WriteFiles = writeFiles

		// ファイルアクセスマップを更新
		for _, f := range readFiles {
			da.fileReaders[f] = append(da.fileReaders[f], result.Steps[i].ID)
		}
		for _, f := range writeFiles {
			da.fileWriters[f] = append(da.fileWriters[f], result.Steps[i].ID)
		}
	}

	// 依存関係を推論
	result.Steps = da.InferDependencies(result.Steps)

	// LSPで強化（利用可能な場合）
	if da.lspClient != nil {
		result.Steps = da.EnhanceWithLSP(result.Steps)
	}

	return result
}

// ExtractFilesFromStep はステップからファイルパスを抽出
func (da *DependencyAnalyzer) ExtractFilesFromStep(step *PlanStep) (readFiles, writeFiles []string) {
	readSet := make(map[string]bool)
	writeSet := make(map[string]bool)

	// Description からファイルパスを抽出
	files := extractFilePaths(step.Description)

	// ツールの種類に基づいて分類
	hasReadTool := false
	hasWriteTool := false

	for _, tool := range step.Tools {
		if readTools[tool] {
			hasReadTool = true
		}
		if writeTools[tool] {
			hasWriteTool = true
		}
	}

	// 抽出されたファイルを分類
	for _, f := range files {
		// 書き込みツールがある場合、そのファイルは書き込み対象
		if hasWriteTool {
			writeSet[f] = true
		}
		// 読み取りツールがある場合、または書き込みツールがない場合は読み取り
		if hasReadTool || !hasWriteTool {
			readSet[f] = true
		}
	}

	// マップからスライスに変換
	for f := range readSet {
		readFiles = append(readFiles, f)
	}
	for f := range writeSet {
		writeFiles = append(writeFiles, f)
	}

	return readFiles, writeFiles
}

// extractFilePaths はテキストからファイルパスを抽出
func extractFilePaths(text string) []string {
	pathSet := make(map[string]bool)

	for _, pattern := range filePathPatterns {
		matches := pattern.FindAllStringSubmatch(text, -1)
		for _, match := range matches {
			if len(match) > 1 {
				path := match[1]
				// 明らかに非ファイルを除外
				if isValidFilePath(path) {
					pathSet[path] = true
				}
			}
		}
	}

	var paths []string
	for p := range pathSet {
		paths = append(paths, p)
	}
	return paths
}

// isValidFilePath はパスがファイルパスとして妥当かチェック
func isValidFilePath(path string) bool {
	// 空文字列
	if path == "" {
		return false
	}

	// 明らかにURLの場合
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return false
	}

	// 拡張子のない場合は除外（ディレクトリかもしれない）
	if !strings.Contains(path, ".") {
		return false
	}

	// コモンな非ファイル拡張子を除外
	invalidExts := []string{".com", ".org", ".net", ".io"}
	for _, ext := range invalidExts {
		if strings.HasSuffix(path, ext) && !strings.Contains(path, "/") {
			return false
		}
	}

	return true
}

// InferDependencies は依存関係を自動推論
// RAW (Read After Write): BがAの書き込みファイルを読む
// WAW (Write After Write): 両方が同じファイルに書く
// WAR (Write After Read): BがAの読み取りファイルに書く
func (da *DependencyAnalyzer) InferDependencies(steps []PlanStep) []PlanStep {
	for i := range steps {
		currentID := steps[i].ID
		existingDeps := make(map[int]bool)
		for _, d := range steps[i].DependsOn {
			existingDeps[d] = true
		}

		newDeps := make(map[int]bool)

		// RAW: 書き込み後読み取り
		for _, readFile := range steps[i].ReadFiles {
			writers := da.fileWriters[readFile]
			for _, writerID := range writers {
				// 自分より前のステップで書き込んでいる場合
				if writerID < currentID && !existingDeps[writerID] {
					newDeps[writerID] = true
				}
			}
		}

		// WAW: 書き込み後書き込み
		for _, writeFile := range steps[i].WriteFiles {
			writers := da.fileWriters[writeFile]
			for _, writerID := range writers {
				// 自分より前のステップで書き込んでいる場合
				if writerID < currentID && !existingDeps[writerID] {
					newDeps[writerID] = true
				}
			}
		}

		// WAR: 読み取り後書き込み
		for _, writeFile := range steps[i].WriteFiles {
			readers := da.fileReaders[writeFile]
			for _, readerID := range readers {
				// 自分より前のステップで読み取っている場合
				if readerID < currentID && !existingDeps[readerID] {
					newDeps[readerID] = true
				}
			}
		}

		// 新しい依存関係を追加
		for depID := range newDeps {
			steps[i].DependsOn = append(steps[i].DependsOn, depID)
		}
	}

	return steps
}

// DetectConflicts は並列実行時の競合を検出
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

// EnhanceWithLSP はLSPを使用して依存関係を強化
// WriteFilesに含まれるファイルのシンボル参照を取得し、
// 参照元ファイルが他のステップで使用されていれば依存関係を追加
func (da *DependencyAnalyzer) EnhanceWithLSP(steps []PlanStep) []PlanStep {
	if da.lspClient == nil {
		return steps
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Build: file -> stepIndex map (for quick lookup)
	fileToStepIdx := make(map[string][]int)
	for idx, step := range steps {
		for _, f := range step.ReadFiles {
			fileToStepIdx[f] = append(fileToStepIdx[f], idx)
		}
		for _, f := range step.WriteFiles {
			fileToStepIdx[f] = append(fileToStepIdx[f], idx)
		}
	}

	// For each step i, find references to its WriteFiles
	for i := range steps {
		for _, writeFile := range steps[i].WriteFiles {
			// Get files that reference writeFile via LSP
			refs, err := da.lspClient.FindReferences(ctx, writeFile, 1, 1, true)
			if err != nil {
				continue // LSP errors are not fatal
			}

			// For each reference file, check if later steps use it
			for _, ref := range refs {
				refFile := lsp.URIToFile(ref.URI)
				if refFile == writeFile {
					continue // skip self-reference
				}

				// Find later steps that use refFile
				for _, j := range fileToStepIdx[refFile] {
					// Compare by ID (not index) for safety
					if steps[j].ID > steps[i].ID && !containsInt(steps[j].DependsOn, steps[i].ID) {
						// Step j uses a file that references Step i's writeFile
						// => Step j depends on Step i
						steps[j].DependsOn = append(steps[j].DependsOn, steps[i].ID)
					}
				}
			}
		}
	}

	return steps
}

// containsInt checks if a slice contains an integer
func containsInt(slice []int, val int) bool {
	for _, v := range slice {
		if v == val {
			return true
		}
	}
	return false
}

// AnalyzeAndWarn は依存関係を解析し、競合があれば警告を返す
func (da *DependencyAnalyzer) AnalyzeAndWarn(steps []PlanStep, parallelStepIDs []int) ([]PlanStep, []string) {
	result := da.Analyze(steps)

	var warnings []string

	// 並列実行時の競合チェック
	if len(parallelStepIDs) > 1 {
		conflicts := da.DetectConflicts(parallelStepIDs, result.Steps)
		for _, c := range conflicts {
			warnings = append(warnings, FormatConflictWarning(c))
		}
	}

	return result.Steps, warnings
}

// FormatConflictWarning は競合情報を警告文字列に整形
func FormatConflictWarning(c Conflict) string {
	return "Conflict detected: " + c.Message +
		" (steps: " + formatIntSlice(c.StepIDs) +
		", files: " + strings.Join(c.Files, ", ") + ")"
}

// formatIntSlice は int スライスを文字列に整形
func formatIntSlice(ids []int) string {
	var strs []string
	for _, id := range ids {
		strs = append(strs, fmt.Sprintf("%d", id))
	}
	return strings.Join(strs, ", ")
}
