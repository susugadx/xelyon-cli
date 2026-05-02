package review

// TargetKind は review 対象の種類を表す。
type TargetKind string

const (
	// TargetCurrentChanges は現在の作業ツリー差分を review 対象にする。
	TargetCurrentChanges TargetKind = "current_changes"
)

// ReviewRequest は review runner に渡す入力契約を表す。
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
