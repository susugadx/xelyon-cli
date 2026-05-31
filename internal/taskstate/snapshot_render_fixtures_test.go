package taskstate

import "strconv"

func populatedSnapshotRenderState() RuntimeTaskState {
	return RuntimeTaskState{
		ChangedFiles: ChangedFiles{files: []fileFact{{path: "src/main.go"}}},
		TouchedFiles: TouchedFiles{files: []fileFact{{path: "docs/commands.md"}}},
		Evidence: Evidence{items: []evidenceFact{{
			path:       "internal/taskstate/taskstate.go",
			startLine:  20,
			endLine:    22,
			source:     "read_file",
			toolCallID: "call_x",
			fileHash:   "hash",
			stale:      true,
			excerpt:    "type RuntimeTaskState struct {\nChangedFiles ChangedFiles\n}",
		}}},
		RecommendedReads: RecommendedReads{items: []recommendedReadFact{{
			path:       "internal/agent/runtime.go",
			reason:     "owner boundary",
			source:     "read_file",
			toolCallID: "call_x",
		}}},
		LastFailedTests: LastFailedTests{results: []TestResult{
			NewTestResultWithExitCode("go test ./internal/agent", 1, "failed", "FAIL internal/agent\npanic trace"),
		}},
		LastPassedTests: LastPassedTests{results: []TestResult{
			NewTestResultWithExitCode("go test ./internal/taskstate", 0, "passed", ""),
		}},
	}
}

func snapshotRenderStateWithThreeFactsPerSection() RuntimeTaskState {
	return RuntimeTaskState{
		ChangedFiles: ChangedFiles{files: numberedFileFacts("src/changed_", 3)},
		TouchedFiles: TouchedFiles{files: numberedFileFacts("src/touched_", 3)},
		Evidence: Evidence{items: []evidenceFact{
			{path: "src/evidence_1.go", startLine: 1, endLine: 1, excerpt: "one"},
			{path: "src/evidence_2.go", startLine: 2, endLine: 2, excerpt: "two"},
			{path: "src/evidence_3.go", startLine: 3, endLine: 3, excerpt: "three"},
		}},
		RecommendedReads: RecommendedReads{items: numberedRecommendedReadFacts("src/read_", 3)},
		LastFailedTests:  LastFailedTests{results: numberedFailedTestResults(3)},
		LastPassedTests:  LastPassedTests{results: numberedPassedTestResults(3, "")},
	}
}

func snapshotRenderStateExceedingDefaultLimits() RuntimeTaskState {
	return RuntimeTaskState{
		ChangedFiles:     ChangedFiles{files: numberedFileFacts("src/changed_", 21)},
		TouchedFiles:     TouchedFiles{files: numberedFileFacts("src/touched_", 31)},
		Evidence:         Evidence{items: numberedEvidenceFacts("src/evidence_", 21)},
		RecommendedReads: RecommendedReads{items: numberedRecommendedReadFacts("src/read_", 11)},
		LastFailedTests:  LastFailedTests{results: numberedFailedTestResults(4)},
		LastPassedTests:  LastPassedTests{results: numberedPassedTestResults(4, "passed")},
	}
}

func numberedFileFacts(prefix string, count int) []fileFact {
	files := make([]fileFact, 0, count)
	for i := 1; i <= count; i++ {
		files = append(files, fileFact{path: prefix + strconv.Itoa(i) + ".go"})
	}
	return files
}

func numberedEvidenceFacts(prefix string, count int) []evidenceFact {
	items := make([]evidenceFact, 0, count)
	for i := 1; i <= count; i++ {
		items = append(items, evidenceFact{
			path:      prefix + strconv.Itoa(i) + ".go",
			startLine: i,
			endLine:   i,
			excerpt:   "evidence " + strconv.Itoa(i),
		})
	}
	return items
}

func numberedRecommendedReadFacts(prefix string, count int) []recommendedReadFact {
	items := make([]recommendedReadFact, 0, count)
	for i := 1; i <= count; i++ {
		items = append(items, recommendedReadFact{path: prefix + strconv.Itoa(i) + ".go"})
	}
	return items
}

func numberedFailedTestResults(count int) []TestResult {
	return numberedTestResults("failed", 1, count, "failed")
}

func numberedPassedTestResults(count int, excerptPrefix string) []TestResult {
	return numberedTestResults("passed", 0, count, excerptPrefix)
}

func numberedTestResults(status string, exitCode, count int, excerptPrefix string) []TestResult {
	results := make([]TestResult, 0, count)
	for i := 1; i <= count; i++ {
		command := "go test ./" + status + strconv.Itoa(i)
		excerpt := ""
		if excerptPrefix != "" {
			excerpt = excerptPrefix + " " + strconv.Itoa(i)
		}
		results = append(results, NewTestResultWithExitCode(command, exitCode, status, excerpt))
	}
	return results
}
