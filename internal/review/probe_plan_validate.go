package review

import (
	"fmt"
	"path"
	"strings"
)

// ValidateReviewProbePlan は LLM probe plan schema v1 の構造契約を検証する。
func ValidateReviewProbePlan(plan ReviewProbePlan) error {
	if plan.SchemaVersion != ReviewProbePlanSchemaVersionV1 {
		return fmt.Errorf("schema_version must be %q: got %q", ReviewProbePlanSchemaVersionV1, plan.SchemaVersion)
	}
	if plan.TargetKind != TargetCurrentChanges {
		return fmt.Errorf("target_kind must be %q: got %q", TargetCurrentChanges, plan.TargetKind)
	}
	if len(plan.Probes) > MaxReviewProbePlanProbes {
		return fmt.Errorf("probes must contain at most %d entries: got %d", MaxReviewProbePlanProbes, len(plan.Probes))
	}
	if len(plan.Probes) == 0 {
		if strings.TrimSpace(plan.NoProbeReason) == "" {
			return fmt.Errorf("no_probe_reason must be non-empty when probes is empty")
		}
		return nil
	}
	if plan.NoProbeReason != "" {
		return fmt.Errorf("no_probe_reason must be empty when probes is non-empty")
	}

	seenIDs := make(map[string]struct{}, len(plan.Probes))
	for i, probe := range plan.Probes {
		if err := validateReviewPlannedProbe(i, probe, seenIDs); err != nil {
			return err
		}
	}
	return nil
}

func validateReviewPlannedProbe(index int, probe ReviewPlannedProbe, seenIDs map[string]struct{}) error {
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

func validateReviewProbePlanID(field, candidate string) (string, error) {
	if strings.TrimSpace(candidate) == "" {
		return "", fmt.Errorf("%s must be non-empty", field)
	}
	if strings.TrimSpace(candidate) != candidate {
		return "", fmt.Errorf("%s must be canonical ID without leading/trailing whitespace: got %q", field, candidate)
	}
	if containsAnyWhitespace(candidate) {
		return "", fmt.Errorf("%s must not include whitespace: got %q", field, candidate)
	}
	return candidate, nil
}

func validateReviewProbePlanPurpose(field, candidate string) error {
	if strings.TrimSpace(candidate) == "" {
		return fmt.Errorf("%s must be non-empty", field)
	}
	if strings.TrimSpace(candidate) != candidate {
		return fmt.Errorf("%s must be canonical purpose without leading/trailing whitespace: got %q", field, candidate)
	}
	if len([]byte(candidate)) > MaxReviewProbePlanPurposeBytes {
		return fmt.Errorf("%s must be at most %d bytes", field, MaxReviewProbePlanPurposeBytes)
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

func validateReviewPlannedProbeFiles(field string, mode ReviewProbeMode, files []ReviewPlannedProbeFile) error {
	if mode == ReviewProbeHostReadOnly && len(files) > 0 {
		return fmt.Errorf("%s must be empty when mode is %q", field, ReviewProbeHostReadOnly)
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
	if strings.TrimSpace(candidate) == "" {
		return fmt.Errorf("%s must be non-empty", field)
	}
	if strings.TrimSpace(candidate) != candidate {
		return fmt.Errorf("%s must be canonical relative path without leading/trailing whitespace: got %q", field, candidate)
	}
	if strings.ContainsRune(candidate, '\x00') {
		return fmt.Errorf("%s must not contain null byte", field)
	}
	if isReviewAbsolutePathLike(candidate) {
		return fmt.Errorf("%s must be relative path: got absolute path %q", field, candidate)
	}
	if strings.Contains(candidate, `\`) {
		return fmt.Errorf("%s must use '/' separators: got %q", field, candidate)
	}
	for _, segment := range strings.Split(candidate, "/") {
		if segment == "." || segment == ".." {
			return fmt.Errorf("%s must not contain %q segment: got %q", field, segment, candidate)
		}
	}

	cleaned := path.Clean(candidate)
	if cleaned != candidate {
		return fmt.Errorf("%s must be canonical relative path: got %q (canonical: %q)", field, candidate, cleaned)
	}
	return nil
}
