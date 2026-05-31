package taskstate

import "strings"

// Recorder は RuntimeTaskState の mutation を集約する。
type Recorder struct {
	store *Store
}

// RecordToolObservation はツール観測から台帳 fact を更新する。
func (r *Recorder) RecordToolObservation(observation ToolObservation) {
	if r == nil || r.store == nil {
		return
	}
	invocationCWD := r.store.invocationCWD
	if strings.TrimSpace(observation.InvocationCWD) != "" {
		invocationCWD = observation.InvocationCWD
	}
	facts := collectToolObservationFacts(r.store.repoRoot, invocationCWD, observation)
	r.mutate(func(state *RuntimeTaskState) {
		for _, path := range facts.changedFiles {
			state.ChangedFiles.recordPath(path)
		}
		for _, path := range facts.touchedFiles {
			state.TouchedFiles.recordPath(path)
		}
		for _, fact := range facts.evidence {
			state.Evidence.record(fact)
		}
		for _, fact := range facts.recommendedReads {
			state.RecommendedReads.recordFact(fact)
		}
		for _, result := range facts.failedTests {
			state.LastFailedTests.append(result)
		}
		for _, result := range facts.passedTests {
			state.LastPassedTests.append(result)
		}
	})
}

// RecordChangedFile は ChangedFiles へファイルパスを記録する。
func (r *Recorder) RecordChangedFile(path string) {
	normalized, ok := normalizeLedgerPath(r.repoRoot(), r.invocationCWD(), path)
	if !ok {
		return
	}
	r.mutate(func(state *RuntimeTaskState) {
		state.ChangedFiles.recordPath(normalized)
	})
}

// RecordTouchedFile は TouchedFiles へファイルパスを記録する。
func (r *Recorder) RecordTouchedFile(path string) {
	normalized, ok := normalizeLedgerPath(r.repoRoot(), r.invocationCWD(), path)
	if !ok {
		return
	}
	r.mutate(func(state *RuntimeTaskState) {
		state.TouchedFiles.recordPath(normalized)
	})
}

// RecordEvidence は Evidence へ根拠を記録する。
func (r *Recorder) RecordEvidence(text, source string) {
	excerpt := strings.TrimSpace(truncateBytes(text, maxFactExcerptBytes))
	if excerpt == "" {
		return
	}
	r.mutate(func(state *RuntimeTaskState) {
		state.Evidence.record(evidenceFact{excerpt: excerpt, source: source})
	})
}

// RecordRecommendedRead は RecommendedReads へファイルパスと理由を記録する。
func (r *Recorder) RecordRecommendedRead(path, reason string) {
	normalized, ok := normalizeLedgerPath(r.repoRoot(), r.invocationCWD(), path)
	if !ok {
		return
	}
	r.mutate(func(state *RuntimeTaskState) {
		state.RecommendedReads.record(normalized, strings.TrimSpace(reason))
	})
}

// RecordTestObservation はテスト実行観測から LastPassedTests / LastFailedTests を更新する。
func (r *Recorder) RecordTestObservation(observation TestObservation) {
	result, ok := testResultFromObservation(observation)
	if !ok {
		return
	}
	r.mutate(func(state *RuntimeTaskState) {
		switch result.status {
		case "passed":
			state.LastPassedTests.append(result)
		case "failed":
			state.LastFailedTests.append(result)
		}
	})
}

// SetLastFailedTests は LastFailedTests を指定値で置き換える。
func (r *Recorder) SetLastFailedTests(results []TestResult) {
	r.mutate(func(state *RuntimeTaskState) {
		state.LastFailedTests.replace(results)
	})
}

// SetLastPassedTests は LastPassedTests を指定値で置き換える。
func (r *Recorder) SetLastPassedTests(results []TestResult) {
	r.mutate(func(state *RuntimeTaskState) {
		state.LastPassedTests.replace(results)
	})
}

func (r *Recorder) mutate(fn func(*RuntimeTaskState)) {
	if r == nil || r.store == nil || fn == nil {
		return
	}
	r.store.mu.Lock()
	defer r.store.mu.Unlock()
	fn(&r.store.state)
}

func (r *Recorder) repoRoot() string {
	if r == nil || r.store == nil {
		return ""
	}
	return r.store.repoRoot
}

func (r *Recorder) invocationCWD() string {
	if r == nil || r.store == nil {
		return ""
	}
	return r.store.invocationCWD
}
