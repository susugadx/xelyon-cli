package taskstate

import "context"

// EditReadinessStatus は編集対象が直近の根拠に基づいているかの判定結果。
type EditReadinessStatus string

const (
	EditReadinessStatusOK      EditReadinessStatus = "ok"
	EditReadinessStatusWarning EditReadinessStatus = "warning"
	EditReadinessStatusUnknown EditReadinessStatus = "unknown"
)

// EditReadinessReason は warning 判定になった理由。
type EditReadinessReason string

const (
	EditReadinessReasonNoRecentRead         EditReadinessReason = "no_recent_read"
	EditReadinessReasonStaleEvidence        EditReadinessReason = "stale_evidence"
	EditReadinessReasonEvidenceRangeMissing EditReadinessReason = "evidence_range_missing"
	EditReadinessReasonPathNotInLedger      EditReadinessReason = "path_not_in_ledger"
	EditReadinessReasonRehydrateFailed      EditReadinessReason = "rehydrate_failed"
)

// EditReadinessTarget は編集前 readiness 判定の対象。
type EditReadinessTarget struct {
	Path       string
	ToolName   string
	ToolCallID string
}

// EditReadinessOptions は編集前 readiness 判定の追加検証を制御する。
type EditReadinessOptions struct {
	RehydrateEvidence bool
}

// EditReadinessObservation は編集前 readiness 判定の内部 observation。
type EditReadinessObservation struct {
	Path             string
	NormalizedPath   string
	ToolName         string
	ToolCallID       string
	Status           EditReadinessStatus
	Reasons          []EditReadinessReason
	EvidencePointers []EvidencePointer
	RehydrateResults []EvidenceRehydrateResult
}

// HasRecentEvidenceForPath は normalized ledger path に対応する evidence pointer があるかを返す。
func (s RuntimeTaskState) HasRecentEvidenceForPath(path string) bool {
	return len(s.EvidencePointersForPath(path)) > 0
}

// EvidencePointersForPath は normalized ledger path に対応する evidence pointer を防御コピーで返す。
func (s RuntimeTaskState) EvidencePointersForPath(path string) []EvidencePointer {
	if path == "" {
		return nil
	}
	var pointers []EvidencePointer
	for _, pointer := range s.Evidence.Pointers() {
		if pointer.Path == path {
			pointers = append(pointers, pointer)
		}
	}
	return pointers
}

// WasRecentlyTouched は normalized ledger path が直近の touched files にあるかを返す。
func (s RuntimeTaskState) WasRecentlyTouched(path string) bool {
	return s.TouchedFiles.contains(path)
}

// WasRecentlyChanged は normalized ledger path が直近の changed files にあるかを返す。
func (s RuntimeTaskState) WasRecentlyChanged(path string) bool {
	return s.ChangedFiles.contains(path)
}

// CheckEditReadiness は Store の workspace 情報と現在の RuntimeTaskState で編集前 readiness を判定する。
func (s *Store) CheckEditReadiness(ctx context.Context, target EditReadinessTarget, opts EditReadinessOptions) EditReadinessObservation {
	observation := EditReadinessObservation{
		Path:       target.Path,
		ToolName:   target.ToolName,
		ToolCallID: target.ToolCallID,
		Status:     EditReadinessStatusUnknown,
	}
	if s == nil {
		return observation
	}

	s.mu.Lock()
	repoRoot := s.repoRoot
	invocationCWD := s.invocationCWD
	state := s.state.clone()
	s.mu.Unlock()

	normalized, ok := normalizeLedgerPath(repoRoot, invocationCWD, target.Path)
	if !ok {
		return observation
	}
	observation.NormalizedPath = normalized

	pointers := state.EvidencePointersForPath(normalized)
	observation.EvidencePointers = cloneEvidencePointers(pointers)
	if len(pointers) == 0 {
		return classifyEditReadinessWithoutEvidence(observation, state, normalized)
	}

	reasons, results := evaluateEditEvidencePointers(ctx, pointers, EvidenceRehydrateOptions{
		RepoRoot:      repoRoot,
		InvocationCWD: invocationCWD,
	}, opts)
	observation.RehydrateResults = results
	if len(reasons) > 0 {
		observation.Status = EditReadinessStatusWarning
		observation.Reasons = reasons
		return observation
	}
	observation.Status = EditReadinessStatusOK
	return observation
}

func classifyEditReadinessWithoutEvidence(observation EditReadinessObservation, state RuntimeTaskState, normalizedPath string) EditReadinessObservation {
	changed := state.WasRecentlyChanged(normalizedPath)
	touched := state.WasRecentlyTouched(normalizedPath)
	observation.Status = EditReadinessStatusWarning
	switch {
	case changed && !touched:
		observation.Reasons = []EditReadinessReason{EditReadinessReasonNoRecentRead}
	case changed || touched:
		observation.Reasons = []EditReadinessReason{EditReadinessReasonEvidenceRangeMissing}
	default:
		observation.Reasons = []EditReadinessReason{EditReadinessReasonPathNotInLedger}
	}
	return observation
}

func evaluateEditEvidencePointers(ctx context.Context, pointers []EvidencePointer, opts EvidenceRehydrateOptions, readinessOpts EditReadinessOptions) ([]EditReadinessReason, []EvidenceRehydrateResult) {
	var reasons []EditReadinessReason
	var results []EvidenceRehydrateResult
	for _, pointer := range pointers {
		if pointer.Stale {
			reasons = appendUniqueEditReadinessReason(reasons, EditReadinessReasonStaleEvidence)
		}
		if !readinessOpts.RehydrateEvidence {
			continue
		}
		result, err := RehydrateEvidencePointer(ctx, pointer, opts)
		results = append(results, result)
		if err != nil {
			reasons = appendUniqueEditReadinessReason(reasons, EditReadinessReasonRehydrateFailed)
			continue
		}
		if result.Stale {
			reasons = appendUniqueEditReadinessReason(reasons, EditReadinessReasonStaleEvidence)
		}
	}
	return reasons, cloneEvidenceRehydrateResults(results)
}

func appendUniqueEditReadinessReason(reasons []EditReadinessReason, reason EditReadinessReason) []EditReadinessReason {
	for _, existing := range reasons {
		if existing == reason {
			return reasons
		}
	}
	return append(reasons, reason)
}
