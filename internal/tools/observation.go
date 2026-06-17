package tools

import "fmt"

// RuntimeObservation は tool が実際に surface した runtime fact を表す。
// 人間向け rendered output ではなく、tool owner が持つ構造化結果から生成する。
type RuntimeObservation struct {
	TouchedFiles     []ObservationPath
	Evidence         []ObservationEvidence
	RecommendedReads []ObservationRecommendedRead
}

// ObservationPath は表示 path と解決済み path を分けて保持する。
// ResolvedPath がある場合、ledger などの記録側はそれを優先できる。
type ObservationPath struct {
	Path         string
	ResolvedPath string
}

// ObservationEvidence は tool が提示した根拠箇所を表す。
// EndLine が 0 の場合、記録側は StartLine と同じ 1 行根拠として扱う。
type ObservationEvidence struct {
	Path         string
	ResolvedPath string
	StartLine    int
	EndLine      int
	Excerpt      string
}

// Normalize は ObservationEvidence contract に沿って line range を正規化する。
func (e ObservationEvidence) Normalize() ObservationEvidence {
	e.StartLine, e.EndLine = NormalizeObservationLineRange(e.StartLine, e.EndLine)
	return e
}

// NormalizeObservationLineRange は runtime observation の line-only evidence を正規化する。
func NormalizeObservationLineRange(startLine, endLine int) (int, int) {
	if endLine == 0 && startLine > 0 {
		endLine = startLine
	}
	return startLine, endLine
}

func (e ObservationEvidence) mergeKey() string {
	e = e.Normalize()
	return fmt.Sprintf("%s\x00%s\x00%d\x00%d\x00%s", e.Path, e.ResolvedPath, e.StartLine, e.EndLine, e.Excerpt)
}

// ObservationRecommendedRead は tool が追加で読むべき箇所として提示した file を表す。
type ObservationRecommendedRead struct {
	Path         string
	ResolvedPath string
	Reason       string
}

// Empty は記録すべき fact がないかを返す。
func (o RuntimeObservation) Empty() bool {
	return len(o.TouchedFiles) == 0 && len(o.Evidence) == 0 && len(o.RecommendedReads) == 0
}

// CloneRuntimeObservation は observation を呼び出し元で安全に保持できるよう複製する。
func CloneRuntimeObservation(item *RuntimeObservation) *RuntimeObservation {
	if item == nil || item.Empty() {
		return nil
	}
	return &RuntimeObservation{
		TouchedFiles:     append([]ObservationPath(nil), item.TouchedFiles...),
		Evidence:         append([]ObservationEvidence(nil), item.Evidence...),
		RecommendedReads: append([]ObservationRecommendedRead(nil), item.RecommendedReads...),
	}
}

// CloneRuntimeObservationGroups は pattern 別 observation map を呼び出し元で安全に保持できるよう複製する。
func CloneRuntimeObservationGroups(groups map[string]*RuntimeObservation) map[string]*RuntimeObservation {
	if len(groups) == 0 {
		return nil
	}
	cloned := make(map[string]*RuntimeObservation, len(groups))
	for key, observation := range groups {
		clonedObservation := CloneRuntimeObservation(observation)
		if clonedObservation == nil {
			continue
		}
		cloned[key] = clonedObservation
	}
	if len(cloned) == 0 {
		return nil
	}
	return cloned
}

// MergeRuntimeObservations は複数 tool observation を表示順を保って重複排除する。
func MergeRuntimeObservations(items ...*RuntimeObservation) *RuntimeObservation {
	merged := &RuntimeObservation{}
	seenTouched := make(map[string]bool)
	seenEvidence := make(map[string]bool)
	seenReads := make(map[string]bool)
	for _, item := range items {
		if item == nil || item.Empty() {
			continue
		}
		for _, touched := range item.TouchedFiles {
			key := touched.Path + "\x00" + touched.ResolvedPath
			if seenTouched[key] {
				continue
			}
			seenTouched[key] = true
			merged.TouchedFiles = append(merged.TouchedFiles, touched)
		}
		for _, evidence := range item.Evidence {
			evidence = evidence.Normalize()
			key := evidence.mergeKey()
			if seenEvidence[key] {
				continue
			}
			seenEvidence[key] = true
			merged.Evidence = append(merged.Evidence, evidence)
		}
		for _, read := range item.RecommendedReads {
			key := read.Path + "\x00" + read.ResolvedPath + "\x00" + read.Reason
			if seenReads[key] {
				continue
			}
			seenReads[key] = true
			merged.RecommendedReads = append(merged.RecommendedReads, read)
		}
	}
	if merged.Empty() {
		return nil
	}
	return merged
}

// ToolRunResult は optional structured-result tool が返す実行結果。
type ToolRunResult struct {
	Output      string
	Change      *FileChange
	Observation *RuntimeObservation
	// ObservationGroups は batched result を元の tool call へ分配するための任意グループ。
	ObservationGroups map[string]*RuntimeObservation
}

// StructuredResultTool は rendered output と一緒に runtime observation を返せる tool。
// 通常の Tool.Run contract は維持し、必要な tool だけがこの interface を追加実装する。
type StructuredResultTool interface {
	RunResult(execCtx ExecutionContext, args map[string]string) (ToolRunResult, error)
}
