package taskstate

// RecordEditReadinessObservation は編集前 readiness observation を Store 内部に記録する。
func (s *Store) RecordEditReadinessObservation(observation EditReadinessObservation) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.editReadinessObservations = append(s.editReadinessObservations, cloneEditReadinessObservation(observation))
}

// EditReadinessObservations は記録済みの編集前 readiness observation を防御コピーで返す。
func (s *Store) EditReadinessObservations() []EditReadinessObservation {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return cloneEditReadinessObservations(s.editReadinessObservations)
}

func cloneEditReadinessObservations(observations []EditReadinessObservation) []EditReadinessObservation {
	if len(observations) == 0 {
		return nil
	}
	cloned := make([]EditReadinessObservation, len(observations))
	for i, observation := range observations {
		cloned[i] = cloneEditReadinessObservation(observation)
	}
	return cloned
}

func cloneEditReadinessObservation(observation EditReadinessObservation) EditReadinessObservation {
	observation.Reasons = cloneEditReadinessReasons(observation.Reasons)
	observation.EvidencePointers = cloneEvidencePointers(observation.EvidencePointers)
	observation.RehydrateResults = cloneEvidenceRehydrateResults(observation.RehydrateResults)
	return observation
}

func cloneEditReadinessReasons(reasons []EditReadinessReason) []EditReadinessReason {
	if len(reasons) == 0 {
		return nil
	}
	cloned := make([]EditReadinessReason, len(reasons))
	copy(cloned, reasons)
	return cloned
}

func cloneEvidencePointers(pointers []EvidencePointer) []EvidencePointer {
	if len(pointers) == 0 {
		return nil
	}
	cloned := make([]EvidencePointer, len(pointers))
	copy(cloned, pointers)
	return cloned
}

func cloneEvidenceRehydrateResults(results []EvidenceRehydrateResult) []EvidenceRehydrateResult {
	if len(results) == 0 {
		return nil
	}
	cloned := make([]EvidenceRehydrateResult, len(results))
	copy(cloned, results)
	return cloned
}
