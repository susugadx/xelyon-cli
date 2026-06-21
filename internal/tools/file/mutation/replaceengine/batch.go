package replaceengine

// BatchFailure は batch sequential planning の失敗データを表す。
type BatchFailure struct {
	editIndex  int
	oldContent string
	oldStr     string
	failure    StringFailure
}

// EditIndex は失敗した edits index を返す。
func (f BatchFailure) EditIndex() int {
	return f.editIndex
}

// OldContent は失敗時点の in-memory content を返す。
func (f BatchFailure) OldContent() string {
	return f.oldContent
}

// OldStr は失敗した edit の old_str を返す。
func (f BatchFailure) OldStr() string {
	return f.oldStr
}

// StringFailure は失敗した edit の文字列置換失敗データを返す。
func (f BatchFailure) StringFailure() StringFailure {
	return f.failure
}

// BatchPlan は batch sequential planning の成功結果を表す。
type BatchPlan struct {
	newContent               string
	normalizedAttemptedEdits []int
}

// NewContent は全 edit 適用後の全文を返す。
func (p BatchPlan) NewContent() string {
	return p.newContent
}

// NormalizedAttemptedEdits は normalized fallback を試した edits index を返す。
func (p BatchPlan) NormalizedAttemptedEdits() []int {
	return append([]int(nil), p.normalizedAttemptedEdits...)
}

// BatchOutcome は batch sequential planning の成功 plan または失敗データを持つ。
type BatchOutcome struct {
	plan    BatchPlan
	failure *BatchFailure
}

// Plan は batch planning の plan を返す。
func (o BatchOutcome) Plan() BatchPlan {
	return o.plan
}

// Failure は batch planning の失敗データを返す。
func (o BatchOutcome) Failure() (BatchFailure, bool) {
	if o.failure == nil {
		return BatchFailure{}, false
	}
	return *o.failure, true
}

// BuildBatchOutcome は edits を順番に in-memory 適用する pure plan を作る。
func BuildBatchOutcome(oldContent string, edits []Edit) BatchOutcome {
	content := oldContent
	normalizedAttemptedEdits := make([]int, 0, len(edits))

	for i, edit := range edits {
		execution := BuildStringExecution(content, edit.OldStr, edit.NewStr)
		if execution.AttemptedNormalized() {
			normalizedAttemptedEdits = append(normalizedAttemptedEdits, i)
		}
		if execution.Failure().HasFailure() {
			return BatchOutcome{
				plan: BatchPlan{
					newContent:               oldContent,
					normalizedAttemptedEdits: normalizedAttemptedEdits,
				},
				failure: &BatchFailure{
					editIndex:  i,
					oldContent: content,
					oldStr:     edit.OldStr,
					failure:    execution.Failure(),
				},
			}
		}
		content = execution.Plan().NewContent()
	}

	return BatchOutcome{
		plan: BatchPlan{
			newContent:               content,
			normalizedAttemptedEdits: normalizedAttemptedEdits,
		},
	}
}
