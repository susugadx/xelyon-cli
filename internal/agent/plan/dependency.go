package plan

import (
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
