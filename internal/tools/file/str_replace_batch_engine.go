package file

type batchStringReplacementFailure struct {
	editIndex  int
	oldContent string
	oldStr     string
	failure    stringReplacementFailure
}

type batchStringReplacementPlan struct {
	newContent               string
	normalizedAttemptedEdits []int
}

type batchStringReplacementOutcome struct {
	plan    batchStringReplacementPlan
	failure *batchStringReplacementFailure
}

func buildBatchStringReplacementOutcome(oldContent string, edits []EditEntry) batchStringReplacementOutcome {
	content := oldContent
	normalizedAttemptedEdits := make([]int, 0, len(edits))

	for i, edit := range edits {
		execution := buildStringReplacementExecution(content, edit.OldStr, edit.NewStr)
		if execution.attemptedNormalized {
			normalizedAttemptedEdits = append(normalizedAttemptedEdits, i)
		}
		if execution.failure.hasFailure() {
			return batchStringReplacementOutcome{
				plan: batchStringReplacementPlan{
					newContent:               oldContent,
					normalizedAttemptedEdits: normalizedAttemptedEdits,
				},
				failure: &batchStringReplacementFailure{
					editIndex:  i,
					oldContent: content,
					oldStr:     edit.OldStr,
					failure:    execution.failure,
				},
			}
		}
		content = execution.plan.newContent
	}

	return batchStringReplacementOutcome{
		plan: batchStringReplacementPlan{
			newContent:               content,
			normalizedAttemptedEdits: normalizedAttemptedEdits,
		},
	}
}
