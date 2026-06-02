package probe

import (
	"errors"
	"fmt"
)

func newBlockedCommandErrorf(format string, args ...any) error {
	return newHostReadOnlyBlockedError(fmt.Sprintf("blocked command: "+format, args...))
}

func newBlockedCommandArgError(command, arg string) error {
	return newBlockedCommandErrorf("%s argument %s is not allowed in host_readonly", command, arg)
}

func newBlockedCommandOptionError(command, option string) error {
	return newBlockedCommandErrorf("%s option %s is not allowed in host_readonly", command, option)
}

func newOutsideRepoCommandPathError(command, pathArg string) error {
	return newHostReadOnlyOutsideRepoPathError(
		fmt.Sprintf("blocked command: %s path %q is outside repository root", command, pathArg),
	)
}

func isOutsideRepoPathError(err error) bool {
	return errors.Is(err, ErrHostReadOnlyOutsideRepoPath)
}
