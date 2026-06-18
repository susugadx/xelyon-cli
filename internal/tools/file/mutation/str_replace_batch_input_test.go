package mutation

import (
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/tools/file/mutation/replaceengine"
)

func TestParseBatchEditEntriesResult_InvalidJSON(t *testing.T) {
	_, result := parseBatchEditEntriesResult("not-json")
	if !result.IsTerminal() {
		t.Fatal("expected terminal result for invalid JSON")
	}
	if !strings.Contains(result.message, "Error: invalid edits JSON:") {
		t.Fatalf("unexpected result message: %s", result.message)
	}
}

func TestParseBatchEditEntriesResult_ValidJSON(t *testing.T) {
	edits, result := parseBatchEditEntriesResult(`[{"old_str":"a","new_str":"b"}]`)
	if result.IsTerminal() {
		t.Fatalf("did not expect terminal result: %s", result.message)
	}
	if len(edits) != 1 {
		t.Fatalf("expected one edit, got %d", len(edits))
	}
	if edits[0].OldStr != "a" || edits[0].NewStr != "b" {
		t.Fatalf("unexpected edit payload: %+v", edits[0])
	}
}

func TestValidateBatchEditEntries_EmptyArray(t *testing.T) {
	result := validateBatchEditEntries("test.txt", nil)
	if !result.IsTerminal() {
		t.Fatal("expected terminal result for empty edits")
	}
	if result.message != "Error: edits array is empty" {
		t.Fatalf("unexpected result message: %s", result.message)
	}
}

func TestValidateBatchEditEntries_EmptyOldStr(t *testing.T) {
	result := validateBatchEditEntries("test.txt", []replaceengine.Edit{
		{OldStr: "", NewStr: "x"},
	})
	if !result.IsTerminal() {
		t.Fatal("expected terminal result for empty old_str")
	}
	if !strings.Contains(result.message, "Error: edits[0].old_str is empty in test.txt") {
		t.Fatalf("unexpected result message: %s", result.message)
	}
}

func TestValidateBatchEditEntries_IdenticalOldAndNew(t *testing.T) {
	result := validateBatchEditEntries("test.txt", []replaceengine.Edit{
		{OldStr: "x", NewStr: "x"},
	})
	if !result.IsTerminal() {
		t.Fatal("expected terminal result for identical old/new")
	}
	if !strings.Contains(result.message, "Error: edits[0] old_str and new_str are identical (no change needed) in test.txt") {
		t.Fatalf("unexpected result message: %s", result.message)
	}
}

func TestValidateBatchEditEntries_Success(t *testing.T) {
	result := validateBatchEditEntries("test.txt", []replaceengine.Edit{
		{OldStr: "a", NewStr: "b"},
	})
	if result.IsTerminal() {
		t.Fatalf("did not expect terminal result: %s", result.message)
	}
}
