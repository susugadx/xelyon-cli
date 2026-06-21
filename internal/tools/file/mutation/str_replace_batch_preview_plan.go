package mutation

import "github.com/susugadx/xelyon-cli/internal/tools/file/mutation/replaceengine"

type batchStringReplacementPreviewPlan struct {
	outcome        replaceengine.BatchOutcome
	terminalResult fileMutationResult
}

func buildBatchStringReplacementPreviewPlan(path, oldContent string, edits []replaceengine.Edit) batchStringReplacementPreviewPlan {
	outcome := replaceengine.BuildBatchOutcome(oldContent, edits)
	if failure, ok := outcome.Failure(); ok {
		return batchStringReplacementPreviewPlan{
			outcome:        outcome,
			terminalResult: newErrorMutationResult(buildBatchStringReplacementFailure(path, failure)),
		}
	}
	if outcome.Plan().NewContent() == oldContent {
		return batchStringReplacementPreviewPlan{
			outcome:        outcome,
			terminalResult: newNoopMutationResult("No changes after applying all edits"),
		}
	}
	return batchStringReplacementPreviewPlan{outcome: outcome}
}
