package review

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
	if target == ErrHostReadOnlyBlocked {
		return true
	}
	return target == ErrHostReadOnlyOutsideRepoPath && e.code == hostReadOnlyPolicyOutsideRepoPath
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
