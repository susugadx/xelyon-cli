package rawoutputs

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/susugadx/xelyon-cli/internal/crypto"
)

func TestStoreCreateResolveAndVerify(t *testing.T) {
	store := newTestStore(t, StoreOptions{})
	body := "api-result-1\napi-result-2\n"

	result, err := store.Create(context.Background(), testCreateRequest("session-1", "call-curl", body))
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if result.Ref.RefID == "" || !strings.HasPrefix(result.Ref.RefID, refIDPrefix) {
		t.Fatalf("RefID = %q, want opaque rawout_ id", result.Ref.RefID)
	}
	if result.Ref.ContentHash == "" || result.Ref.ArtifactID != result.Ref.ContentHash {
		t.Fatalf("ref hash/artifact = %#v, want matching content hash", result.Ref)
	}
	if result.Ref.ByteSize != len(body) || result.Ref.Surface != string(SurfaceCommandOutput) {
		t.Fatalf("ref = %#v, want command output byte metadata", result.Ref)
	}

	resolved, err := store.Resolve(context.Background(), result.Ref)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	got, err := readResolved(resolved)
	if err != nil {
		t.Fatalf("read resolved: %v", err)
	}
	if got != body {
		t.Fatalf("resolved body = %q, want %q", got, body)
	}
	verify, err := store.Verify(context.Background(), result.Ref)
	if err != nil || !verify.OK || verify.ContentHash != result.Ref.ContentHash {
		t.Fatalf("Verify() = %#v, %v", verify, err)
	}

	objectPath, err := store.safeSessionPath("session-1", result.Artifact.RelativePath)
	if err != nil {
		t.Fatalf("object path: %v", err)
	}
	info, err := os.Stat(objectPath)
	if err != nil {
		t.Fatalf("stat object: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("object perm = %v, want 0600", info.Mode().Perm())
	}
	manifest := readFile(t, store.manifestPath("session-1"))
	if bytes.Contains(manifest, []byte(body)) {
		t.Fatalf("manifest contains raw body:\n%s", manifest)
	}
	for _, reject := range []string{
		"foo=bar",
		"token=secret",
		"#frag",
		"Bearer abcdef",
		"PASSWORD=super-secret",
	} {
		if bytes.Contains(manifest, []byte(reject)) {
			t.Fatalf("manifest contains unsanitized source metadata %q:\n%s", reject, manifest)
		}
		if strings.Contains(result.Ref.CommandPreview, reject) {
			t.Fatalf("ref command preview contains unsanitized source metadata %q: %#v", reject, result.Ref)
		}
	}
	for _, want := range []string{"?redacted", "#redacted", "authorization: bearer [redacted]", "PASSWORD=[redacted]"} {
		if !strings.Contains(strings.ToLower(result.Ref.CommandPreview), strings.ToLower(want)) {
			t.Fatalf("ref command preview = %q, want sanitized marker %q", result.Ref.CommandPreview, want)
		}
	}
}

func TestStoreRejectsSensitiveOversizedAndUnsafeRefs(t *testing.T) {
	store := newTestStore(t, StoreOptions{MaxArtifactBytes: 16, SessionQuotaBytes: 64})

	req := testCreateRequest("session-1", "call-secret", "TOKEN=secret\n")
	req.Classification.Sensitive = true
	if _, err := store.Create(context.Background(), req); ReasonOf(err) != ReasonSensitiveArtifactForbidden {
		t.Fatalf("Create(sensitive) error = %v, want %s", err, ReasonSensitiveArtifactForbidden)
	}

	req = testCreateRequest("session-body-secret", "call-body-secret", "X-Api-Key: abc\n")
	if _, err := store.Create(context.Background(), req); ReasonOf(err) != ReasonSensitiveArtifactForbidden {
		t.Fatalf("Create(sensitive body) error = %v, want %s", err, ReasonSensitiveArtifactForbidden)
	}
	if _, err := os.Stat(store.sessionRoot("session-body-secret")); !os.IsNotExist(err) {
		t.Fatalf("sensitive body created session dir or unexpected stat error: %v", err)
	}

	req = testCreateRequest("session-1", "call-big", strings.Repeat("x", 17))
	if _, err := store.Create(context.Background(), req); ReasonOf(err) != ReasonArtifactTooLarge {
		t.Fatalf("Create(oversized) error = %v, want %s", err, ReasonArtifactTooLarge)
	}

	req = testCreateRequest("../bad", "call-bad", "body")
	if _, err := store.Create(context.Background(), req); ReasonOf(err) != ReasonRefInvalid {
		t.Fatalf("Create(unsafe session) error = %v, want %s", err, ReasonRefInvalid)
	}
}

func TestStoreCreateRejectsSymlinkedPlainObjectParent(t *testing.T) {
	store := newTestStore(t, StoreOptions{})
	sessionID := "session-symlink-object"
	body := "safe response body\n"
	outside, _ := prepareSymlinkedArtifactPrefix(t, store, sessionID, body)

	if _, err := store.Create(context.Background(), testCreateRequest(sessionID, "call-symlink", body)); ReasonOf(err) != ReasonPathInvalid {
		t.Fatalf("Create(symlink object parent) error = %v, want %s", err, ReasonPathInvalid)
	}
	assertDirectoryEmpty(t, outside)
}

func TestOpenStoreRejectsSymlinkedRoot(t *testing.T) {
	target := t.TempDir()
	root := filepath.Join(t.TempDir(), "rawoutputs-link")
	if err := os.Symlink(target, root); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}

	if _, err := OpenStore(Root(root), StoreOptions{}); ReasonOf(err) != ReasonPathInvalid {
		t.Fatalf("OpenStore(symlink root) error = %v, want %s", err, ReasonPathInvalid)
	}
}

func TestOpenStoreRejectsSymlinkedRootParent(t *testing.T) {
	target := t.TempDir()
	parent := filepath.Join(t.TempDir(), "xelyon-link")
	if err := os.Symlink(target, parent); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	root := filepath.Join(parent, "history", "rawoutputs")

	if _, err := OpenStore(Root(root), StoreOptions{}); ReasonOf(err) != ReasonPathInvalid {
		t.Fatalf("OpenStore(symlink parent) error = %v, want %s", err, ReasonPathInvalid)
	}
	if entries, err := os.ReadDir(target); err != nil {
		t.Fatalf("read symlink target: %v", err)
	} else if len(entries) != 0 {
		t.Fatalf("symlink target entries = %#v, want empty rejected store root", entries)
	}
}

func TestOpenStoreReadOnlyDoesNotCreateMissingRoot(t *testing.T) {
	root := filepath.Join(t.TempDir(), "missing", "rawoutputs")
	store, err := OpenStoreReadOnly(Root(root), StoreOptions{})
	if err != nil {
		t.Fatalf("OpenStoreReadOnly() error = %v", err)
	}
	diagnostics, err := store.Diagnostics(context.Background(), DiagnosticsRequest{SessionID: "session-ro"})
	if err != nil {
		t.Fatalf("Diagnostics() error = %v", err)
	}
	if diagnostics.StoreExists {
		t.Fatalf("Diagnostics().StoreExists = true, want false")
	}
	if _, err := os.Stat(root); !os.IsNotExist(err) {
		t.Fatalf("OpenStoreReadOnly created root or unexpected stat error: %v", err)
	}
}

func TestStoreRebuildIndexRejectsSymlinkedSessionsDir(t *testing.T) {
	store := newTestStore(t, StoreOptions{})
	outside := t.TempDir()
	sessionsDir := filepath.Join(string(store.root), "sessions")
	if err := os.Symlink(outside, sessionsDir); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}

	if _, err := store.RebuildIndex(context.Background()); ReasonOf(err) != ReasonPathInvalid {
		t.Fatalf("RebuildIndex(symlink sessions dir) error = %v, want %s", err, ReasonPathInvalid)
	}
}

func TestStoreEncryptedCreateRejectsSymlinkedObjectParent(t *testing.T) {
	store := newTestStore(t, StoreOptions{EncryptionEnabled: true, Passphrase: "test-pass"})
	sessionID := "session-symlink-encrypted-object"
	body := "safe encrypted response body\n"
	outside, _ := prepareSymlinkedArtifactPrefix(t, store, sessionID, body)

	if _, err := store.Create(context.Background(), testCreateRequest(sessionID, "call-symlink-enc", body)); ReasonOf(err) != ReasonPathInvalid {
		t.Fatalf("Create(encrypted symlink object parent) error = %v, want %s", err, ReasonPathInvalid)
	}
	assertDirectoryEmpty(t, outside)
}

func TestStoreCreateRejectsSymlinkedTempDir(t *testing.T) {
	store := newTestStore(t, StoreOptions{})
	sessionID := "session-symlink-tmp"
	objectsDir := filepath.Join(store.sessionRoot(sessionID), "objects")
	if err := os.MkdirAll(objectsDir, 0o700); err != nil {
		t.Fatalf("create objects dir: %v", err)
	}
	outside := t.TempDir()
	tmpDir := filepath.Join(objectsDir, "tmp")
	if err := os.Symlink(outside, tmpDir); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}

	if _, err := store.Create(context.Background(), testCreateRequest(sessionID, "call-symlink-tmp", "safe response body\n")); ReasonOf(err) != ReasonPathInvalid {
		t.Fatalf("Create(symlink tmp dir) error = %v, want %s", err, ReasonPathInvalid)
	}
	assertDirectoryEmpty(t, outside)
}

func TestStoreResolveRejectsSymlinkedObjectParent(t *testing.T) {
	store := newTestStore(t, StoreOptions{})
	sessionID := "session-resolve-symlink-parent"
	body := "safe resolve body\n"
	result, err := store.Create(context.Background(), testCreateRequest(sessionID, "call-resolve", body))
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	outsideObject := replaceArtifactPrefixWithSymlink(t, store, sessionID, result.Artifact.RelativePath, body)

	if _, err := store.Resolve(context.Background(), result.Ref); ReasonOf(err) != ReasonPathInvalid {
		t.Fatalf("Resolve(symlink object parent) error = %v, want %s", err, ReasonPathInvalid)
	}
	if got := strings.TrimSpace(string(readFile(t, outsideObject))); got != strings.TrimSpace(body) {
		t.Fatalf("outside object changed/read unexpectedly = %q, want original outside body", got)
	}
}

func TestStoreGCRejectsSymlinkedObjectParentWithoutDeletingOutsideFile(t *testing.T) {
	store := newTestStore(t, StoreOptions{})
	sessionID := "session-gc-symlink-parent"
	body := "safe gc body\n"
	result, err := store.Create(context.Background(), testCreateRequest(sessionID, "call-gc", body))
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	outsideObject := replaceArtifactPrefixWithSymlink(t, store, sessionID, result.Artifact.RelativePath, body)

	if _, err := store.CollectGarbage(context.Background(), GCRequest{SessionID: sessionID}); ReasonOf(err) != ReasonPathInvalid {
		t.Fatalf("CollectGarbage(symlink object parent) error = %v, want %s", err, ReasonPathInvalid)
	}
	if got := strings.TrimSpace(string(readFile(t, outsideObject))); got != strings.TrimSpace(body) {
		t.Fatalf("outside object was deleted or changed = %q, want original outside body", got)
	}
}

func TestStoreQuotaUsesLiveSessionArtifacts(t *testing.T) {
	store := newTestStore(t, StoreOptions{MaxArtifactBytes: 32, SessionQuotaBytes: 20})

	if _, err := store.Create(context.Background(), testCreateRequest("session-1", "call-a", strings.Repeat("a", 12))); err != nil {
		t.Fatalf("Create(first) error = %v", err)
	}
	if _, err := store.Create(context.Background(), testCreateRequest("session-1", "call-b", strings.Repeat("b", 12))); ReasonOf(err) != ReasonSessionQuotaExceeded {
		t.Fatalf("Create(over quota) error = %v, want %s", err, ReasonSessionQuotaExceeded)
	}
}

func TestStoreCreateIsIdempotentForExistingLiveRef(t *testing.T) {
	body := strings.Repeat("a", 12)
	store := newTestStore(t, StoreOptions{MaxArtifactBytes: 32, SessionQuotaBytes: int64(len(body))})
	sessionID := "session-idempotent"

	first, err := store.Create(context.Background(), testCreateRequest(sessionID, "call-same", body))
	if err != nil {
		t.Fatalf("Create(first) error = %v", err)
	}
	second, err := store.Create(context.Background(), testCreateRequest(sessionID, "call-same", body))
	if err != nil {
		t.Fatalf("Create(second same ref) error = %v", err)
	}

	if !reflect.DeepEqual(second, first) {
		t.Fatalf("second CreateResult = %#v, want existing first result %#v", second, first)
	}
	if got := manifestRecordCount(t, store, sessionID, recordTypeCreated); got != 1 {
		t.Fatalf("created manifest records = %d, want 1 for idempotent same ref", got)
	}
	if used, err := store.sessionArtifactBytes(sessionID); err != nil || used != int64(len(body)) {
		t.Fatalf("sessionArtifactBytes() = %d, %v; want %d", used, err, len(body))
	}
}

func TestStoreCreateIdempotentRefRestoresMissingObject(t *testing.T) {
	body := strings.Repeat("r", 12)
	store := newTestStore(t, StoreOptions{MaxArtifactBytes: 32, SessionQuotaBytes: int64(len(body))})
	sessionID := "session-idempotent-restore"

	first, err := store.Create(context.Background(), testCreateRequest(sessionID, "call-same", body))
	if err != nil {
		t.Fatalf("Create(first) error = %v", err)
	}
	objectPath, err := store.safeSessionPath(sessionID, first.Artifact.RelativePath)
	if err != nil {
		t.Fatalf("object path: %v", err)
	}
	if err := os.Remove(objectPath); err != nil {
		t.Fatalf("remove object: %v", err)
	}

	second, err := store.Create(context.Background(), testCreateRequest(sessionID, "call-same", body))
	if err != nil {
		t.Fatalf("Create(second same ref) error = %v", err)
	}
	if !reflect.DeepEqual(second, first) {
		t.Fatalf("second CreateResult = %#v, want existing first result %#v", second, first)
	}
	if got := manifestRecordCount(t, store, sessionID, recordTypeCreated); got != 1 {
		t.Fatalf("created manifest records = %d, want 1 after object restore", got)
	}
	resolved, err := store.Resolve(context.Background(), first.Ref)
	if err != nil {
		t.Fatalf("Resolve(restored) error = %v", err)
	}
	got, err := readResolved(resolved)
	if err != nil {
		t.Fatalf("read restored: %v", err)
	}
	if got != body {
		t.Fatalf("restored body = %q, want %q", got, body)
	}
}

func TestStoreCreateDoesNotDoubleChargeQuotaForExistingArtifactWithDifferentRef(t *testing.T) {
	body := strings.Repeat("b", 12)
	store := newTestStore(t, StoreOptions{MaxArtifactBytes: 32, SessionQuotaBytes: int64(len(body))})

	first, err := store.Create(context.Background(), testCreateRequest("session-shared-artifact", "call-a", body))
	if err != nil {
		t.Fatalf("Create(first) error = %v", err)
	}
	secondReq := testCreateRequest("session-shared-artifact", "call-b", body)
	second, err := store.Create(context.Background(), secondReq)
	if err != nil {
		t.Fatalf("Create(second same body different ref) error = %v", err)
	}

	if first.Ref.RefID == second.Ref.RefID {
		t.Fatalf("second ref = %q, want distinct ref for different source", second.Ref.RefID)
	}
	if first.Artifact.ArtifactID != second.Artifact.ArtifactID {
		t.Fatalf("artifact IDs = %s/%s, want shared content-addressed artifact", first.Artifact.ArtifactID, second.Artifact.ArtifactID)
	}
	if got := manifestRecordCount(t, store, secondReq.SessionID, recordTypeCreated); got != 2 {
		t.Fatalf("created manifest records = %d, want one ref per source", got)
	}
	if used, err := store.sessionArtifactBytes(secondReq.SessionID); err != nil || used != int64(len(body)) {
		t.Fatalf("sessionArtifactBytes() = %d, %v; want %d", used, err, len(body))
	}
}

func TestStoreCreateRefreshesOnlyTargetSessionIndex(t *testing.T) {
	store := newTestStore(t, StoreOptions{})
	writeCorruptManifestForTest(t, store, "session-corrupt")

	result, err := store.Create(context.Background(), testCreateRequest("session-good", "call-good", "good body\n"))
	if err != nil {
		t.Fatalf("Create(good session with unrelated corrupt session) error = %v", err)
	}
	index := string(readFile(t, store.indexPath("session-good")))
	if !strings.Contains(index, result.Ref.RefID) {
		t.Fatalf("good session index missing created ref %q:\n%s", result.Ref.RefID, index)
	}
	if _, err := store.RebuildIndex(context.Background()); ReasonOf(err) != ReasonManifestCorrupt {
		t.Fatalf("RebuildIndex() error = %v, want %s for unrelated corrupt session", err, ReasonManifestCorrupt)
	}
}

func TestStoreCreateDoesNotReuseDeadRef(t *testing.T) {
	store := newTestStore(t, StoreOptions{})
	sessionID := "session-dead-ref"
	first, err := store.Create(context.Background(), testCreateRequest(sessionID, "call-a", "body\n"))
	if err != nil {
		t.Fatalf("Create(first) error = %v", err)
	}
	gc, err := store.CollectGarbage(context.Background(), GCRequest{SessionID: sessionID})
	if err != nil {
		t.Fatalf("CollectGarbage() error = %v", err)
	}
	if len(gc.TombstonedRefIDs) != 1 || gc.TombstonedRefIDs[0] != first.Ref.RefID {
		t.Fatalf("GC = %#v, want first ref tombstoned", gc)
	}

	if _, err := store.Create(context.Background(), testCreateRequest(sessionID, "call-a", "body\n")); ReasonOf(err) != ReasonArtifactTombstoned {
		t.Fatalf("Create(dead ref) error = %v, want %s", err, ReasonArtifactTombstoned)
	}
}

func TestStoreGCRefreshesOnlyTargetSessionIndex(t *testing.T) {
	store := newTestStore(t, StoreOptions{})
	sessionID := "session-gc-good"
	first, err := store.Create(context.Background(), testCreateRequest(sessionID, "call-live", "live body\n"))
	if err != nil {
		t.Fatalf("Create(first) error = %v", err)
	}
	second, err := store.Create(context.Background(), testCreateRequest(sessionID, "call-dead", "dead body\n"))
	if err != nil {
		t.Fatalf("Create(second) error = %v", err)
	}
	writeCorruptManifestForTest(t, store, "session-gc-corrupt")

	if _, err := store.CollectGarbage(context.Background(), GCRequest{SessionID: sessionID, LiveRefs: []RawOutputRef{first.Ref}}); err != nil {
		t.Fatalf("CollectGarbage(good session with unrelated corrupt session) error = %v", err)
	}
	index := string(readFile(t, store.indexPath(sessionID)))
	if !strings.Contains(index, first.Ref.RefID) {
		t.Fatalf("good session index missing live ref %q:\n%s", first.Ref.RefID, index)
	}
	if strings.Contains(index, second.Ref.RefID) {
		t.Fatalf("good session index kept tombstoned ref %q:\n%s", second.Ref.RefID, index)
	}
	if _, err := store.RebuildIndex(context.Background()); ReasonOf(err) != ReasonManifestCorrupt {
		t.Fatalf("RebuildIndex() error = %v, want %s for unrelated corrupt session", err, ReasonManifestCorrupt)
	}
}

func TestStoreResolveRejectsHashMismatchAndQuarantines(t *testing.T) {
	store := newTestStore(t, StoreOptions{})
	result, err := store.Create(context.Background(), testCreateRequest("session-1", "call-curl", "safe body\n"))
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	objectPath, err := store.safeSessionPath("session-1", result.Artifact.RelativePath)
	if err != nil {
		t.Fatalf("object path: %v", err)
	}
	if err := os.WriteFile(objectPath, []byte("corrupted"), 0o600); err != nil {
		t.Fatalf("corrupt object: %v", err)
	}

	if _, err := store.Resolve(context.Background(), result.Ref); ReasonOf(err) != ReasonArtifactHashMismatch {
		t.Fatalf("Resolve(corrupt) error = %v, want %s", err, ReasonArtifactHashMismatch)
	}
	if _, err := store.Resolve(context.Background(), result.Ref); ReasonOf(err) != ReasonArtifactQuarantined {
		t.Fatalf("Resolve(quarantined) error = %v, want %s", err, ReasonArtifactQuarantined)
	}
}

func TestStoreGCUsesCallerProvidedLiveRefs(t *testing.T) {
	store := newTestStore(t, StoreOptions{})
	first, err := store.Create(context.Background(), testCreateRequest("session-1", "call-a", "shared body\n"))
	if err != nil {
		t.Fatalf("Create(first) error = %v", err)
	}
	secondReq := testCreateRequest("session-1", "call-b", "shared body\n")
	second, err := store.Create(context.Background(), secondReq)
	if err != nil {
		t.Fatalf("Create(second) error = %v", err)
	}
	if first.Artifact.ArtifactID != second.Artifact.ArtifactID {
		t.Fatalf("same session identical body should dedupe by artifact id: %s vs %s", first.Artifact.ArtifactID, second.Artifact.ArtifactID)
	}

	dryRun, err := store.CollectGarbage(context.Background(), GCRequest{SessionID: "session-1", LiveRefs: []RawOutputRef{first.Ref}, DryRun: true})
	if err != nil {
		t.Fatalf("CollectGarbage(dry-run) error = %v", err)
	}
	if len(dryRun.TombstonedRefIDs) != 1 || dryRun.TombstonedRefIDs[0] != second.Ref.RefID || len(dryRun.CollectedArtifactIDs) != 0 {
		t.Fatalf("dry-run GC = %#v, want second ref tombstone only", dryRun)
	}
	if _, err := store.Resolve(context.Background(), second.Ref); err != nil {
		t.Fatalf("dry-run should not tombstone second ref, resolve error = %v", err)
	}

	real, err := store.CollectGarbage(context.Background(), GCRequest{SessionID: "session-1", LiveRefs: []RawOutputRef{first.Ref}})
	if err != nil {
		t.Fatalf("CollectGarbage(real) error = %v", err)
	}
	if len(real.TombstonedRefIDs) != 1 || len(real.CollectedArtifactIDs) != 0 {
		t.Fatalf("real GC = %#v, want tombstone only while first ref live", real)
	}
	if _, err := store.Resolve(context.Background(), second.Ref); ReasonOf(err) != ReasonArtifactTombstoned {
		t.Fatalf("Resolve(tombstoned second) error = %v, want %s", err, ReasonArtifactTombstoned)
	}
	if _, err := store.Resolve(context.Background(), first.Ref); err != nil {
		t.Fatalf("first ref should remain live: %v", err)
	}

	collected, err := store.CollectGarbage(context.Background(), GCRequest{SessionID: "session-1"})
	if err != nil {
		t.Fatalf("CollectGarbage(collect) error = %v", err)
	}
	if len(collected.CollectedArtifactIDs) != 1 {
		t.Fatalf("collect GC = %#v, want shared artifact collected", collected)
	}
	if _, err := store.Resolve(context.Background(), first.Ref); ReasonOf(err) != ReasonArtifactTombstoned {
		t.Fatalf("Resolve(first after tombstone) error = %v, want %s", err, ReasonArtifactTombstoned)
	}
}

func TestStoreEncryptionDoesNotLeavePlaintextArtifactsManifestOrIndex(t *testing.T) {
	store := newTestStore(t, StoreOptions{EncryptionEnabled: true, Passphrase: "test-pass"})
	body := "secret response body should not be plaintext\n"

	result, err := store.Create(context.Background(), testCreateRequest("session-enc", "call-curl", body))
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	objectPath, err := store.safeSessionPath("session-enc", result.Artifact.RelativePath)
	if err != nil {
		t.Fatalf("object path: %v", err)
	}
	for _, path := range []string{objectPath, store.manifestPath("session-enc"), store.indexPath("session-enc")} {
		raw := readFile(t, path)
		if bytes.Contains(raw, []byte(body)) {
			t.Fatalf("%s contains plaintext body", path)
		}
		if bytes.Contains(raw, []byte("raw_output_artifact_created")) {
			t.Fatalf("%s contains plaintext manifest/index event", path)
		}
	}
	if err := filepath.WalkDir(string(store.root), func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		raw := readFile(t, path)
		if bytes.Contains(raw, []byte(body)) {
			t.Fatalf("%s contains plaintext body", path)
		}
		if bytes.Contains(raw, []byte("raw_output_artifact_created")) {
			t.Fatalf("%s contains plaintext manifest/index event", path)
		}
		return nil
	}); err != nil {
		t.Fatalf("walk encrypted store: %v", err)
	}

	resolved, err := store.Resolve(context.Background(), result.Ref)
	if err != nil {
		t.Fatalf("Resolve(encrypted) error = %v", err)
	}
	got, err := readResolved(resolved)
	if err != nil {
		t.Fatalf("read resolved: %v", err)
	}
	if got != body {
		t.Fatalf("resolved encrypted body = %q, want %q", got, body)
	}
	if result.Artifact.StorageEncoding != storageEncodingEncStreamV2 {
		t.Fatalf("StorageEncoding = %q, want %q", result.Artifact.StorageEncoding, storageEncodingEncStreamV2)
	}
	rawObject := readFile(t, objectPath)
	if !bytes.HasPrefix(rawObject, []byte(crypto.SessionStreamEncryptionMagic)) {
		t.Fatalf("encrypted object does not use streaming envelope magic")
	}
}

func TestStoreEncryptedCreateIsIdempotentForExistingLiveRef(t *testing.T) {
	store := newTestStore(t, StoreOptions{EncryptionEnabled: true, Passphrase: "test-pass"})
	sessionID := "session-enc-idempotent"

	first, err := store.Create(context.Background(), testCreateRequest(sessionID, "call-curl", "encrypted body\n"))
	if err != nil {
		t.Fatalf("Create(first encrypted) error = %v", err)
	}
	second, err := store.Create(context.Background(), testCreateRequest(sessionID, "call-curl", "encrypted body\n"))
	if err != nil {
		t.Fatalf("Create(second encrypted same ref) error = %v", err)
	}

	if !reflect.DeepEqual(second, first) {
		t.Fatalf("second encrypted CreateResult = %#v, want existing first result %#v", second, first)
	}
	if got := manifestRecordCount(t, store, sessionID, recordTypeCreated); got != 1 {
		t.Fatalf("created encrypted manifest records = %d, want 1 for idempotent same ref", got)
	}
}

func TestStoreMaterializeLegacyRequiresExactSource(t *testing.T) {
	store := newTestStore(t, StoreOptions{})
	base := LegacyMaterializeRequest{CreateRequest: testCreateRequest("session-1", "call-legacy", "legacy body\n")}

	if _, err := store.MaterializeLegacy(context.Background(), base); ReasonOf(err) != ReasonLegacySourceMissing {
		t.Fatalf("MaterializeLegacy(missing source) error = %v, want %s", err, ReasonLegacySourceMissing)
	}
	base.ExactSourceID = "history:42"
	base.Ambiguous = true
	if _, err := store.MaterializeLegacy(context.Background(), base); ReasonOf(err) != ReasonLegacySourceAmbiguous {
		t.Fatalf("MaterializeLegacy(ambiguous) error = %v, want %s", err, ReasonLegacySourceAmbiguous)
	}
	base.Ambiguous = false
	result, err := store.MaterializeLegacy(context.Background(), base)
	if err != nil {
		t.Fatalf("MaterializeLegacy() error = %v", err)
	}
	if result.Ref.RefID == "" {
		t.Fatalf("MaterializeLegacy() result = %#v, want ref", result)
	}
}

func TestStoreDedupeIsSessionLocalOnly(t *testing.T) {
	store := newTestStore(t, StoreOptions{})
	first, err := store.Create(context.Background(), testCreateRequest("session-a", "call-a", "same body\n"))
	if err != nil {
		t.Fatalf("Create(session-a) error = %v", err)
	}
	second, err := store.Create(context.Background(), testCreateRequest("session-b", "call-b", "same body\n"))
	if err != nil {
		t.Fatalf("Create(session-b) error = %v", err)
	}
	if first.Artifact.ArtifactID != second.Artifact.ArtifactID {
		t.Fatalf("content hash should match for same body: %s vs %s", first.Artifact.ArtifactID, second.Artifact.ArtifactID)
	}
	firstPath, _ := store.safeSessionPath("session-a", first.Artifact.RelativePath)
	secondPath, _ := store.safeSessionPath("session-b", second.Artifact.RelativePath)
	if firstPath == secondPath {
		t.Fatalf("cross-session artifacts share object path %q", firstPath)
	}
}

func manifestRecordCount(t *testing.T, store *Store, sessionID, recordType string) int {
	t.Helper()
	count := 0
	for _, record := range manifestRecords(t, store, sessionID) {
		if record.RecordType == recordType {
			count++
		}
	}
	return count
}

func manifestRecords(t *testing.T, store *Store, sessionID string) []ManifestRecord {
	t.Helper()
	raw := strings.TrimSpace(string(readFile(t, store.manifestPath(sessionID))))
	if raw == "" {
		return nil
	}
	lines := strings.Split(raw, "\n")
	records := make([]ManifestRecord, 0, len(lines))
	for _, line := range lines {
		record, err := store.decodeManifestLine([]byte(line))
		if err != nil {
			t.Fatalf("decode manifest line: %v\nline=%s", err, line)
		}
		records = append(records, record)
	}
	return records
}

func writeCorruptManifestForTest(t *testing.T, store *Store, sessionID string) {
	t.Helper()
	if err := store.ensureSessionDirs(sessionID); err != nil {
		t.Fatalf("ensureSessionDirs(%q): %v", sessionID, err)
	}
	if err := os.WriteFile(store.manifestPath(sessionID), []byte("{not-json}\n"), 0o600); err != nil {
		t.Fatalf("write corrupt manifest: %v", err)
	}
}

func newTestStore(t *testing.T, opts StoreOptions) *Store {
	t.Helper()
	if opts.Now == nil {
		opts.Now = func() time.Time { return time.Date(2026, 6, 6, 0, 0, 0, 0, time.UTC) }
	}
	store, err := OpenStore(Root(filepath.Join(t.TempDir(), "rawoutputs")), opts)
	if err != nil {
		t.Fatalf("OpenStore() error = %v", err)
	}
	return store
}

func testCreateRequest(sessionID, callID, body string) CreateRequest {
	return CreateRequest{
		Surface:   SurfaceCommandOutput,
		SessionID: sessionID,
		Source: SourceMetadata{
			Provider:       "openai",
			Model:          "gpt-test",
			CommandHash:    "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			CommandPreview: "curl -H 'Authorization: Bearer abcdef' 'https://api.example.test/items?foo=bar&token=secret#frag' PASSWORD=super-secret",
			ToolName:       "bash",
			ToolCallID:     callID,
			EventID:        "history:42",
			HistoryIndex:   42,
		},
		Classification: ClassificationMetadata{
			SemanticRole: "data_bearing",
			Family:       "network",
			Classifier:   "network_response",
		},
		Body:          strings.NewReader(body),
		SizeHintBytes: int64(len(body)),
	}
}

func readResolved(resolved ResolvedArtifact) (string, error) {
	defer resolved.Body.Close()
	data, err := ioReadAll(resolved.Body)
	return string(data), err
}

func ioReadAll(r interface{ Read([]byte) (int, error) }) ([]byte, error) {
	var buf bytes.Buffer
	_, err := buf.ReadFrom(r)
	return buf.Bytes(), err
}

func readFile(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return data
}

func prepareSymlinkedArtifactPrefix(t *testing.T, store *Store, sessionID, body string) (string, string) {
	t.Helper()
	hashHex := sha256HexForString(body)
	prefix := filepath.Join(store.sessionRoot(sessionID), "objects", "sha256", hashHex[:2])
	if err := os.MkdirAll(filepath.Dir(prefix), 0o700); err != nil {
		t.Fatalf("create symlink parent: %v", err)
	}
	outside := t.TempDir()
	if err := os.Symlink(outside, prefix); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	return outside, hashHex
}

func replaceArtifactPrefixWithSymlink(t *testing.T, store *Store, sessionID, relativePath, outsideBody string) string {
	t.Helper()
	hashHex := strings.TrimSuffix(filepath.Base(relativePath), ".raw")
	objectPath, err := store.safeSessionPath(sessionID, relativePath)
	if err != nil {
		t.Fatalf("object path: %v", err)
	}
	prefix := filepath.Join(store.sessionRoot(sessionID), "objects", "sha256", hashHex[:2])
	if err := os.Remove(objectPath); err != nil && !os.IsNotExist(err) {
		t.Fatalf("remove object: %v", err)
	}
	if err := os.Remove(filepath.Dir(objectPath)); err != nil && !os.IsNotExist(err) {
		t.Fatalf("remove object leaf dir: %v", err)
	}
	if err := os.Remove(prefix); err != nil && !os.IsNotExist(err) {
		t.Fatalf("remove object prefix dir: %v", err)
	}
	outside := t.TempDir()
	outsideObject := filepath.Join(outside, hashHex[2:4], hashHex+".raw")
	if err := os.MkdirAll(filepath.Dir(outsideObject), 0o700); err != nil {
		t.Fatalf("create outside object dir: %v", err)
	}
	if err := os.WriteFile(outsideObject, []byte(outsideBody), 0o600); err != nil {
		t.Fatalf("write outside object: %v", err)
	}
	if err := os.Symlink(outside, prefix); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	return outsideObject
}

func sha256HexForString(value string) string {
	hash := sha256.Sum256([]byte(value))
	return hex.EncodeToString(hash[:])
}

func assertDirectoryEmpty(t *testing.T, path string) {
	t.Helper()
	entries, err := os.ReadDir(path)
	if err != nil {
		t.Fatalf("read outside dir: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("outside dir %s has entries after rejected store operation: %#v", path, entries)
	}
}
