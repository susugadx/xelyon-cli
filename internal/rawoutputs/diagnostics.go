package rawoutputs

import (
	"context"
	"os"
	"sort"
)

const defaultDiagnosticRefLimit = 50

func (s *Store) diagnostics(ctx context.Context, req DiagnosticsRequest) (DiagnosticsResult, error) {
	if err := ctx.Err(); err != nil {
		return DiagnosticsResult{}, err
	}
	result := DiagnosticsResult{
		Root:               string(s.root),
		SessionID:          req.SessionID,
		LiveRefSourceCount: len(req.LiveRefs),
	}
	if err := validateSessionID(req.SessionID); err != nil {
		return result, err
	}
	result.StoreExists = s.storeRootExists()
	states, err := s.loadLifecycle(req.SessionID)
	if err != nil {
		return result, err
	}

	liveRefs := diagnosticLiveRefSet(req.SessionID, req.LiveRefs)
	artifactIDs := map[string]struct{}{}
	refIDs := sortedLifecycleRefIDs(states)
	refLimit := req.RefLimit
	if refLimit <= 0 {
		refLimit = defaultDiagnosticRefLimit
	}

	for _, refID := range refIDs {
		state := states[refID]
		if state.created.RecordType != recordTypeCreated {
			continue
		}
		result.RefCount++
		artifact := state.created.Artifact
		if artifact.ArtifactID != "" {
			if _, ok := artifactIDs[artifact.ArtifactID]; !ok {
				artifactIDs[artifact.ArtifactID] = struct{}{}
				result.ByteSize += int64(artifact.ByteSize)
			}
		}
		lifecycle, reason := diagnosticLifecycle(state)
		switch lifecycle {
		case "quarantined":
			result.QuarantinedRefs++
		case "tombstoned":
			result.TombstonedRefs++
		case "collected":
			result.CollectedRefs++
		}
		if req.IncludeVerify && reason == "" {
			verifyReason := s.diagnosticVerifyReason(state.created)
			reason = verifyReason
			switch verifyReason {
			case ReasonArtifactMissing:
				result.MissingObjects++
			case ReasonArtifactHashMismatch:
				result.HashMismatches++
			case ReasonDecryptFailed:
				result.DecryptFailures++
			case ReasonPathInvalid:
				result.PathFailures++
			}
		}
		if req.IncludeRefs && len(result.Refs) < refLimit {
			result.Refs = append(result.Refs, RefDiagnostic{
				Ref:          state.created.Ref,
				Artifact:     artifact,
				Lifecycle:    lifecycle,
				LiveStatus:   diagnosticLiveStatus(req.LiveRefs, liveRefs, state.created.Ref),
				VerifyReason: reason,
			})
		}
	}
	result.ArtifactCount = len(artifactIDs)
	if req.IncludeGCDryRun {
		if len(req.LiveRefs) == 0 {
			result.GCDryRunUnavailableReason = "unknown_live_state"
		} else {
			gcResult, err := s.collectGarbage(ctx, GCRequest{
				SessionID: req.SessionID,
				LiveRefs:  req.LiveRefs,
				DryRun:    true,
			})
			if err != nil {
				return result, err
			}
			result.GCDryRun = gcResult
			result.GCDryRunAvailable = true
		}
	}
	return result, nil
}

func (s *Store) storeRootExists() bool {
	info, err := os.Lstat(string(s.root))
	return err == nil && info.IsDir() && info.Mode()&os.ModeSymlink == 0
}

func sortedLifecycleRefIDs(states map[string]lifecycleState) []string {
	refIDs := make([]string, 0, len(states))
	for refID := range states {
		refIDs = append(refIDs, refID)
	}
	sort.Strings(refIDs)
	return refIDs
}

func diagnosticLifecycle(state lifecycleState) (string, Reason) {
	switch {
	case state.created.RecordType == "":
		return "missing", ReasonArtifactMissing
	case state.quarantined:
		return "quarantined", ReasonArtifactQuarantined
	case state.tombstoned:
		return "tombstoned", ReasonArtifactTombstoned
	case state.collected:
		return "collected", ReasonArtifactGCCollected
	default:
		return "created", ""
	}
}

func (s *Store) diagnosticVerifyReason(record ManifestRecord) Reason {
	if _, err := s.readAndVerifyObject(record); err != nil {
		return ReasonOf(err)
	}
	return ""
}

func diagnosticLiveRefSet(sessionID string, refs []RawOutputRef) map[string]struct{} {
	if len(refs) == 0 {
		return nil
	}
	out := map[string]struct{}{}
	for _, ref := range refs {
		if ref.SessionID == sessionID && ref.RefID != "" {
			out[ref.RefID] = struct{}{}
		}
	}
	return out
}

func diagnosticLiveStatus(sourceRefs []RawOutputRef, liveRefs map[string]struct{}, ref RawOutputRef) string {
	if len(sourceRefs) == 0 {
		return "unknown"
	}
	if _, ok := liveRefs[ref.RefID]; ok {
		return "live"
	}
	return "not_live"
}
