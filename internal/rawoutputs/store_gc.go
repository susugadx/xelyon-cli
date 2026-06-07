package rawoutputs

import (
	"context"
	"os"
	"sort"
)

func (s *Store) collectGarbage(ctx context.Context, req GCRequest) (GCResult, error) {
	if err := ctx.Err(); err != nil {
		return GCResult{}, err
	}
	if err := validateSessionID(req.SessionID); err != nil {
		return GCResult{}, err
	}
	states, err := s.loadLifecycle(req.SessionID)
	if err != nil {
		return GCResult{}, err
	}
	liveRefs := map[string]struct{}{}
	for _, ref := range req.LiveRefs {
		if ref.SessionID == req.SessionID && ref.RefID != "" {
			liveRefs[ref.RefID] = struct{}{}
		}
	}
	liveArtifactRefs := map[string]int{}
	result := GCResult{DryRun: req.DryRun}
	for refID, state := range states {
		if state.created.RecordType == "" || state.collected || state.quarantined {
			continue
		}
		artifactID := state.created.Artifact.ArtifactID
		if _, ok := liveRefs[refID]; ok && !state.tombstoned {
			liveArtifactRefs[artifactID]++
			result.KeptArtifactIDs = appendUnique(result.KeptArtifactIDs, artifactID)
			continue
		}
		result.TombstonedRefIDs = append(result.TombstonedRefIDs, refID)
	}
	for _, state := range states {
		if state.created.RecordType == "" || state.collected {
			continue
		}
		artifactID := state.created.Artifact.ArtifactID
		if liveArtifactRefs[artifactID] > 0 {
			continue
		}
		result.CollectedArtifactIDs = appendUnique(result.CollectedArtifactIDs, artifactID)
	}
	sort.Strings(result.TombstonedRefIDs)
	sort.Strings(result.CollectedArtifactIDs)
	sort.Strings(result.KeptArtifactIDs)
	if req.DryRun {
		return result, nil
	}
	for _, refID := range result.TombstonedRefIDs {
		state := states[refID]
		if err := s.appendLifecycleRecord(state.created.Ref, state.created.Artifact, recordTypeTombstoned, "gc_unreachable"); err != nil {
			return GCResult{}, err
		}
	}
	for _, artifactID := range result.CollectedArtifactIDs {
		state, ok := firstStateForArtifact(states, artifactID)
		if !ok {
			continue
		}
		path, err := s.safeExistingSessionFilePath(req.SessionID, state.created.Artifact.RelativePath)
		if err != nil {
			return GCResult{}, err
		}
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return GCResult{}, reasonError(ReasonPathInvalid, "delete object: %w", err)
		}
		if err := s.appendLifecycleRecord(state.created.Ref, state.created.Artifact, recordTypeGCCollected, "gc_collected"); err != nil {
			return GCResult{}, err
		}
	}
	_, err = s.rebuildIndex(ctx)
	return result, err
}

func (s *Store) rebuildIndex(ctx context.Context) (IndexResult, error) {
	if err := ctx.Err(); err != nil {
		return IndexResult{}, err
	}
	sessionDirs, err := s.sessionDirs()
	if err != nil {
		return IndexResult{}, err
	}
	var total IndexResult
	for _, sessionID := range sessionDirs {
		states, err := s.loadLifecycle(sessionID)
		if err != nil {
			return IndexResult{}, err
		}
		records := make([]ManifestRecord, 0, len(states))
		for _, state := range states {
			if state.created.RecordType == "" {
				continue
			}
			total.RecordCount++
			if err := lifecycleUsable(state); err == nil {
				total.LiveRefs++
				records = append(records, state.created)
			}
		}
		if err := s.writeIndex(sessionID, records); err != nil {
			return IndexResult{}, err
		}
	}
	return total, nil
}

func (s *Store) appendLifecycleRecord(ref RawOutputRef, artifact RawOutputArtifact, recordType, reason string) error {
	return s.appendManifestRecord(ManifestRecord{
		SchemaVersion: schemaVersion,
		RecordType:    recordType,
		Ref:           ref,
		Artifact:      artifact,
		Reason:        reason,
		CreatedAt:     s.opts.Now().UTC(),
	})
}

func appendUnique(values []string, value string) []string {
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}

func firstStateForArtifact(states map[string]lifecycleState, artifactID string) (lifecycleState, bool) {
	for _, state := range states {
		if state.created.Artifact.ArtifactID == artifactID {
			return state, true
		}
	}
	return lifecycleState{}, false
}
