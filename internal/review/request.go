package review

// ReviewRequest は次フェーズの ReviewRunner に渡す入力契約を表す。
// TUI はこの request 生成までを担当し、evidence 収集以降の実行は runner が owner する。
type ReviewRequest struct {
	TargetKind         TargetKind
	CustomInstructions string
}

// NewCurrentChangesRequest は current_changes 向けの ReviewRequest を構築する。
func NewCurrentChangesRequest(customInstructions string) ReviewRequest {
	return ReviewRequest{
		TargetKind:         TargetCurrentChanges,
		CustomInstructions: customInstructions,
	}
}
