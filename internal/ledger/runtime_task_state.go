package ledger

// RuntimeTaskState はセッション中だけ保持するタスク台帳のスナップショット。
// prompt/history/provider request への接続はこの package の外側で明示的に行う。
type RuntimeTaskState struct {
	ChangedFiles     ChangedFiles
	TouchedFiles     TouchedFiles
	Evidence         Evidence
	RecommendedReads RecommendedReads
	LastFailedTests  LastFailedTests
	LastPassedTests  LastPassedTests
}

// IsEmpty は model-facing snapshot に載せる runtime task fact が未記録かを返す。
func (s RuntimeTaskState) IsEmpty() bool {
	return s.ChangedFiles.Len() == 0 &&
		s.TouchedFiles.Len() == 0 &&
		s.Evidence.Len() == 0 &&
		s.RecommendedReads.Len() == 0 &&
		s.LastFailedTests.Len() == 0 &&
		s.LastPassedTests.Len() == 0
}

func (s RuntimeTaskState) clone() RuntimeTaskState {
	return RuntimeTaskState{
		ChangedFiles:     s.ChangedFiles.clone(),
		TouchedFiles:     s.TouchedFiles.clone(),
		Evidence:         s.Evidence.clone(),
		RecommendedReads: s.RecommendedReads.clone(),
		LastFailedTests:  s.LastFailedTests.clone(),
		LastPassedTests:  s.LastPassedTests.clone(),
	}
}
