package review

import "fmt"

func validateFindHostReadOnlyArgs(args []string) error {
	for _, arg := range args {
		if isBlockedFlagArg(arg, []string{"-delete", "-exec", "-execdir", "-ok", "-okdir", "-fprint", "-fprint0", "-fprintf", "-fls"}) {
			return fmt.Errorf("blocked command: find argument %s is not allowed in host_readonly", arg)
		}
	}
	return nil
}
