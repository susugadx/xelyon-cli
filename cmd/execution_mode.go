package cmd

import (
	"fmt"
	"strings"
)

const (
	outputFormatText = "text"
	outputFormatJSON = "json"
)

type executionMode string

const (
	executionModeHeadless         executionMode = "headless"
	executionModeOnce             executionMode = "once"
	executionModeInteractive      executionMode = "interactive"
	executionModeInteractiveImage executionMode = "interactive_image"
	executionModeOnceImage        executionMode = "once_image"
	executionModeResume           executionMode = "resume"
)

type executionModeRequest struct {
	outputFormat string
	hasQuery     bool
	hasImage     bool
	once         bool
	interactive  bool
	resume       bool
	quiet        bool
}

func newExecutionModeRequest(args []string, resolvedOutputFormat string) executionModeRequest {
	return executionModeRequest{
		outputFormat: resolvedOutputFormat,
		hasQuery:     len(args) > 0,
		hasImage:     imageFlag != "",
		once:         once,
		interactive:  interactive,
		resume:       resume,
		quiet:        quiet,
	}
}

func (r executionModeRequest) implicitTextOnce() bool {
	return r.outputFormat == outputFormatText && r.hasQuery && !r.hasImage && !r.interactive && !r.resume
}

func (r executionModeRequest) implicitImageOnce() bool {
	return r.outputFormat == outputFormatText && r.hasImage && !r.interactive && !r.resume
}

func (r executionModeRequest) explicitImageOnce() bool {
	return r.hasImage && r.once
}

func (r executionModeRequest) effectiveOnce() bool {
	return r.once || r.implicitTextOnce() || r.implicitImageOnce()
}

func (r executionModeRequest) validate() error {
	if r.interactive && r.once {
		return fmt.Errorf("--interactive cannot be used with --once")
	}

	if r.resume && r.hasImage {
		return fmt.Errorf("--resume cannot be used with --image")
	}

	if r.resume && r.hasQuery && r.outputFormat == outputFormatText {
		return fmt.Errorf("--resume cannot be used with query arguments")
	}

	if r.hasImage && r.outputFormat == outputFormatJSON {
		return fmt.Errorf("--image cannot be used with --headless or --output-format json")
	}

	if r.quiet && !r.effectiveOnce() {
		return fmt.Errorf("--quiet can only be used with one-shot execution")
	}

	if r.once {
		if r.resume {
			return fmt.Errorf("--once cannot be used with --resume")
		}
		if r.outputFormat == outputFormatJSON {
			return fmt.Errorf("--once cannot be used with --headless or --output-format json")
		}
		if !r.hasQuery && !r.hasImage {
			return fmt.Errorf("query argument is required when using --once")
		}
	}

	return nil
}

func (r executionModeRequest) resolve() (executionMode, error) {
	if err := r.validate(); err != nil {
		return "", err
	}

	switch {
	case r.outputFormat == outputFormatJSON:
		return executionModeHeadless, nil
	case r.hasImage && r.interactive:
		return executionModeInteractiveImage, nil
	case r.explicitImageOnce() || r.implicitImageOnce():
		return executionModeOnceImage, nil
	case r.once:
		return executionModeOnce, nil
	case r.resume && !r.hasQuery:
		return executionModeResume, nil
	case !r.hasQuery:
		return executionModeInteractive, nil
	case r.implicitTextOnce():
		return executionModeOnce, nil
	default:
		return executionModeInteractive, nil
	}
}

func resolveOutputFormat(flagValue string, headless bool) (string, error) {
	format := strings.ToLower(strings.TrimSpace(flagValue))
	if format == "" {
		format = outputFormatText
	}

	switch format {
	case outputFormatText, outputFormatJSON:
	case "":
		format = outputFormatText
	default:
		return "", fmt.Errorf("invalid --output-format %q (expected text or json)", flagValue)
	}

	if headless {
		return outputFormatJSON, nil
	}

	return format, nil
}

func resolveExecutionMode(args []string, resolvedOutputFormat string) (executionMode, error) {
	return newExecutionModeRequest(args, resolvedOutputFormat).resolve()
}
