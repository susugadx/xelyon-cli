package rawoutputs

import "fmt"

// Error は raw output artifact の structured reason 付き error。
type Error struct {
	Reason Reason
	Err    error
}

func (e Error) Error() string {
	if e.Err == nil {
		return string(e.Reason)
	}
	return fmt.Sprintf("%s: %v", e.Reason, e.Err)
}

func (e Error) Unwrap() error {
	return e.Err
}

func reasonError(reason Reason, format string, args ...interface{}) error {
	return Error{Reason: reason, Err: fmt.Errorf(format, args...)}
}

// ReasonOf は error に含まれる structured reason を返す。
func ReasonOf(err error) Reason {
	if err == nil {
		return ""
	}
	if typed, ok := err.(Error); ok {
		return typed.Reason
	}
	return ""
}
