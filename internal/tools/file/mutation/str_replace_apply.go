package mutation

import (
	"fmt"
	"os"
)

func applyStringReplaceMutation(ctx fileMutationContext, newContent, successStdoutMessage, successResultMessage string) (fileMutationResult, error) {
	syntaxWarning := validateGoSyntaxForReplace(ctx.absPath, []byte(newContent))
	if syntaxWarning != "" && !ctx.out.SuppressStdout() {
		ctx.out.Yellow.Printf("%s\n", syntaxWarning)
	}
	if err := os.WriteFile(ctx.absPath, []byte(newContent), 0644); err != nil {
		return newErrorMutationResult(fmt.Sprintf("Error writing file: %v", err)), nil
	}

	if successStdoutMessage != "" {
		ctx.out.Green.Printf("%s\n", successStdoutMessage)
	}
	return newAppliedMutationResult(appendSyntaxWarning(successResultMessage, syntaxWarning)), nil
}
