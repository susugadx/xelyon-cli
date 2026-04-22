package file

type batchStringReplacementPreviewPlan struct {
	outcome        batchStringReplacementOutcome
	terminalResult fileMutationResult
}

func buildBatchStringReplacementPreviewPlan(path, oldContent string, edits []EditEntry) batchStringReplacementPreviewPlan {
	outcome := buildBatchStringReplacementOutcome(oldContent, edits)
	if outcome.failure != nil {
		return batchStringReplacementPreviewPlan{
			outcome:        outcome,
			terminalResult: newErrorMutationResult(buildBatchStringReplacementFailure(path, *outcome.failure)),
		}
	}
	if outcome.plan.newContent == oldContent {
		return batchStringReplacementPreviewPlan{
			outcome:        outcome,
			terminalResult: newNoopMutationResult("No changes after applying all edits"),
		}
	}
	return batchStringReplacementPreviewPlan{outcome: outcome}
}
