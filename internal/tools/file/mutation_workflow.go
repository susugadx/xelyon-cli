package file

import "github.com/susugadx/xelyon-cli/internal/tools/common"

type fileMutationWorkflow struct {
	toolName       string
	confirmMessage string
	preview        func() fileMutationResult
	confirm        mutationConfirmHandlers
	apply          func() (fileMutationResult, error)
}

func executeFileMutationWorkflow(ctx fileMutationContext, options common.ConfirmOptions, workflow fileMutationWorkflow) (fileMutationResult, error) {
	if workflow.preview != nil {
		result := workflow.preview()
		if result.IsTerminal() {
			return result, nil
		}
	}

	if result, ok := confirmFileMutation(ctx, options, workflow.toolName, workflow.confirmMessage, workflow.confirm); !ok {
		return result, nil
	}

	return workflow.apply()
}
