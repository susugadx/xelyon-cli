package review

import "fmt"

var blockedFindFlags = []string{
	"-delete",
	"-exec",
	"-execdir",
	"-ok",
	"-okdir",
	"-fprint",
	"-fprint0",
	"-fprintf",
	"-fls",
}

func validateAndPrepareFindHostReadOnlyArgs(args []string) (hostReadOnlyCommandState, error) {
	for _, arg := range args {
		if isBlockedFlagArg(arg, blockedFindFlags) {
			return hostReadOnlyCommandState{}, newHostReadOnlyBlockedError(fmt.Sprintf("blocked command: find argument %s is not allowed in host_readonly", arg))
		}
	}
	return hostReadOnlyCommandState{}, nil
}
