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
