package rawoutputs

import (
	"bytes"
	"context"
	"errors"
	"os"
	"strings"
	"testing"
)

func TestStoreLookupRefReturnsOnlyLiveSessionRefs(t *testing.T) {
	store := newTestStore(t, StoreOptions{})
	live, err := store.Create(context.Background(), testCreateRequest("session-lookup", "call-live", "live body\n"))
	if err != nil {
		t.Fatalf("Create(live) error = %v", err)
	}
	tombstoned, err := store.Create(context.Background(), testCreateRequest("session-lookup", "call-tombstone", "tombstoned body\n"))
	if err != nil {
		t.Fatalf("Create(tombstoned) error = %v", err)
	}
	quarantined, err := store.Create(context.Background(), testCreateRequest("session-lookup", "call-quarantine", "quarantined body\n"))
	if err != nil {
		t.Fatalf("Create(quarantined) error = %v", err)
	}
	collected, err := store.Create(context.Background(), testCreateRequest("session-lookup", "call-collected", "collected body\n"))
	if err != nil {
		t.Fatalf("Create(collected) error = %v", err)
	}
	other, err := store.Create(context.Background(), testCreateRequest("session-other", "call-live", "other body\n"))
	if err != nil {
		t.Fatalf("Create(other) error = %v", err)
	}
	if err := store.appendLifecycleRecord(tombstoned.Ref, tombstoned.Artifact, recordTypeTombstoned, "test_tombstoned"); err != nil {
		t.Fatalf("append tombstoned: %v", err)
	}
	if err := store.appendLifecycleRecord(quarantined.Ref, quarantined.Artifact, recordTypeQuarantined, "test_quarantined"); err != nil {
		t.Fatalf("append quarantined: %v", err)
	}
	if err := store.appendLifecycleRecord(collected.Ref, collected.Artifact, recordTypeGCCollected, "test_collected"); err != nil {
		t.Fatalf("append collected: %v", err)
	}

	got, err := store.LookupRef(context.Background(), live.Ref.SessionID, live.Ref.RefID)
	if err != nil {
		t.Fatalf("LookupRef(live) error = %v", err)
	}
	if got.RefID != live.Ref.RefID || got.ContentHash != live.Ref.ContentHash {
		t.Fatalf("LookupRef(live) = %#v, want %#v", got, live.Ref)
	}

	tests := []struct {
		name      string
		sessionID string
		refID     string
		want      Reason
	}{
		{name: "missing", sessionID: "session-lookup", refID: "rawout_missing", want: ReasonArtifactMissing},
		{name: "cross session", sessionID: live.Ref.SessionID, refID: other.Ref.RefID, want: ReasonArtifactMissing},
		{name: "tombstoned", sessionID: tombstoned.Ref.SessionID, refID: tombstoned.Ref.RefID, want: ReasonArtifactTombstoned},
		{name: "quarantined", sessionID: quarantined.Ref.SessionID, refID: quarantined.Ref.RefID, want: ReasonArtifactQuarantined},
		{name: "collected", sessionID: collected.Ref.SessionID, refID: collected.Ref.RefID, want: ReasonArtifactGCCollected},
		{name: "invalid session", sessionID: "../bad", refID: live.Ref.RefID, want: ReasonRefInvalid},
		{name: "invalid ref", sessionID: live.Ref.SessionID, refID: "not-rawout", want: ReasonRefInvalid},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := store.LookupRef(context.Background(), tt.sessionID, tt.refID); ReasonOf(err) != tt.want {
				t.Fatalf("LookupRef() error = %v, want %s", err, tt.want)
			}
		})
	}
}

func TestStoreScanStreamsChunksAndVerifiesHash(t *testing.T) {
	store := newTestStore(t, StoreOptions{ChunkBytes: 7})
	body := strings.Repeat("chunk-boundary-body\n", 20)
	result, err := store.Create(context.Background(), testCreateRequest("session-scan", "call-scan", body))
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	scanner := &collectingChunkScanner{}
	scan, err := store.Scan(context.Background(), ScanRequest{Ref: result.Ref, Scanner: scanner})
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}
	if scanner.body.String() != body {
		t.Fatalf("scanned body = %q, want %q", scanner.body.String(), body)
	}
	if scanner.chunks < 2 {
		t.Fatalf("scanner chunks = %d, want multiple chunks", scanner.chunks)
	}
	if scan.ContentHash != result.Ref.ContentHash || scan.SizeBytes != int64(len(body)) {
		t.Fatalf("Scan() = %#v, want hash %s size %d", scan, result.Ref.ContentHash, len(body))
	}

	objectPath, err := store.safeSessionPath(result.Ref.SessionID, result.Artifact.RelativePath)
	if err != nil {
		t.Fatalf("object path: %v", err)
	}
	if err := os.WriteFile(objectPath, []byte("corrupt body\n"), 0o600); err != nil {
		t.Fatalf("corrupt object: %v", err)
	}
	if _, err := store.Scan(context.Background(), ScanRequest{Ref: result.Ref, Scanner: discardChunkScanner{}}); ReasonOf(err) != ReasonArtifactHashMismatch {
		t.Fatalf("Scan(corrupt) error = %v, want %s", err, ReasonArtifactHashMismatch)
	}
	if _, err := store.Scan(context.Background(), ScanRequest{Ref: result.Ref, Scanner: discardChunkScanner{}}); ReasonOf(err) != ReasonArtifactQuarantined {
		t.Fatalf("Scan(quarantined) error = %v, want %s", err, ReasonArtifactQuarantined)
	}
}

func TestStoreScanDecryptsEncryptedStreamWithoutPlaintextManifest(t *testing.T) {
	store := newTestStore(t, StoreOptions{EncryptionEnabled: true, Passphrase: "test-pass", ChunkBytes: 11})
	body := strings.Repeat("encrypted stream body\n", 32)
	result, err := store.Create(context.Background(), testCreateRequest("session-scan-enc", "call-enc", body))
	if err != nil {
		t.Fatalf("Create(encrypted) error = %v", err)
	}
	scanner := &collectingChunkScanner{}
	if _, err := store.Scan(context.Background(), ScanRequest{Ref: result.Ref, Scanner: scanner}); err != nil {
		t.Fatalf("Scan(encrypted) error = %v", err)
	}
	if scanner.body.String() != body {
		t.Fatalf("encrypted scanned body = %q, want %q", scanner.body.String(), body)
	}
	objectPath, err := store.safeSessionPath(result.Ref.SessionID, result.Artifact.RelativePath)
	if err != nil {
		t.Fatalf("object path: %v", err)
	}
	if bytes.Contains(readFile(t, objectPath), []byte("encrypted stream body")) {
		t.Fatalf("encrypted object contains plaintext")
	}
}

func TestStoreScanRejectsContextCancelAndReadError(t *testing.T) {
	store := newTestStore(t, StoreOptions{})
	result, err := store.Create(context.Background(), testCreateRequest("session-scan-errors", "call-errors", "body\n"))
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := store.Scan(ctx, ScanRequest{Ref: result.Ref, Scanner: discardChunkScanner{}}); err != context.Canceled {
		t.Fatalf("Scan(canceled) error = %v, want context.Canceled", err)
	}

	objectPath, err := store.safeSessionPath(result.Ref.SessionID, result.Artifact.RelativePath)
	if err != nil {
		t.Fatalf("object path: %v", err)
	}
	if err := os.Remove(objectPath); err != nil {
		t.Fatalf("remove object: %v", err)
	}
	if err := os.Mkdir(objectPath, 0o700); err != nil {
		t.Fatalf("mkdir object path: %v", err)
	}
	if _, err := store.Scan(context.Background(), ScanRequest{Ref: result.Ref, Scanner: discardChunkScanner{}}); ReasonOf(err) != ReasonPathInvalid {
		t.Fatalf("Scan(read error) error = %v, want %s", err, ReasonPathInvalid)
	}
}

func TestStoreScanEncryptedStreamContextCancelDoesNotQuarantine(t *testing.T) {
	store := newTestStore(t, StoreOptions{EncryptionEnabled: true, Passphrase: "test-pass", ChunkBytes: 8})
	body := strings.Repeat("cancel-safe encrypted body\n", 128)
	result, err := store.Create(context.Background(), testCreateRequest("session-scan-cancel-enc", "call-cancel", body))
	if err != nil {
		t.Fatalf("Create(encrypted) error = %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	scanner := cancelingChunkScanner{cancel: cancel}
	if _, err := store.Scan(ctx, ScanRequest{Ref: result.Ref, Scanner: scanner}); !errors.Is(err, context.Canceled) {
		t.Fatalf("Scan(encrypted canceled) error = %v, want context.Canceled", err)
	}
	if _, err := store.Scan(context.Background(), ScanRequest{Ref: result.Ref, Scanner: discardChunkScanner{}}); err != nil {
		t.Fatalf("Scan(after encrypted cancel) error = %v, want artifact not quarantined", err)
	}
}

func TestStoreScanEncryptedStreamConsumerErrorDoesNotQuarantine(t *testing.T) {
	store := newTestStore(t, StoreOptions{EncryptionEnabled: true, Passphrase: "test-pass", ChunkBytes: 8})
	body := strings.Repeat("consumer-error encrypted body\n", 128)
	result, err := store.Create(context.Background(), testCreateRequest("session-scan-consumer-enc", "call-consumer", body))
	if err != nil {
		t.Fatalf("Create(encrypted) error = %v", err)
	}
	sentinel := errors.New("consumer failed")
	if _, err := store.Scan(context.Background(), ScanRequest{Ref: result.Ref, Scanner: errChunkScanner{err: sentinel}}); !errors.Is(err, sentinel) {
		t.Fatalf("Scan(consumer error) error = %v, want sentinel", err)
	}
	if _, err := store.Scan(context.Background(), ScanRequest{Ref: result.Ref, Scanner: discardChunkScanner{}}); err != nil {
		t.Fatalf("Scan(after consumer error) error = %v, want artifact not quarantined", err)
	}
}

func TestStoreScanConsumerReasonErrorDoesNotQuarantine(t *testing.T) {
	tests := []struct {
		name string
		opts StoreOptions
	}{
		{name: "plain", opts: StoreOptions{ChunkBytes: 8}},
		{name: "encrypted stream", opts: StoreOptions{EncryptionEnabled: true, Passphrase: "test-pass", ChunkBytes: 8}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := newTestStore(t, tt.opts)
			body := strings.Repeat("consumer-reason-error body\n", 128)
			result, err := store.Create(context.Background(), testCreateRequest("session-scan-consumer-reason-"+strings.ReplaceAll(tt.name, " ", "-"), "call-consumer", body))
			if err != nil {
				t.Fatalf("Create() error = %v", err)
			}
			consumerErr := reasonError(ReasonPathInvalid, "consumer path-like failure")
			if _, err := store.Scan(context.Background(), ScanRequest{Ref: result.Ref, Scanner: errChunkScanner{err: consumerErr}}); ReasonOf(err) != ReasonPathInvalid {
				t.Fatalf("Scan(consumer reason error) error = %v, want %s", err, ReasonPathInvalid)
			}
			if _, err := store.Scan(context.Background(), ScanRequest{Ref: result.Ref, Scanner: discardChunkScanner{}}); err != nil {
				t.Fatalf("Scan(after consumer reason error) error = %v, want artifact not quarantined", err)
			}
			if _, err := store.Verify(context.Background(), result.Ref); err != nil {
				t.Fatalf("Verify(after consumer reason error) error = %v, want artifact not quarantined", err)
			}
		})
	}
}

type collectingChunkScanner struct {
	body   strings.Builder
	chunks int
}

func (s *collectingChunkScanner) Scan(chunk []byte) error {
	s.chunks++
	_, err := s.body.Write(chunk)
	return err
}

type cancelingChunkScanner struct {
	cancel context.CancelFunc
}

func (s cancelingChunkScanner) Scan([]byte) error {
	s.cancel()
	return context.Canceled
}

type errChunkScanner struct {
	err error
}

func (s errChunkScanner) Scan([]byte) error {
	return s.err
}
