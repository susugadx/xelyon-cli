package review

import "time"

// BuildReviewProbeRequestsFromPlan は検証済み plan DTO を ProbeRunner 用 runtime request へ変換する。
func BuildReviewProbeRequestsFromPlan(plan ReviewProbePlan) ([]ReviewProbeRequest, error) {
	// DecodeReviewProbePlanJSON と同じ schema validation を direct caller 向けにも通す。
	// 以降は semantic validation を増やさず、runtime request への機械的な変換だけを行う。
	if err := ValidateReviewProbePlan(plan); err != nil {
		return nil, err
	}

	requests := make([]ReviewProbeRequest, 0, len(plan.Probes))
	for _, probe := range plan.Probes {
		requests = append(requests, ReviewProbeRequest{
			ID:             probe.ID,
			Purpose:        probe.Purpose,
			Mode:           probe.Mode,
			Commands:       buildReviewProbeRequestCommands(probe.Commands),
			Files:          buildReviewProbeRequestFiles(probe.Files),
			Timeout:        buildReviewProbeRequestTimeout(probe.TimeoutSeconds),
			MaxOutputBytes: probe.MaxOutputBytes,
		})
	}
	return requests, nil
}

func buildReviewProbeRequestTimeout(seconds int) time.Duration {
	if seconds <= 0 {
		return 0
	}
	return time.Duration(seconds) * time.Second
}

func buildReviewProbeRequestCommands(commands []ReviewPlannedProbeCommand) []ReviewProbeCommand {
	requestCommands := make([]ReviewProbeCommand, 0, len(commands))
	for _, command := range commands {
		requestCommands = append(requestCommands, ReviewProbeCommand{
			Command: command.Command,
			Args:    append([]string(nil), command.Args...),
			WorkDir: normalizeReviewProbePlanWorkDir(command.WorkDir),
		})
	}
	return requestCommands
}

func buildReviewProbeRequestFiles(files []ReviewPlannedProbeFile) []ReviewProbeFile {
	requestFiles := make([]ReviewProbeFile, 0, len(files))
	for _, file := range files {
		requestFiles = append(requestFiles, ReviewProbeFile(file))
	}
	return requestFiles
}

func normalizeReviewProbePlanWorkDir(workDir string) string {
	if workDir == "." {
		return ""
	}
	return workDir
}
