package mutation

// EditEntry は batch edits の1エントリ
type EditEntry struct {
	OldStr string `json:"old_str"`
	NewStr string `json:"new_str"`
}

type lineRange struct {
	StartLine int
	EndLine   int
}

const (
	maxFailureCandidatesToShow = 2
	maxFailurePreviewLines     = 3
	failurePreviewLineWidth    = 72
)
