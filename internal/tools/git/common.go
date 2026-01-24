package git

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/tools/common"
)

// Colors from common package
var (
	cyan   = common.Cyan
	yellow = common.Yellow
)

// ExecuteGitCommand executes a git command and returns the result
func ExecuteGitCommand(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

// FormatGitError formats a git command error
func FormatGitError(err error, output string) string {
	if output != "" {
		return fmt.Sprintf("Error: %v\n%s", err, output)
	}
	return fmt.Sprintf("Error: %v", err)
}

// FormatGitSuccess formats a git success message
func FormatGitSuccess(action, target string) string {
	return fmt.Sprintf("✅ %s: %s", action, target)
}

// CheckUncommittedChanges checks for uncommitted changes
// status: git status --porcelain output
// hasChanges: whether there are changes
func CheckUncommittedChanges() (status string, hasChanges bool, err error) {
	output, err := ExecuteGitCommand("status", "--porcelain")
	if err != nil {
		return "", false, err
	}
	return output, len(strings.TrimSpace(output)) > 0, nil
}

// TruncateOutput truncates output to specified number of lines
func TruncateOutput(output string, maxLines int) string {
	lines := strings.Split(output, "\n")
	if len(lines) <= maxLines {
		return output
	}
	result := strings.Join(lines[:maxLines], "\n")
	return fmt.Sprintf("%s\n... (%d more lines)", result, len(lines)-maxLines)
}

// DisplayGitConfirmHeader displays the confirm UI header
func DisplayGitConfirmHeader(title, subtitle, target string) {
	cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	cyan.Printf("%s\n", title)
	cyan.Printf("%s: %s\n", subtitle, target)
	cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
}

// DisplayUncommittedChangesWarning displays a warning for uncommitted changes
func DisplayUncommittedChangesWarning(status string) {
	yellow.Println("⚠️  Warning: You have uncommitted changes / 警告: 未コミットの変更があります")
	fmt.Println("\nUncommitted changes / 未コミットの変更:")
	fmt.Println(status)
}
