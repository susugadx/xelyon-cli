package applypatch

import (
	"reflect"
	"testing"
)

func TestParsePatch(t *testing.T) {
	_, err := parsePatchText("bad", parseModeStrict)
	assertParseError(t, err, "The first line of the patch must be '*** Begin Patch'", 0)

	_, err = parsePatchText("*** Begin Patch\nbad", parseModeStrict)
	assertParseError(t, err, "The last line of the patch must be '*** End Patch'", 0)

	parsed, err := parsePatchText(
		"*** Begin Patch \n*** Add File: foo\n+hi\n *** End Patch",
		parseModeStrict,
	)
	if err != nil {
		t.Fatalf("parsePatchText() error = %v", err)
	}
	assertHunksEqual(t, parsed.Hunks, []Hunk{{
		Type:     "add",
		Path:     "foo",
		Contents: "hi\n",
	}})

	_, err = parsePatchText(
		"*** Begin Patch\n*** Update File: test.py\n*** End Patch",
		parseModeStrict,
	)
	assertParseError(t, err, "Update file hunk for path 'test.py' is empty", 2)

	parsed, err = parsePatchText("*** Begin Patch\n*** End Patch", parseModeStrict)
	if err != nil {
		t.Fatalf("parsePatchText() error = %v", err)
	}
	assertHunksEqual(t, parsed.Hunks, []Hunk{})

	parsed, err = parsePatchText(
		"*** Begin Patch\n"+
			"*** Add File: path/add.py\n"+
			"+abc\n"+
			"+def\n"+
			"*** Delete File: path/delete.py\n"+
			"*** Update File: path/update.py\n"+
			"*** Move to: path/update2.py\n"+
			"@@ def f():\n"+
			"-    pass\n"+
			"+    return 123\n"+
			"*** End Patch",
		parseModeStrict,
	)
	if err != nil {
		t.Fatalf("parsePatchText() error = %v", err)
	}
	assertHunksEqual(t, parsed.Hunks, []Hunk{
		{
			Type:     "add",
			Path:     "path/add.py",
			Contents: "abc\ndef\n",
		},
		{
			Type: "delete",
			Path: "path/delete.py",
		},
		{
			Type:     "update",
			Path:     "path/update.py",
			MovePath: "path/update2.py",
			Chunks: []UpdateFileChunk{{
				ChangeContext: "def f():",
				OldLines:      []string{"    pass"},
				NewLines:      []string{"    return 123"},
			}},
		},
	})

	parsed, err = parsePatchText(
		"*** Begin Patch\n"+
			"*** Update File: file.py\n"+
			"@@\n"+
			"+line\n"+
			"*** Add File: other.py\n"+
			"+content\n"+
			"*** End Patch",
		parseModeStrict,
	)
	if err != nil {
		t.Fatalf("parsePatchText() error = %v", err)
	}
	assertHunksEqual(t, parsed.Hunks, []Hunk{
		{
			Type: "update",
			Path: "file.py",
			Chunks: []UpdateFileChunk{{
				OldLines: []string{},
				NewLines: []string{"line"},
			}},
		},
		{
			Type:     "add",
			Path:     "other.py",
			Contents: "content\n",
		},
	})

	parsed, err = parsePatchText(
		"*** Begin Patch\n"+
			"*** Update File: file2.py\n"+
			" import foo\n"+
			"+bar\n"+
			"*** End Patch",
		parseModeStrict,
	)
	if err != nil {
		t.Fatalf("parsePatchText() error = %v", err)
	}
	assertHunksEqual(t, parsed.Hunks, []Hunk{{
		Type: "update",
		Path: "file2.py",
		Chunks: []UpdateFileChunk{{
			OldLines: []string{"import foo"},
			NewLines: []string{"import foo", "bar"},
		}},
	}})
}

func TestParsePatchLenient(t *testing.T) {
	patchText := "*** Begin Patch\n*** Update File: file2.py\n import foo\n+bar\n*** End Patch"
	expected := []Hunk{{
		Type: "update",
		Path: "file2.py",
		Chunks: []UpdateFileChunk{{
			OldLines: []string{"import foo"},
			NewLines: []string{"import foo", "bar"},
		}},
	}}

	expectedError := "The first line of the patch must be '*** Begin Patch'"
	heredocs := []string{
		"<<EOF\n" + patchText + "\nEOF\n",
		"<<'EOF'\n" + patchText + "\nEOF\n",
		"<<\"EOF\"\n" + patchText + "\nEOF\n",
	}

	for _, heredoc := range heredocs {
		_, err := parsePatchText(heredoc, parseModeStrict)
		assertParseError(t, err, expectedError, 0)

		parsed, err := parsePatchText(heredoc, parseModeLenient)
		if err != nil {
			t.Fatalf("parsePatchText(lenient) error = %v", err)
		}
		assertHunksEqual(t, parsed.Hunks, expected)
		if parsed.Patch != patchText {
			t.Fatalf("parsed patch = %q, want %q", parsed.Patch, patchText)
		}
	}

	_, err := parsePatchText("<<\"EOF'\n"+patchText+"\nEOF\n", parseModeLenient)
	assertParseError(t, err, expectedError, 0)

	_, err = parsePatchText("<<EOF\n*** Begin Patch\n*** Update File: file2.py\nEOF\n", parseModeLenient)
	assertParseError(t, err, "The last line of the patch must be '*** End Patch'", 0)
}

func TestParseOneHunk_InvalidHeader(t *testing.T) {
	_, _, err := parseOneHunk([]string{"bad"}, 234)
	assertParseError(
		t,
		err,
		"'bad' is not a valid hunk header. Valid hunk headers: '*** Add File: {path}', '*** Delete File: {path}', '*** Update File: {path}'",
		234,
	)
}

func TestParseUpdateFileChunk(t *testing.T) {
	_, _, err := parseUpdateFileChunk([]string{"bad"}, 123, false)
	assertParseError(t, err, "Expected update hunk to start with a @@ context marker, got: 'bad'", 123)

	_, _, err = parseUpdateFileChunk([]string{"@@"}, 123, false)
	assertParseError(t, err, "Update hunk does not contain any lines", 124)

	_, _, err = parseUpdateFileChunk([]string{"@@", "bad"}, 123, false)
	assertParseError(
		t,
		err,
		"Unexpected line found in update hunk: 'bad'. Every line should start with ' ' (context line), '+' (added line), or '-' (removed line)",
		124,
	)

	_, _, err = parseUpdateFileChunk([]string{"@@", "*** End of File"}, 123, false)
	assertParseError(t, err, "Update hunk does not contain any lines", 124)

	chunk, parsedLines, err := parseUpdateFileChunk(
		[]string{
			"@@ change_context",
			"",
			" context",
			"-remove",
			"+add",
			" context2",
			"*** End Patch",
		},
		123,
		false,
	)
	if err != nil {
		t.Fatalf("parseUpdateFileChunk() error = %v", err)
	}
	assertHunksEqual(t, []Hunk{{Type: "update", Chunks: []UpdateFileChunk{chunk}}}, []Hunk{{
		Type: "update",
		Chunks: []UpdateFileChunk{{
			ChangeContext: "change_context",
			OldLines:      []string{"", "context", "remove", "context2"},
			NewLines:      []string{"", "context", "add", "context2"},
		}},
	}})
	if parsedLines != 6 {
		t.Fatalf("parsed lines = %d, want 6", parsedLines)
	}

	chunk, parsedLines, err = parseUpdateFileChunk([]string{"@@", "+line", "*** End of File"}, 123, false)
	if err != nil {
		t.Fatalf("parseUpdateFileChunk() error = %v", err)
	}
	if parsedLines != 3 {
		t.Fatalf("parsed lines = %d, want 3", parsedLines)
	}
	assertHunksEqual(t, []Hunk{{Type: "update", Chunks: []UpdateFileChunk{chunk}}}, []Hunk{{
		Type: "update",
		Chunks: []UpdateFileChunk{{
			OldLines:    []string{},
			NewLines:    []string{"line"},
			IsEndOfFile: true,
		}},
	}})
}

func assertParseError(t *testing.T, err error, message string, lineNumber int) {
	t.Helper()

	parseErr, ok := err.(*ParseError)
	if !ok {
		t.Fatalf("error = %T (%v), want *ParseError", err, err)
	}
	if parseErr.Message != message {
		t.Fatalf("message = %q, want %q", parseErr.Message, message)
	}
	if parseErr.LineNumber != lineNumber {
		t.Fatalf("line number = %d, want %d", parseErr.LineNumber, lineNumber)
	}
}

func assertHunksEqual(t *testing.T, got []Hunk, want []Hunk) {
	t.Helper()
	if !reflect.DeepEqual(normalizeHunks(got), normalizeHunks(want)) {
		t.Fatalf("hunks mismatch:\n got: %#v\nwant: %#v", got, want)
	}
}

func normalizeHunks(hunks []Hunk) []Hunk {
	cloned := make([]Hunk, len(hunks))
	for i, hunk := range hunks {
		cloned[i] = hunk
		if len(hunk.Chunks) == 0 {
			continue
		}
		cloned[i].Chunks = make([]UpdateFileChunk, len(hunk.Chunks))
		copy(cloned[i].Chunks, hunk.Chunks)
		for j := range cloned[i].Chunks {
			cloned[i].Chunks[j].previewLines = nil
		}
	}
	return cloned
}
