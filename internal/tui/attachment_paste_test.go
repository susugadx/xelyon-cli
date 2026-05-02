package tui

import (
	"fmt"
	"strings"
	"testing"
)

func TestNormalizePastedPathToken_FileURIWindowsDriveConvertsInWSL(t *testing.T) {
	t.Setenv("WSL_DISTRO_NAME", "Ubuntu")
	t.Setenv("WSL_INTEROP", "")

	prevLookPath := execLookPathTUI
	prevRunOutput := runCommandOutputTUI
	execLookPathTUI = func(name string) (string, error) {
		if name != "wslpath" {
			return "", fmt.Errorf("unexpected command: %s", name)
		}
		return "/usr/bin/wslpath", nil
	}
	runCommandOutputTUI = func(name string, args ...string) ([]byte, error) {
		if name != "wslpath" {
			return nil, fmt.Errorf("unexpected command: %s", name)
		}
		if len(args) != 2 || args[0] != "-u" {
			return nil, fmt.Errorf("unexpected args: %#v", args)
		}
		if args[1] != "C:/Users/me/file.txt" {
			return nil, fmt.Errorf("unexpected path: %q", args[1])
		}
		return []byte("/mnt/c/Users/me/file.txt\n"), nil
	}
	t.Cleanup(func() {
		execLookPathTUI = prevLookPath
		runCommandOutputTUI = prevRunOutput
	})

	result := normalizePastedPathToken("file:///C:/Users/me/file.txt")
	if !result.isOK() {
		t.Fatalf("normalizePastedPathToken() status = %v, want ok", result.status)
	}
	if got := result.path; got != "/mnt/c/Users/me/file.txt" {
		t.Fatalf("normalizePastedPathToken() path = %q, want %q", got, "/mnt/c/Users/me/file.txt")
	}
}

func TestNormalizePastedPathToken_InvalidFileURIStatus(t *testing.T) {
	result := normalizePastedPathToken("file://%zz")
	if result.status != normalizePastedPathInvalidFileURI {
		t.Fatalf("normalizePastedPathToken() status = %v, want %v", result.status, normalizePastedPathInvalidFileURI)
	}
}

func TestDecodeFileURIPath_FileHostDriveKeepsDrivePrefix(t *testing.T) {
	got, ok := decodeFileURIPath("file://C:/Users/me/file.txt")
	if !ok {
		t.Fatal("decodeFileURIPath() = !ok, want ok")
	}
	if got != "C:/Users/me/file.txt" {
		t.Fatalf("decodeFileURIPath() = %q, want %q", got, "C:/Users/me/file.txt")
	}
}

func TestParseDroppedPaths_NotPath(t *testing.T) {
	result := parseDroppedPaths("this is plain text")
	if result.kind != droppedPathParseNotPath {
		t.Fatalf("parseDroppedPaths() kind = %v, want droppedPathParseNotPath", result.kind)
	}
}

func TestParseDroppedPaths_URLFallsBackToText(t *testing.T) {
	result := parseDroppedPaths("https://example.com/docs")
	if result.kind != droppedPathParseNotPath {
		t.Fatalf("parseDroppedPaths() kind = %v, want droppedPathParseNotPath", result.kind)
	}
}

func TestParseDroppedPaths_SlashContainingTextFallsBackToText(t *testing.T) {
	result := parseDroppedPaths("a/b testing")
	if result.kind != droppedPathParseNotPath {
		t.Fatalf("parseDroppedPaths() kind = %v, want droppedPathParseNotPath", result.kind)
	}
}

func TestParseDroppedPaths_Limit(t *testing.T) {
	dir := t.TempDir()
	paths := make([]string, 0, maxDroppedAttachments+1)
	for i := 0; i < maxDroppedAttachments+1; i++ {
		paths = append(paths, writeTempFile(t, dir, fmt.Sprintf("f%02d.txt", i), []byte("x")))
	}

	result := parseDroppedPaths(strings.Join(paths, "\n"))
	if result.kind != droppedPathParseLimit {
		t.Fatalf("parseDroppedPaths() kind = %v, want droppedPathParseLimit", result.kind)
	}
}

func TestParseDroppedPaths_InvalidMalformedFileURI(t *testing.T) {
	result := parseDroppedPaths("file://%zz")
	if result.kind != droppedPathParseNotPath {
		t.Fatalf("parseDroppedPaths() kind = %v, want droppedPathParseNotPath", result.kind)
	}
}

func TestParseDroppedPaths_UnterminatedQuoteFallsBackToText(t *testing.T) {
	result := parseDroppedPaths(`"/tmp/file.txt`)
	if result.kind != droppedPathParseNotPath {
		t.Fatalf("parseDroppedPaths() kind = %v, want droppedPathParseNotPath", result.kind)
	}
}

func TestParseDroppedPaths_ApostropheTextFallsBackToText(t *testing.T) {
	result := parseDroppedPaths("don't panic")
	if result.kind != droppedPathParseNotPath {
		t.Fatalf("parseDroppedPaths() kind = %v, want droppedPathParseNotPath", result.kind)
	}
}

func TestDroppedPathLines_TrimsAndSkipsEmptyLines(t *testing.T) {
	lines := droppedPathLines(" \r\n  /tmp/a  \r\n\t\r\n /tmp/b\t\n")
	if got, want := len(lines), 2; got != want {
		t.Fatalf("len(droppedPathLines()) = %d, want %d", got, want)
	}
	if lines[0] != "/tmp/a" || lines[1] != "/tmp/b" {
		t.Fatalf("droppedPathLines() = %#v, want [/tmp/a /tmp/b]", lines)
	}
}

func TestParseDroppedPaths_BareRelativeFilenameRecognized(t *testing.T) {
	dir := withTempWorkingDir(t)

	tests := []struct {
		name      string
		fileName  string
		pasteText string
	}{
		{name: "with extension", fileName: "notes.txt", pasteText: "notes.txt"},
		{name: "without extension", fileName: "Makefile", pasteText: "Makefile"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			path := writeTempFile(t, dir, tc.fileName, []byte("hello"))
			result := parseDroppedPaths(tc.pasteText)
			if result.kind != droppedPathParseReady {
				t.Fatalf("parseDroppedPaths() kind = %v, want droppedPathParseReady", result.kind)
			}
			if got, want := len(result.paths), 1; got != want {
				t.Fatalf("len(parseDroppedPaths().paths) = %d, want %d", got, want)
			}
			if result.paths[0] != path {
				t.Fatalf("parseDroppedPaths().paths[0] = %q, want %q", result.paths[0], path)
			}
		})
	}
}

func TestParseDroppedPaths_BareRelativeWordWithoutExistingFileFallsBackToText(t *testing.T) {
	_ = withTempWorkingDir(t)

	result := parseDroppedPaths("README")
	if result.kind != droppedPathParseNotPath {
		t.Fatalf("parseDroppedPaths() kind = %v, want droppedPathParseNotPath", result.kind)
	}
}

func TestDecideDroppedPathHandling(t *testing.T) {
	dir := withTempWorkingDir(t)
	existing := writeTempFile(t, dir, "notes.txt", []byte("hello"))

	tests := []struct {
		name string
		text string
		want droppedPathDecisionKind
	}{
		{name: "plain text", text: "hello world", want: droppedPathDecisionFallbackText},
		{name: "existing path", text: existing, want: droppedPathDecisionApplyCandidates},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decision := decideDroppedPathHandling(tt.text)
			if decision.kind != tt.want {
				t.Fatalf("decision.kind = %v, want %v", decision.kind, tt.want)
			}
		})
	}
}
