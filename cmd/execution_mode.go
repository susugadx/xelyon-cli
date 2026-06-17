package cmd

import "github.com/susugadx/xelyon-cli/internal/climode"

const (
	outputFormatText = climode.OutputFormatText
	outputFormatJSON = climode.OutputFormatJSON
)

type executionMode = climode.Mode

const (
	executionModeHeadless         = climode.ModeHeadless
	executionModeOnce             = climode.ModeOnce
	executionModeInteractive      = climode.ModeInteractive
	executionModeInteractiveImage = climode.ModeInteractiveImage
	executionModeOnceImage        = climode.ModeOnceImage
	executionModeResume           = climode.ModeResume
)

type executionModeRequest = climode.Request

func newExecutionModeRequest(args []string, resolvedOutputFormat string) executionModeRequest {
	return executionModeRequest{
		OutputFormat: resolvedOutputFormat,
		HasQuery:     len(args) > 0,
		HasImage:     imageFlag != "",
		Once:         once,
		Interactive:  interactive,
		Resume:       resume,
		Quiet:        quiet,
	}
}

func resolveOutputFormat(flagValue string, headless bool) (string, error) {
	return climode.ResolveOutputFormat(flagValue, headless)
}

func resolveExecutionMode(args []string, resolvedOutputFormat string) (executionMode, error) {
	return newExecutionModeRequest(args, resolvedOutputFormat).Resolve()
}

func executionModeIsInteractive(mode executionMode) bool {
	return climode.IsInteractive(mode)
}
