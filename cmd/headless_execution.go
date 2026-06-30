package cmd

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/susugadx/xelyon-cli/internal/app"
)

type headlessProviderSetupRequiredError struct {
	provider string
	model    string
	message  string
}

func (e *headlessProviderSetupRequiredError) Error() string {
	return e.message
}

type headlessRuntimeSelectionConfigError struct {
	provider string
	model    string
	message  string
}

func (e *headlessRuntimeSelectionConfigError) Error() string {
	return e.message
}

func newHeadlessRuntimeSelectionConfigError(provider string, model string, err error) error {
	if err == nil {
		return nil
	}
	return &headlessRuntimeSelectionConfigError{
		provider: provider,
		model:    model,
		message:  err.Error(),
	}
}

func runHeadlessMode(cmd *cobra.Command, args []string, policy app.HeadlessExitPolicy) error {
	promptInput, err := resolveHeadlessPromptInput(cmd, args, imageFlag != "")
	if err != nil {
		promptInput.input = withHeadlessImageInputMetadataForCurrentProvider(promptInput.input, imageFlag)
		result := app.NewHeadlessUsageErrorResult("", "", err.Error()).WithInput(promptInput.input)
		return writeHeadlessResult(cmd, result, policy)
	}

	runtime, err := loadRuntimeSelectionForMode(cmd, executionModeHeadless)
	if err != nil {
		return writeHeadlessRuntimeSelectionError(cmd, err, promptInput.input, policy)
	}

	options := newHeadlessRunOptions()
	imageResolution := resolveHeadlessImageInput(runtime.provider, runtime.model, promptInput.input, imageFlag)
	promptInput.input = imageResolution.input
	if imageResolution.result != nil {
		return writeHeadlessResult(cmd, imageResolution.result, policy)
	}
	options.Image = imageResolution.image
	if strings.TrimSpace(promptInput.query) == "" && options.Image != nil {
		promptInput.query = app.DefaultImagePrompt
	}

	result := runHeadless(cmd.Context(), promptInput.query, runtime.model, runtime.provider, runtime.cfg, options)
	if result == nil {
		result = app.NewHeadlessConfigErrorResult(runtime.provider.Name(), runtime.model, "headless run returned nil result")
	}
	result.WithInput(promptInput.input)
	return writeHeadlessResult(cmd, result, policy)
}

func newHeadlessRunOptions() app.HeadlessRunOptions {
	return app.HeadlessRunOptions{
		FailOnToolError: failOnToolError,
		ReadOnly:        readOnly || dryRun,
	}
}

func writeHeadlessRuntimeSelectionError(cmd *cobra.Command, err error, input app.HeadlessInput, policy app.HeadlessExitPolicy) error {
	var setupErr *headlessProviderSetupRequiredError
	if errors.As(err, &setupErr) {
		return writeHeadlessProviderSetupRequiredError(cmd, setupErr, input, policy)
	}

	var configErr *headlessRuntimeSelectionConfigError
	if errors.As(err, &configErr) {
		input = withHeadlessImageInputMetadataForProviderName(input, imageFlag, configErr.provider)
		result := app.NewHeadlessConfigErrorResult(configErr.provider, configErr.model, configErr.message).WithInput(input)
		return writeHeadlessResult(cmd, result, policy)
	}

	input = withHeadlessImageInputMetadataForCurrentProvider(input, imageFlag)
	result := app.NewHeadlessConfigErrorResult("", "", err.Error()).WithInput(input)
	return writeHeadlessResult(cmd, result, policy)
}

func writeHeadlessProviderSetupRequiredError(cmd *cobra.Command, setupErr *headlessProviderSetupRequiredError, input app.HeadlessInput, policy app.HeadlessExitPolicy) error {
	imageSetupResolution := resolveHeadlessProviderSetupRequiredImageInput(setupErr.provider, setupErr.model, input, imageFlag)
	input = imageSetupResolution.input
	if imageSetupResolution.result != nil {
		return writeHeadlessResult(cmd, imageSetupResolution.result, policy)
	}
	result := app.NewHeadlessProviderSetupRequiredResult(setupErr.provider, setupErr.model, setupErr.message).WithInput(input)
	return writeHeadlessResult(cmd, result, policy)
}

func writeHeadlessUsageErrorResult(cmd *cobra.Command, args []string, err error, policy app.HeadlessExitPolicy) error {
	result := app.NewHeadlessUsageErrorResult("", "", err.Error()).
		WithInput(newHeadlessPreRunInputMetadata(cmd, args))
	return writeHeadlessResult(cmd, result, policy)
}

func writeHeadlessResult(cmd *cobra.Command, result *app.HeadlessResult, policy app.HeadlessExitPolicy) error {
	result, err := app.ApplyHeadlessExitPolicy(result, policy)
	if err != nil {
		return err
	}
	jsonOutput, err := result.ToJSON()
	if err != nil {
		return err
	}
	fmt.Println(jsonOutput)
	if result.Status == app.HeadlessStatusError {
		if cmd != nil {
			cmd.SilenceUsage = true
		}
		return &commandExitCodeError{
			message: "headless execution failed",
			code:    result.RecommendedExitCode,
		}
	}
	return nil
}
