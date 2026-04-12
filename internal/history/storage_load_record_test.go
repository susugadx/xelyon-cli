package history

import (
	"bufio"
	"errors"
	"io"
	"strings"
	"testing"
)

type errReader struct{}

func (errReader) Read([]byte) (int, error) {
	return 0, errors.New("read failed")
}

func TestReadSessionRecord(t *testing.T) {
	t.Run("returns record without trailing delimiter", func(t *testing.T) {
		reader := bufio.NewReader(strings.NewReader("{\"role\":\"user\"}\n{\"role\":\"assistant\"}"))

		first, err := readSessionRecord(reader)
		if err != nil {
			t.Fatalf("readSessionRecord(first) error = %v", err)
		}
		if string(first) != "{\"role\":\"user\"}" {
			t.Fatalf("first record = %q, want delimiter-trimmed line", first)
		}

		second, err := readSessionRecord(reader)
		if err != nil {
			t.Fatalf("readSessionRecord(second) error = %v", err)
		}
		if string(second) != "{\"role\":\"assistant\"}" {
			t.Fatalf("second record = %q, want final unterminated line", second)
		}

		if _, err := readSessionRecord(reader); !errors.Is(err, io.EOF) {
			t.Fatalf("readSessionRecord(eof) error = %v, want EOF", err)
		}
	})

	t.Run("wraps reader failure", func(t *testing.T) {
		reader := bufio.NewReader(errReader{})

		if _, err := readSessionRecord(reader); err == nil || !strings.Contains(err.Error(), "failed to read session file") {
			t.Fatalf("readSessionRecord() error = %v, want wrapped read error", err)
		}
	})
}

func TestDecodeSessionRecord(t *testing.T) {
	t.Run("decodes plain json record", func(t *testing.T) {
		storage := &Storage{}

		msg, err := storage.decodeSessionRecord([]byte(`{"role":"user","content":"hello"}`))
		if err != nil {
			t.Fatalf("decodeSessionRecord() error = %v", err)
		}
		if msg.Role != "user" || msg.Content != "hello" {
			t.Fatalf("msg = %#v, want decoded entry", msg)
		}
	})

	t.Run("reports decrypt failure as skippable error", func(t *testing.T) {
		withStorageCryptoHooks(t)
		decryptSessionForStorage = func([]byte, string) ([]byte, error) {
			return nil, errors.New("decrypt failed")
		}

		storage := &Storage{encryption: true, passphrase: "test-pass"}
		msg, err := storage.decodeSessionRecord([]byte("ciphertext"))
		if msg != nil {
			t.Fatalf("decodeSessionRecord() msg = %#v, want nil", msg)
		}
		if err == nil || !shouldSkipSessionRecordDecodeError(err) {
			t.Fatalf("decodeSessionRecord() error = %v, want skippable decrypt error", err)
		}
	})

	t.Run("reports unmarshal failure as skippable error", func(t *testing.T) {
		storage := &Storage{}

		msg, err := storage.decodeSessionRecord([]byte("{invalid json"))
		if msg != nil {
			t.Fatalf("decodeSessionRecord() msg = %#v, want nil", msg)
		}
		if err == nil || !shouldSkipSessionRecordDecodeError(err) {
			t.Fatalf("decodeSessionRecord() error = %v, want skippable unmarshal error", err)
		}
	})
}

func TestShouldSkipSessionRecordDecodeError(t *testing.T) {
	if !shouldSkipSessionRecordDecodeError(errSessionRecordDecrypt) {
		t.Fatal("shouldSkipSessionRecordDecodeError(errSessionRecordDecrypt) = false, want true")
	}
	if !shouldSkipSessionRecordDecodeError(errSessionRecordUnmarshal) {
		t.Fatal("shouldSkipSessionRecordDecodeError(errSessionRecordUnmarshal) = false, want true")
	}
	if shouldSkipSessionRecordDecodeError(errors.New("other")) {
		t.Fatal("shouldSkipSessionRecordDecodeError(other) = true, want false")
	}
}
