package probeplan

import (
	"fmt"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/review/domain"
)

func validateReviewPlannedProbe(index int, probe ReviewPlannedProbe, seenIDs map[string]struct{}, linkage reviewProbePlanProbeLinkageValidator) error {
	field := fmt.Sprintf("probes[%d]", index)

	id, err := validateReviewProbePlanID(field+".id", probe.ID)
	if err != nil {
		return err
	}
	if _, exists := seenIDs[id]; exists {
		return fmt.Errorf("%s.id duplicates %q", field, id)
	}
	seenIDs[id] = struct{}{}

	if err := validateReviewProbePlanPurpose(field+".purpose", probe.Purpose); err != nil {
		return err
	}
	if err := linkage.validateProbe(field, probe); err != nil {
		return err
	}
	if !isKnownReviewProbeMode(probe.Mode) {
		return fmt.Errorf("%s.mode must be known enum value: got %q", field, probe.Mode)
	}
	if err := validateReviewPlannedProbeFiles(field+".files", probe.Mode, probe.Files); err != nil {
		return err
	}
	if len(probe.Commands) == 0 {
		return fmt.Errorf("%s.commands must contain at least one command", field)
	}
	if len(probe.Commands) > MaxReviewProbePlanCommands {
		return fmt.Errorf("%s.commands must contain at most %d entries: got %d", field, MaxReviewProbePlanCommands, len(probe.Commands))
	}
	if probe.TimeoutSeconds < 0 {
		return fmt.Errorf("%s.timeout_seconds must be non-negative: got %d", field, probe.TimeoutSeconds)
	}
	if probe.TimeoutSeconds > MaxReviewProbePlanTimeoutSeconds {
		return fmt.Errorf("%s.timeout_seconds must be at most %d: got %d", field, MaxReviewProbePlanTimeoutSeconds, probe.TimeoutSeconds)
	}
	if probe.MaxOutputBytes < 0 {
		return fmt.Errorf("%s.max_output_bytes must be non-negative: got %d", field, probe.MaxOutputBytes)
	}
	if probe.MaxOutputBytes > MaxReviewProbePlanMaxOutputBytes {
		return fmt.Errorf("%s.max_output_bytes must be at most %d: got %d", field, MaxReviewProbePlanMaxOutputBytes, probe.MaxOutputBytes)
	}

	for i, command := range probe.Commands {
		if err := validateReviewPlannedProbeCommand(fmt.Sprintf("%s.commands[%d]", field, i), command); err != nil {
			return err
		}
	}
	return nil
}

func validateReviewPlannedProbeCommand(field string, command ReviewPlannedProbeCommand) error {
	if strings.TrimSpace(command.Command) == "" {
		return fmt.Errorf("%s.command must be non-empty", field)
	}
	if strings.TrimSpace(command.Command) != command.Command {
		return fmt.Errorf("%s.command must be canonical command without leading/trailing whitespace: got %q", field, command.Command)
	}
	if containsAnyWhitespace(command.Command) {
		return fmt.Errorf("%s.command must not include whitespace: got %q", field, command.Command)
	}
	if strings.ContainsRune(command.Command, '\x00') {
		return fmt.Errorf("%s.command must not contain null byte", field)
	}
	if strings.ContainsAny(command.Command, `/\`) {
		return fmt.Errorf("%s.command must be a command name without path separators: got %q", field, command.Command)
	}
	for i, arg := range command.Args {
		if strings.ContainsRune(arg, '\x00') {
			return fmt.Errorf("%s.args[%d] must not contain null byte", field, i)
		}
	}
	if command.WorkDir == "" || command.WorkDir == "." {
		return nil
	}
	return validateReviewProbePlanRelativePath(field+".work_dir", command.WorkDir)
}

func validateReviewPlannedProbeFiles(field string, mode domain.ReviewProbeMode, files []ReviewPlannedProbeFile) error {
	if mode == domain.ReviewProbeHostReadOnly && len(files) > 0 {
		return fmt.Errorf("%s must be empty when mode is %q", field, domain.ReviewProbeHostReadOnly)
	}
	if len(files) > MaxReviewProbePlanFiles {
		return fmt.Errorf("%s must contain at most %d entries: got %d", field, MaxReviewProbePlanFiles, len(files))
	}

	seenPaths := make(map[string]struct{}, len(files))
	totalContentBytes := 0
	for i, file := range files {
		fileField := fmt.Sprintf("%s[%d]", field, i)
		if err := validateReviewPlannedProbeFile(fileField, file); err != nil {
			return err
		}
		if _, exists := seenPaths[file.Path]; exists {
			return fmt.Errorf("%s.path duplicates %q", fileField, file.Path)
		}
		seenPaths[file.Path] = struct{}{}

		contentBytes := len([]byte(file.Content))
		if contentBytes > MaxReviewProbePlanFileContentBytes {
			return fmt.Errorf("%s.content must be at most %d bytes", fileField, MaxReviewProbePlanFileContentBytes)
		}
		totalContentBytes += contentBytes
		if totalContentBytes > MaxReviewProbePlanTotalFileContentBytes {
			return fmt.Errorf("%s content must total at most %d bytes", field, MaxReviewProbePlanTotalFileContentBytes)
		}
	}
	return nil
}

func validateReviewPlannedProbeFile(field string, file ReviewPlannedProbeFile) error {
	return validateReviewProbePlanRelativePath(field+".path", file.Path)
}

func validateReviewProbePlanRelativePath(field, candidate string) error {
	return validateReviewCanonicalRelativePath(field, candidate, reviewRelativePathValidationPolicy{
		pathKind:       "relative path",
		rejectNullByte: true,
	})
}
