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
	result, err := s.scan(ctx, ScanRequest{
		Ref:     ref,
		Scanner: discardChunkScanner{},
	})
	if err != nil {
		return VerifyResult{Ref: ref, Reason: ReasonOf(err)}, err
	}
	return VerifyResult{
		Ref:         ref,
		OK:          true,
		ContentHash: result.ContentHash,
		SizeBytes:   result.SizeBytes,
	}, nil
}

func (s *Store) lookupRef(ctx context.Context, sessionID, refID string) (RawOutputRef, error) {
	if err := ctx.Err(); err != nil {
		return RawOutputRef{}, err
	}
	if err := validateSessionID(sessionID); err != nil {
		return RawOutputRef{}, err
	}
	if err := validateRefID(refID); err != nil {
		return RawOutputRef{}, err
	}
	state, err := s.lifecycleStateForRef(sessionID, refID)
	if err != nil {
		return RawOutputRef{}, err
	}
	if err := lifecycleUsable(state); err != nil {
		return RawOutputRef{}, err
	}
	ref := state.created.Ref
	if err := validateRef(ref); err != nil {
		return RawOutputRef{}, err
	}
	if ref.SessionID != sessionID || ref.RefID != refID {
		return RawOutputRef{}, reasonError(ReasonRefInvalid, "ref metadata does not match lookup key")
	}
	if err := validateRefMatchesRecord(ref, state.created); err != nil {
		return RawOutputRef{}, err
	}
	return ref, nil
}

func (s *Store) scan(ctx context.Context, req ScanRequest) (ScanResult, error) {
	if err := ctx.Err(); err != nil {
		return ScanResult{}, err
	}
	if req.Scanner == nil {
		return ScanResult{}, reasonError(ReasonRefInvalid, "scan request missing scanner")
	}
	if err := validateRef(req.Ref); err != nil {
		return ScanResult{}, err
	}
	state, err := s.lifecycleStateForRef(req.Ref.SessionID, req.Ref.RefID)
	if err != nil {
		return ScanResult{}, err
	}
	if err := lifecycleUsable(state); err != nil {
		return ScanResult{}, err
	}
	record := state.created
	if err := validateRefMatchesRecord(req.Ref, record); err != nil {
		return ScanResult{}, err
	}
	result, err := s.scanAndVerifyObject(ctx, record, req.Scanner)
	if err != nil {
		if scannerErr, ok := rawOutputChunkScannerErrorCause(err); ok {
			return ScanResult{}, scannerErr
		}
		if ReasonOf(err) == ReasonArtifactHashMismatch || ReasonOf(err) == ReasonPathInvalid || ReasonOf(err) == ReasonDecryptFailed {
			_ = s.appendLifecycleRecord(req.Ref, record.Artifact, recordTypeQuarantined, string(ReasonOf(err)))
		}
		return ScanResult{}, err
	}
	return result, nil
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
		if ReasonOf(err) != "" {
			return CreateResult{}, err
		}
		return CreateResult{}, Error{Reason: ReasonArtifactMaterializationFailed, Err: err}
	}
	return result, nil
}

func (s *Store) lifecycleStateForRef(sessionID, refID string) (lifecycleState, error) {
	states, err := s.loadLifecycle(sessionID)
	if err != nil {
		return lifecycleState{}, err
	}
	state, ok := states[refID]
	if !ok || state.created.RecordType == "" {
		return lifecycleState{}, reasonError(ReasonArtifactMissing, "ref %s missing", refID)
	}
	return state, nil
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

type discardChunkScanner struct{}

func (discardChunkScanner) Scan([]byte) error {
	return nil
}
