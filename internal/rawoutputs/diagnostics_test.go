package rawoutputs

import (
	"context"
	"os"
	"testing"
)

func TestStoreDiagnosticsVerifiesRefsAndReportsLiveStatusAndGCDryRun(t *testing.T) {
	store := newTestStore(t, StoreOptions{})
	live, err := store.Create(context.Background(), testCreateRequest("session-diagnostics", "call-live", "live body\n"))
	if err != nil {
		t.Fatalf("Create(live) error = %v", err)
	}
	stale, err := store.Create(context.Background(), testCreateRequest("session-diagnostics", "call-stale", "stale body\n"))
	if err != nil {
		t.Fatalf("Create(stale) error = %v", err)
	}

	staleObjectPath, err := store.safeSessionPath("session-diagnostics", stale.Artifact.RelativePath)
	if err != nil {
		t.Fatalf("safeSessionPath(stale) error = %v", err)
	}
	if err := os.WriteFile(staleObjectPath, []byte("corrupted body\n"), 0o600); err != nil {
		t.Fatalf("corrupt stale object: %v", err)
	}

	diagnostics, err := store.Diagnostics(context.Background(), DiagnosticsRequest{
		SessionID:       "session-diagnostics",
		LiveRefs:        []RawOutputRef{live.Ref},
		IncludeRefs:     true,
		IncludeVerify:   true,
		IncludeGCDryRun: true,
	})
	if err != nil {
		t.Fatalf("Diagnostics() error = %v", err)
	}
	if !diagnostics.StoreExists || diagnostics.RefCount != 2 || diagnostics.ArtifactCount != 2 {
		t.Fatalf("Diagnostics() = %#v, want existing store with two refs/artifacts", diagnostics)
	}
	if diagnostics.LiveRefSourceCount != 1 || diagnostics.HashMismatches != 1 {
		t.Fatalf("Diagnostics() = %#v, want one live source and one hash mismatch", diagnostics)
	}
	if !diagnostics.GCDryRunAvailable || len(diagnostics.GCDryRun.TombstonedRefIDs) != 1 || diagnostics.GCDryRun.TombstonedRefIDs[0] != stale.Ref.RefID {
		t.Fatalf("GCDryRun = %#v available=%t, want stale ref tombstoned in dry run", diagnostics.GCDryRun, diagnostics.GCDryRunAvailable)
	}

	refsByID := map[string]RefDiagnostic{}
	for _, ref := range diagnostics.Refs {
		refsByID[ref.Ref.RefID] = ref
	}
	if refsByID[live.Ref.RefID].LiveStatus != "live" || refsByID[live.Ref.RefID].VerifyReason != "" {
		t.Fatalf("live ref diagnostic = %#v, want live and verified", refsByID[live.Ref.RefID])
	}
	if refsByID[stale.Ref.RefID].LiveStatus != "not_live" || refsByID[stale.Ref.RefID].VerifyReason != ReasonArtifactHashMismatch {
		t.Fatalf("stale ref diagnostic = %#v, want not_live hash mismatch", refsByID[stale.Ref.RefID])
	}
}

func TestStoreDiagnosticsLifecycleReasonPrecedesVerification(t *testing.T) {
	store := newTestStore(t, StoreOptions{})
	sessionID := "session-diagnostics-lifecycle"
	quarantined, err := store.Create(context.Background(), testCreateRequest(sessionID, "call-quarantine", "quarantined body\n"))
	if err != nil {
		t.Fatalf("Create(quarantined) error = %v", err)
	}
	tombstoned, err := store.Create(context.Background(), testCreateRequest(sessionID, "call-tombstone", "tombstoned body\n"))
	if err != nil {
		t.Fatalf("Create(tombstoned) error = %v", err)
	}
	collected, err := store.Create(context.Background(), testCreateRequest(sessionID, "call-collected", "collected body\n"))
	if err != nil {
		t.Fatalf("Create(collected) error = %v", err)
	}

	writeRawOutputObjectForDiagnostics(t, store, sessionID, quarantined, "corrupted quarantined body\n")
	if _, err := store.Resolve(context.Background(), quarantined.Ref); ReasonOf(err) != ReasonArtifactHashMismatch {
		t.Fatalf("Resolve(corrupted quarantined) error = %v, want %s", err, ReasonArtifactHashMismatch)
	}
	if err := store.appendLifecycleRecord(tombstoned.Ref, tombstoned.Artifact, recordTypeTombstoned, "test_tombstoned"); err != nil {
		t.Fatalf("append tombstoned lifecycle: %v", err)
	}
	removeRawOutputObjectForDiagnostics(t, store, sessionID, tombstoned)
	if err := store.appendLifecycleRecord(collected.Ref, collected.Artifact, recordTypeGCCollected, "test_collected"); err != nil {
		t.Fatalf("append collected lifecycle: %v", err)
	}
	removeRawOutputObjectForDiagnostics(t, store, sessionID, collected)

	diagnostics, err := store.Diagnostics(context.Background(), DiagnosticsRequest{
		SessionID:     sessionID,
		IncludeRefs:   true,
		IncludeVerify: true,
	})
	if err != nil {
		t.Fatalf("Diagnostics() error = %v", err)
	}
	if diagnostics.QuarantinedRefs != 1 || diagnostics.TombstonedRefs != 1 || diagnostics.CollectedRefs != 1 {
		t.Fatalf("lifecycle counters = quarantine %d tombstone %d collected %d, want 1/1/1 in %#v", diagnostics.QuarantinedRefs, diagnostics.TombstonedRefs, diagnostics.CollectedRefs, diagnostics)
	}
	if diagnostics.HashMismatches != 0 || diagnostics.MissingObjects != 0 || diagnostics.DecryptFailures != 0 || diagnostics.PathFailures != 0 {
		t.Fatalf("verify counters = hash %d missing %d decrypt %d path %d, want lifecycle reasons to skip verify", diagnostics.HashMismatches, diagnostics.MissingObjects, diagnostics.DecryptFailures, diagnostics.PathFailures)
	}

	refsByID := diagnosticsRefsByID(diagnostics.Refs)
	assertDiagnosticRef(t, refsByID, quarantined.Ref.RefID, "quarantined", ReasonArtifactQuarantined)
	assertDiagnosticRef(t, refsByID, tombstoned.Ref.RefID, "tombstoned", ReasonArtifactTombstoned)
	assertDiagnosticRef(t, refsByID, collected.Ref.RefID, "collected", ReasonArtifactGCCollected)
}

func TestStoreDiagnosticsReportsUnknownLiveStateWithoutGCDryRun(t *testing.T) {
	store := newTestStore(t, StoreOptions{})
	sessionID := "session-diagnostics-unknown-live"
	first, err := store.Create(context.Background(), testCreateRequest(sessionID, "call-a", "first body\n"))
	if err != nil {
		t.Fatalf("Create(first) error = %v", err)
	}
	second, err := store.Create(context.Background(), testCreateRequest(sessionID, "call-b", "second body\n"))
	if err != nil {
		t.Fatalf("Create(second) error = %v", err)
	}

	diagnostics, err := store.Diagnostics(context.Background(), DiagnosticsRequest{
		SessionID:       sessionID,
		IncludeRefs:     true,
		IncludeGCDryRun: true,
	})
	if err != nil {
		t.Fatalf("Diagnostics() error = %v", err)
	}
	if diagnostics.GCDryRunAvailable || diagnostics.GCDryRunUnavailableReason != "unknown_live_state" {
		t.Fatalf("GC dry-run = %#v available=%t reason=%q, want unavailable unknown_live_state", diagnostics.GCDryRun, diagnostics.GCDryRunAvailable, diagnostics.GCDryRunUnavailableReason)
	}

	refsByID := diagnosticsRefsByID(diagnostics.Refs)
	for _, ref := range []RawOutputRef{first.Ref, second.Ref} {
		if refsByID[ref.RefID].LiveStatus != "unknown" {
			t.Fatalf("ref %s live status = %q, want unknown in %#v", ref.RefID, refsByID[ref.RefID].LiveStatus, refsByID[ref.RefID])
		}
	}
}

func TestStoreDiagnosticsCountsEncryptedObjectDecryptFailures(t *testing.T) {
	store := newTestStore(t, StoreOptions{EncryptionEnabled: true, Passphrase: "test-pass"})
	sessionID := "session-diagnostics-encrypted"
	result, err := store.Create(context.Background(), testCreateRequest(sessionID, "call-encrypted", "encrypted body\n"))
	if err != nil {
		t.Fatalf("Create(encrypted) error = %v", err)
	}
	writeRawOutputObjectForDiagnostics(t, store, sessionID, result, "not an encrypted object\n")

	diagnostics, err := store.Diagnostics(context.Background(), DiagnosticsRequest{
		SessionID:     sessionID,
		IncludeRefs:   true,
		IncludeVerify: true,
	})
	if err != nil {
		t.Fatalf("Diagnostics() error = %v", err)
	}
	if diagnostics.DecryptFailures != 1 || diagnostics.HashMismatches != 0 || diagnostics.MissingObjects != 0 {
		t.Fatalf("Diagnostics() = %#v, want one decrypt failure and no hash/missing counts", diagnostics)
	}
	ref := diagnosticsRefsByID(diagnostics.Refs)[result.Ref.RefID]
	if ref.VerifyReason != ReasonDecryptFailed || ref.Lifecycle != "created" {
		t.Fatalf("encrypted ref diagnostic = %#v, want created decrypt failure", ref)
	}
}

func diagnosticsRefsByID(refs []RefDiagnostic) map[string]RefDiagnostic {
	refsByID := make(map[string]RefDiagnostic, len(refs))
	for _, ref := range refs {
		refsByID[ref.Ref.RefID] = ref
	}
	return refsByID
}

func assertDiagnosticRef(t *testing.T, refs map[string]RefDiagnostic, refID, lifecycle string, reason Reason) {
	t.Helper()
	ref, ok := refs[refID]
	if !ok {
		t.Fatalf("missing diagnostic for ref %s in %#v", refID, refs)
	}
	if ref.Lifecycle != lifecycle || ref.VerifyReason != reason {
		t.Fatalf("ref %s diagnostic = %#v, want lifecycle %q reason %s", refID, ref, lifecycle, reason)
	}
}

func writeRawOutputObjectForDiagnostics(t *testing.T, store *Store, sessionID string, result CreateResult, body string) {
	t.Helper()
	objectPath, err := store.safeSessionPath(sessionID, result.Artifact.RelativePath)
	if err != nil {
		t.Fatalf("safeSessionPath(%s) error = %v", result.Artifact.RelativePath, err)
	}
	if err := os.WriteFile(objectPath, []byte(body), 0o600); err != nil {
		t.Fatalf("write diagnostic object %s: %v", objectPath, err)
	}
}

func removeRawOutputObjectForDiagnostics(t *testing.T, store *Store, sessionID string, result CreateResult) {
	t.Helper()
	objectPath, err := store.safeSessionPath(sessionID, result.Artifact.RelativePath)
	if err != nil {
		t.Fatalf("safeSessionPath(%s) error = %v", result.Artifact.RelativePath, err)
	}
	if err := os.Remove(objectPath); err != nil {
		t.Fatalf("remove diagnostic object %s: %v", objectPath, err)
	}
}
