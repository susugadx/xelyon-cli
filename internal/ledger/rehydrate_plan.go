package ledger

const (
	defaultRehydratePlanMaxItems        = 8
	defaultRehydratePlanMaxLinesPerItem = 80
	defaultRehydratePlanMaxTotalLines   = 240

	RehydratePlanReasonStaleEvidenceRequiresRefresh = "stale_evidence_requires_refresh"
	RehydratePlanReasonEditTargetMissingEvidence    = "edit_target_missing_recent_evidence"
	RehydratePlanReasonOmittedProviderHistory       = "omitted_provider_history_evidence"
)

// RehydratePlan は古い provider history evidence を現在ファイルから再読込する dry-run 計画。
type RehydratePlan struct {
	Items []RehydratePlanItem
}

// RehydratePlanItem は再読込対象の path/range と、その計画理由を表す。
type RehydratePlanItem struct {
	Path       string
	StartLine  int
	EndLine    int
	Source     string
	Reason     string
	ToolCallID string
	Stale      bool
}

// RehydratePlanOptions は RehydratePlan の入力と上限を表す。
type RehydratePlanOptions struct {
	TargetPaths               []string
	OldEvidencePointers       []EvidencePointer
	EditReadinessObservations []EditReadinessObservation
	MaxItems                  int
	MaxLinesPerItem           int
	MaxTotalLines             int
}

type rehydratePlanTarget struct {
	explicit     bool
	editTarget   bool
	staleWarning bool
}

type rehydratePlanTargetSet struct {
	order   []string
	targets map[string]rehydratePlanTarget
}

type rehydratePlanItemKey struct {
	path      string
	startLine int
	endLine   int
}

type rehydratePlanEvidencePointerKey struct {
	path       string
	startLine  int
	endLine    int
	source     string
	toolCallID string
}

// BuildRehydratePlan は provider history で省略された古い evidence の再読込計画を作る。
// この関数は dry-run 専用で、ファイル読込、provider input 注入、history 追記は行わない。
func BuildRehydratePlan(state RuntimeTaskState, workspace EvidenceRehydrateOptions, opts RehydratePlanOptions) RehydratePlan {
	normalized := normalizeRehydratePlanOptions(opts)
	rehydrateWorkspace, _, err := newEvidenceRehydrateWorkspace(workspace)
	if err != nil {
		return RehydratePlan{}
	}

	targets := buildRehydratePlanTargets(state, rehydrateWorkspace, normalized)
	if len(targets.order) == 0 || len(normalized.OldEvidencePointers) == 0 {
		return RehydratePlan{}
	}

	oldEvidenceKeys := buildRehydratePlanOldEvidenceKeySet(rehydrateWorkspace, normalized.OldEvidencePointers)
	var plan RehydratePlan
	seen := make(map[rehydratePlanItemKey]struct{})
	totalLines := 0
	for _, targetPath := range targets.order {
		target := targets.targets[targetPath]
		if !target.staleWarning && hasNonStaleEvidenceOutsideOldPointers(state, targetPath, rehydrateWorkspace, oldEvidenceKeys) {
			continue
		}
		for _, pointer := range normalized.OldEvidencePointers {
			if len(plan.Items) >= normalized.MaxItems || totalLines >= normalized.MaxTotalLines {
				return plan
			}
			item, ok := rehydratePlanItemForPointer(pointer, targetPath, target, rehydrateWorkspace, normalized, normalized.MaxTotalLines-totalLines)
			if !ok {
				continue
			}
			key := rehydratePlanItemKey{path: item.Path, startLine: item.StartLine, endLine: item.EndLine}
			if _, exists := seen[key]; exists {
				continue
			}
			seen[key] = struct{}{}
			plan.Items = append(plan.Items, item)
			totalLines += item.EndLine - item.StartLine + 1
		}
	}
	return plan
}

// BuildRehydratePlan は Store の現在 state/workspace から再読込 dry-run 計画を作る。
func (s *Store) BuildRehydratePlan(opts RehydratePlanOptions) RehydratePlan {
	if s == nil {
		return RehydratePlan{}
	}
	s.mu.Lock()
	state := s.state.clone()
	observations := cloneEditReadinessObservations(s.editReadinessObservations)
	workspace := EvidenceRehydrateOptions{RepoRoot: s.repoRoot, InvocationCWD: s.invocationCWD}
	s.mu.Unlock()

	if len(observations) > 0 {
		opts.EditReadinessObservations = append(cloneEditReadinessObservations(observations), opts.EditReadinessObservations...)
	}
	return BuildRehydratePlan(state, workspace, opts)
}

func normalizeRehydratePlanOptions(opts RehydratePlanOptions) RehydratePlanOptions {
	normalized := RehydratePlanOptions{
		TargetPaths:               append([]string(nil), opts.TargetPaths...),
		OldEvidencePointers:       cloneEvidencePointers(opts.OldEvidencePointers),
		EditReadinessObservations: cloneEditReadinessObservations(opts.EditReadinessObservations),
		MaxItems:                  opts.MaxItems,
		MaxLinesPerItem:           opts.MaxLinesPerItem,
		MaxTotalLines:             opts.MaxTotalLines,
	}
	if normalized.MaxItems <= 0 {
		normalized.MaxItems = defaultRehydratePlanMaxItems
	}
	if normalized.MaxLinesPerItem <= 0 {
		normalized.MaxLinesPerItem = defaultRehydratePlanMaxLinesPerItem
	}
	if normalized.MaxTotalLines <= 0 {
		normalized.MaxTotalLines = defaultRehydratePlanMaxTotalLines
	}
	return normalized
}

func buildRehydratePlanTargets(state RuntimeTaskState, workspace evidenceRehydrateWorkspace, opts RehydratePlanOptions) rehydratePlanTargetSet {
	targets := rehydratePlanTargetSet{targets: make(map[string]rehydratePlanTarget)}
	for _, path := range opts.TargetPaths {
		normalized, ok := normalizeLedgerPath(workspace.repoRoot, workspace.invocationCWD, path)
		if !ok {
			continue
		}
		targets.add(normalized, func(target rehydratePlanTarget) rehydratePlanTarget {
			target.explicit = true
			return target
		})
	}
	for _, observation := range opts.EditReadinessObservations {
		normalized, ok := normalizeRehydratePlanObservationPath(workspace, observation)
		if !ok {
			continue
		}
		staleWarning := editReadinessObservationHasStaleEvidenceWarning(observation)
		targets.add(normalized, func(target rehydratePlanTarget) rehydratePlanTarget {
			target.editTarget = true
			target.staleWarning = target.staleWarning || staleWarning
			return target
		})
	}
	for _, path := range state.ChangedFiles.Paths() {
		normalized, ok := normalizeRepoRelativeLedgerPath(workspace.repoRoot, workspace.invocationCWD, path)
		if !ok {
			continue
		}
		targets.add(normalized, func(target rehydratePlanTarget) rehydratePlanTarget { return target })
	}
	return targets
}

func normalizeRehydratePlanObservationPath(workspace evidenceRehydrateWorkspace, observation EditReadinessObservation) (string, bool) {
	if observation.NormalizedPath != "" {
		return normalizeRepoRelativeLedgerPath(workspace.repoRoot, workspace.invocationCWD, observation.NormalizedPath)
	}
	return normalizeLedgerPath(workspace.repoRoot, workspace.invocationCWD, observation.Path)
}

func (s *rehydratePlanTargetSet) add(path string, update func(rehydratePlanTarget) rehydratePlanTarget) {
	if s == nil || path == "" || update == nil {
		return
	}
	if s.targets == nil {
		s.targets = make(map[string]rehydratePlanTarget)
	}
	if _, exists := s.targets[path]; !exists {
		s.order = append(s.order, path)
	}
	s.targets[path] = update(s.targets[path])
}

func editReadinessObservationHasStaleEvidenceWarning(observation EditReadinessObservation) bool {
	for _, reason := range observation.Reasons {
		if reason == EditReadinessReasonStaleEvidence {
			return true
		}
	}
	for _, pointer := range observation.EvidencePointers {
		if pointer.Stale {
			return true
		}
	}
	for _, result := range observation.RehydrateResults {
		if result.Stale {
			return true
		}
	}
	return false
}

func hasNonStaleEvidenceOutsideOldPointers(state RuntimeTaskState, path string, workspace evidenceRehydrateWorkspace, oldEvidenceKeys map[rehydratePlanEvidencePointerKey]struct{}) bool {
	for _, pointer := range state.EvidencePointersForPath(path) {
		if pointer.Stale {
			continue
		}
		key, ok := rehydratePlanEvidencePointerKeyForPointer(pointer, workspace)
		if ok {
			if _, exists := oldEvidenceKeys[key]; exists {
				continue
			}
		}
		return true
	}
	return false
}

func buildRehydratePlanOldEvidenceKeySet(workspace evidenceRehydrateWorkspace, pointers []EvidencePointer) map[rehydratePlanEvidencePointerKey]struct{} {
	if len(pointers) == 0 {
		return nil
	}
	keys := make(map[rehydratePlanEvidencePointerKey]struct{})
	for _, pointer := range pointers {
		if !isRehydratePlanSupportedEvidenceSource(pointer.Source) || !validRehydratePlanPointerRange(pointer) {
			continue
		}
		key, ok := rehydratePlanEvidencePointerKeyForPointer(pointer, workspace)
		if !ok {
			continue
		}
		keys[key] = struct{}{}
	}
	return keys
}

func rehydratePlanEvidencePointerKeyForPointer(pointer EvidencePointer, workspace evidenceRehydrateWorkspace) (rehydratePlanEvidencePointerKey, bool) {
	resolved, _, err := resolveEvidencePointerPath(pointer, workspace)
	if err != nil {
		return rehydratePlanEvidencePointerKey{}, false
	}
	return rehydratePlanEvidencePointerKey{
		path:       resolved.relativePath,
		startLine:  pointer.StartLine,
		endLine:    pointer.EndLine,
		source:     pointer.Source,
		toolCallID: pointer.ToolCallID,
	}, true
}

func rehydratePlanItemForPointer(pointer EvidencePointer, targetPath string, target rehydratePlanTarget, workspace evidenceRehydrateWorkspace, opts RehydratePlanOptions, remainingLines int) (RehydratePlanItem, bool) {
	if remainingLines <= 0 || !isRehydratePlanSupportedEvidenceSource(pointer.Source) || !validRehydratePlanPointerRange(pointer) {
		return RehydratePlanItem{}, false
	}
	resolved, _, err := resolveEvidencePointerPath(pointer, workspace)
	if err != nil || resolved.relativePath != targetPath {
		return RehydratePlanItem{}, false
	}

	endLine := pointer.EndLine
	if maxEndLine := pointer.StartLine + opts.MaxLinesPerItem - 1; endLine > maxEndLine {
		endLine = maxEndLine
	}
	lineCount := endLine - pointer.StartLine + 1
	if lineCount > remainingLines {
		endLine = pointer.StartLine + remainingLines - 1
		lineCount = remainingLines
	}
	if lineCount <= 0 {
		return RehydratePlanItem{}, false
	}

	stale := pointer.Stale || target.staleWarning
	return RehydratePlanItem{
		Path:       resolved.relativePath,
		StartLine:  pointer.StartLine,
		EndLine:    endLine,
		Source:     pointer.Source,
		Reason:     rehydratePlanReason(pointer, target),
		ToolCallID: pointer.ToolCallID,
		Stale:      stale,
	}, true
}

func isRehydratePlanSupportedEvidenceSource(source string) bool {
	switch source {
	case "read_file", "search_code", "gather_context":
		return true
	default:
		return false
	}
}

func validRehydratePlanPointerRange(pointer EvidencePointer) bool {
	return pointer.StartLine > 0 && pointer.EndLine >= pointer.StartLine
}

func rehydratePlanReason(pointer EvidencePointer, target rehydratePlanTarget) string {
	switch {
	case pointer.Stale || target.staleWarning:
		return RehydratePlanReasonStaleEvidenceRequiresRefresh
	case target.explicit || target.editTarget:
		return RehydratePlanReasonEditTargetMissingEvidence
	default:
		return RehydratePlanReasonOmittedProviderHistory
	}
}
