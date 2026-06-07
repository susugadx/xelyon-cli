package rawoutputs

import (
	"bytes"
	"context"
	"io"
	"strings"
)

func (s *Store) resolve(ctx context.Context, ref RawOutputRef) (ResolvedArtifact, error) {
	if err := ctx.Err(); err != nil {
		return ResolvedArtifact{}, err
	}
	if err := validateRef(ref); err != nil {
		return ResolvedArtifact{}, err
	}
	states, err := s.loadLifecycle(ref.SessionID)
	if err != nil {
		return ResolvedArtifact{}, err
	}
	state, ok := states[ref.RefID]
	if !ok || state.created.RecordType == "" {
		return ResolvedArtifact{}, reasonError(ReasonArtifactMissing, "ref %s missing", ref.RefID)
	}
	if err := lifecycleUsable(state); err != nil {
		return ResolvedArtifact{}, err
	}
	record := state.created
	if err := validateRefMatchesRecord(ref, record); err != nil {
		return ResolvedArtifact{}, err
	}
	body, err := s.readAndVerifyObject(record)
	if err != nil {
		if ReasonOf(err) == ReasonArtifactHashMismatch || ReasonOf(err) == ReasonPathInvalid || ReasonOf(err) == ReasonDecryptFailed {
			_ = s.appendLifecycleRecord(ref, record.Artifact, recordTypeQuarantined, string(ReasonOf(err)))
		}
		return ResolvedArtifact{}, err
	}
	return ResolvedArtifact{
		Ref:         ref,
		Body:        io.NopCloser(bytes.NewReader(body)),
		SizeBytes:   int64(len(body)),
		ContentHash: record.Artifact.ContentHash,
	}, nil
}

func (s *Store) verify(ctx context.Context, ref RawOutputRef) (VerifyResult, error) {
	resolved, err := s.resolve(ctx, ref)
	if err != nil {
		return VerifyResult{Ref: ref, Reason: ReasonOf(err)}, err
	}
	_ = resolved.Body.Close()
	return VerifyResult{
		Ref:         ref,
		OK:          true,
		ContentHash: resolved.ContentHash,
		SizeBytes:   resolved.SizeBytes,
	}, nil
}

func (s *Store) materializeLegacy(ctx context.Context, req LegacyMaterializeRequest) (CreateResult, error) {
	if strings.TrimSpace(req.ExactSourceID) == "" {
		return CreateResult{}, reasonError(ReasonLegacySourceMissing, "legacy source identity is empty")
	}
	if req.Ambiguous {
		return CreateResult{}, reasonError(ReasonLegacySourceAmbiguous, "legacy source identity is ambiguous")
	}
	result, err := s.create(ctx, req.CreateRequest)
	if err != nil {
		return CreateResult{}, Error{Reason: ReasonArtifactMaterializationFailed, Err: err}
	}
	return result, nil
}

func lifecycleUsable(state lifecycleState) error {
	switch {
	case state.quarantined:
		return reasonError(ReasonArtifactQuarantined, "artifact quarantined")
	case state.tombstoned:
		return reasonError(ReasonArtifactTombstoned, "artifact tombstoned")
	case state.collected:
		return reasonError(ReasonArtifactGCCollected, "artifact gc collected")
	default:
		return nil
	}
}

func validateRefMatchesRecord(ref RawOutputRef, record ManifestRecord) error {
	if ref.RefID != record.Ref.RefID ||
		ref.SessionID != record.Ref.SessionID ||
		ref.Surface != record.Ref.Surface ||
		ref.ArtifactID != record.Ref.ArtifactID ||
		ref.ContentHash != record.Ref.ContentHash {
		return reasonError(ReasonRefInvalid, "ref metadata does not match manifest")
	}
	if ref.ContentHash != record.Artifact.ContentHash || ref.ArtifactID != record.Artifact.ArtifactID {
		return reasonError(ReasonRefInvalid, "ref artifact metadata does not match manifest")
	}
	return nil
}

func (s *Store) sessionArtifactBytes(sessionID string) (int64, error) {
	states, err := s.loadLifecycle(sessionID)
	if err != nil {
		return 0, err
	}
	return sessionArtifactBytesFromLifecycle(states), nil
}

func sessionArtifactBytesFromLifecycle(states map[string]lifecycleState) int64 {
	var total int64
	seenArtifacts := map[string]struct{}{}
	for _, state := range states {
		if state.created.RecordType == "" || lifecycleUsable(state) != nil {
			continue
		}
		artifactID := state.created.Artifact.ArtifactID
		if _, ok := seenArtifacts[artifactID]; ok {
			continue
		}
		seenArtifacts[artifactID] = struct{}{}
		total += int64(state.created.Artifact.ByteSize)
	}
	return total
}
