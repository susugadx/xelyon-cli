package review

func findReviewPromptReductionItemForTest(runner *ReviewRunner, id string, phase ReviewModelPhase) *ReviewPromptReductionItem {
	if runner == nil || runner.promptReductionState == nil {
		return nil
	}
	for i := range runner.promptReductionState.Items {
		if runner.promptReductionState.Items[i].ID == id && runner.promptReductionState.Items[i].Phase == phase {
			return &runner.promptReductionState.Items[i]
		}
	}
	return nil
}
