package review

// TargetKind は review 対象の種類を表す。
type TargetKind string

const (
	// TargetUncommitted は未コミット変更を review 対象にする。
	TargetUncommitted TargetKind = "uncommitted"
)

// ReviewRequest は review runner に渡す入力契約を表す。
type ReviewRequest struct {
	TargetKind         TargetKind
	CustomInstructions string
}

// NewUncommittedRequest は未コミット変更向けの ReviewRequest を構築する。
func NewUncommittedRequest(customInstructions string) ReviewRequest {
	return ReviewRequest{
		TargetKind:         TargetUncommitted,
		CustomInstructions: customInstructions,
	}
}
