package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/susugadx/xelyon-cli/internal/app"
)

var rootExecutionArgs []string

type commandExitCodeError struct {
	message string
	code    int
}

func (e *commandExitCodeError) Error() string {
	return e.message
}

func (e *commandExitCodeError) ExitCode() int {
	return e.code
}

func commandErrorForExitPolicy(err error, policy app.HeadlessExitPolicy, code int) error {
	if err == nil {
		return nil
	}
	if policy != app.HeadlessExitPolicyCI {
		return err
	}
	return &commandExitCodeError{
		message: err.Error(),
		code:    code,
	}
}

func Execute() {
	rootExecutionArgs = append(rootExecutionArgs[:0], os.Args[1:]...)
	defer func() {
		rootExecutionArgs = nil
	}()

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(exitCodeForError(err))
	}
}

type exitCodeCarrier interface {
	ExitCode() int
}

func exitCodeForError(err error) int {
	var exitErr exitCodeCarrier
	if errors.As(err, &exitErr) {
		if code := exitErr.ExitCode(); code > 0 {
			return code
		}
	}
	return 1
}
