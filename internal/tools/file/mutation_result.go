package file

type fileMutationStatus int

const (
	fileMutationStatusUnknown fileMutationStatus = iota
	fileMutationStatusApplied
	fileMutationStatusNoop
	fileMutationStatusCancelled
	fileMutationStatusComment
	fileMutationStatusError
)

type fileMutationResult struct {
	status  fileMutationStatus
	message string
}

func (r fileMutationResult) IsTerminal() bool {
	return r.status != fileMutationStatusUnknown || r.message != ""
}

func newAppliedMutationResult(message string) fileMutationResult {
	return fileMutationResult{status: fileMutationStatusApplied, message: message}
}

func newNoopMutationResult(message string) fileMutationResult {
	return fileMutationResult{status: fileMutationStatusNoop, message: message}
}

func newCancelledMutationResult(message string) fileMutationResult {
	return fileMutationResult{status: fileMutationStatusCancelled, message: message}
}

func newCommentMutationResult(message string) fileMutationResult {
	return fileMutationResult{status: fileMutationStatusComment, message: message}
}

func newErrorMutationResult(message string) fileMutationResult {
	return fileMutationResult{status: fileMutationStatusError, message: message}
}

func (r fileMutationResult) ShouldRecordChange() bool {
	return r.status == fileMutationStatusApplied
}
