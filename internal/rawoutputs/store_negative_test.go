package rawoutputs

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
)

func TestStoreRejectsInvalidRefsAndSurfaceWithStructuredReason(t *testing.T) {
	store := newTestStore(t, StoreOptions{})
	result, err := store.Create(context.Background(), testCreateRequest("session-invalid-ref", "call-valid", "valid body\n"))
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	tests := []struct {
		name string
		ref  RawOutputRef
	}{
		{
			name: "invalid session",
			ref: func() RawOutputRef {
				ref := result.Ref
				ref.SessionID = "../bad"
				return ref
			}(),
		},
		{
			name: "invalid ref id",
			ref: func() RawOutputRef {
				ref := result.Ref
				ref.RefID = "not-rawout"
				return ref
			}(),
		},
		{
			name: "invalid surface",
			ref: func() RawOutputRef {
				ref := result.Ref
				ref.Surface = "invalid_surface"
				return ref
			}(),
		},
		{
			name: "mismatched artifact hash",
			ref: func() RawOutputRef {
				ref := result.Ref
				ref.ContentHash = "sha256:" + strings.Repeat("b", 64)
				return ref
			}(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := store.Resolve(context.Background(), tt.ref); ReasonOf(err) != ReasonRefInvalid {
				t.Fatalf("Resolve() error = %v, want %s", err, ReasonRefInvalid)
			}
			verify, err := store.Verify(context.Background(), tt.ref)
			if ReasonOf(err) != ReasonRefInvalid || verify.Reason != ReasonRefInvalid {
				t.Fatalf("Verify() = %#v, %v; want reason %s", verify, err, ReasonRefInvalid)
			}
		})
	}
}

func TestStoreResolveAndVerifyExposeStructuredFailureReasons(t *testing.T) {
	t.Run("missing object", func(t *testing.T) {
		store := newTestStore(t, StoreOptions{})
		result, err := store.Create(context.Background(), testCreateRequest("session-missing-object", "call-missing", "body\n"))
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}
		objectPath, err := store.safeSessionPath(result.Ref.SessionID, result.Artifact.RelativePath)
		if err != nil {
			t.Fatalf("safeSessionPath() error = %v", err)
		}
		if err := os.Remove(objectPath); err != nil {
			t.Fatalf("remove object: %v", err)
		}

		if _, err := store.Resolve(context.Background(), result.Ref); ReasonOf(err) != ReasonArtifactMissing {
			t.Fatalf("Resolve(missing object) error = %v, want %s", err, ReasonArtifactMissing)
		}
		verify, err := store.Verify(context.Background(), result.Ref)
		if ReasonOf(err) != ReasonArtifactMissing || verify.Reason != ReasonArtifactMissing {
			t.Fatalf("Verify(missing object) = %#v, %v; want %s", verify, err, ReasonArtifactMissing)
		}
	})

	t.Run("corrupt manifest", func(t *testing.T) {
		store := newTestStore(t, StoreOptions{})
		result, err := store.Create(context.Background(), testCreateRequest("session-corrupt-lifecycle", "call-corrupt", "body\n"))
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}
		if err := os.WriteFile(store.manifestPath(result.Ref.SessionID), []byte("{not-json}\n"), 0o600); err != nil {
			t.Fatalf("write corrupt manifest: %v", err)
		}

		if _, err := store.Resolve(context.Background(), result.Ref); ReasonOf(err) != ReasonManifestCorrupt {
			t.Fatalf("Resolve(corrupt manifest) error = %v, want %s", err, ReasonManifestCorrupt)
		}
		verify, err := store.Verify(context.Background(), result.Ref)
		if ReasonOf(err) != ReasonManifestCorrupt || verify.Reason != ReasonManifestCorrupt {
			t.Fatalf("Verify(corrupt manifest) = %#v, %v; want %s", verify, err, ReasonManifestCorrupt)
		}
	})

	t.Run("manifest artifact path traversal", func(t *testing.T) {
		store := newTestStore(t, StoreOptions{})
		result, err := store.Create(context.Background(), testCreateRequest("session-path-traversal", "call-path", "body\n"))
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}
		record := result.Record
		record.Artifact.RelativePath = "../escape.raw"
		writeOnlyManifestRecord(t, store, record)

		if _, err := store.Resolve(context.Background(), result.Ref); ReasonOf(err) != ReasonPathInvalid {
			t.Fatalf("Resolve(path traversal artifact) error = %v, want %s", err, ReasonPathInvalid)
		}
	})
}

func TestStoreMaterializeLegacyFailureReasonsAreClassifiable(t *testing.T) {
	t.Run("validation reason passes through", func(t *testing.T) {
		store := newTestStore(t, StoreOptions{})
		req := LegacyMaterializeRequest{
			CreateRequest: testCreateRequest("../bad", "call-invalid", "body\n"),
			ExactSourceID: "history:invalid",
		}

		if _, err := store.MaterializeLegacy(context.Background(), req); ReasonOf(err) != ReasonRefInvalid {
			t.Fatalf("MaterializeLegacy(invalid ref) error = %v, want %s", err, ReasonRefInvalid)
		}
	})

	t.Run("unclassified create error is wrapped", func(t *testing.T) {
		store := newTestStore(t, StoreOptions{})
		sentinel := errors.New("reader failed")
		req := LegacyMaterializeRequest{
			CreateRequest: testCreateRequest("session-legacy-reader", "call-reader", ""),
			ExactSourceID: "history:reader",
		}
		req.Body = errReader{err: sentinel}
		req.SizeHintBytes = 1

		if _, err := store.MaterializeLegacy(context.Background(), req); ReasonOf(err) != ReasonArtifactMaterializationFailed || !errors.Is(err, sentinel) {
			t.Fatalf("MaterializeLegacy(reader failure) error = %v, want %s wrapping sentinel", err, ReasonArtifactMaterializationFailed)
		}
	})
}

func writeOnlyManifestRecord(t *testing.T, store *Store, record ManifestRecord) {
	t.Helper()
	line, err := store.encodeManifestLine(record)
	if err != nil {
		t.Fatalf("encode manifest line: %v", err)
	}
	if err := os.WriteFile(store.manifestPath(record.Ref.SessionID), append(line, '\n'), 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
}

type errReader struct {
	err error
}

func (r errReader) Read([]byte) (int, error) {
	return 0, r.err
}
