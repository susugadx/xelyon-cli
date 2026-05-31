package probe

import "errors"

var (
	// ErrHostReadOnlyBlocked は host_readonly policy で command 実行が拒否されたことを示す。
	ErrHostReadOnlyBlocked = errors.New("host_readonly blocked")
	// ErrHostReadOnlyOutsideRepoPath は repo root 外 path による拒否を示す。
	ErrHostReadOnlyOutsideRepoPath = errors.New("host_readonly outside repo path")
)

type hostReadOnlyPolicyErrorCode int

const (
	hostReadOnlyPolicyBlocked hostReadOnlyPolicyErrorCode = iota
	hostReadOnlyPolicyOutsideRepoPath
)

type hostReadOnlyPolicyError struct {
	message string
	code    hostReadOnlyPolicyErrorCode
}

func (e *hostReadOnlyPolicyError) Error() string {
	return e.message
}

func (e *hostReadOnlyPolicyError) Is(target error) bool {
	switch target {
	case ErrHostReadOnlyBlocked:
		// outside_repo_path は blocked の詳細分類として扱う。
		return e.code == hostReadOnlyPolicyBlocked || e.code == hostReadOnlyPolicyOutsideRepoPath
	case ErrHostReadOnlyOutsideRepoPath:
		return e.code == hostReadOnlyPolicyOutsideRepoPath
	default:
		return false
	}
}

func (e *hostReadOnlyPolicyError) Code() hostReadOnlyPolicyErrorCode {
	return e.code
}

func newHostReadOnlyBlockedError(message string) error {
	return &hostReadOnlyPolicyError{
		message: message,
		code:    hostReadOnlyPolicyBlocked,
	}
}

func newHostReadOnlyOutsideRepoPathError(message string) error {
	return &hostReadOnlyPolicyError{
		message: message,
		code:    hostReadOnlyPolicyOutsideRepoPath,
	}
}
