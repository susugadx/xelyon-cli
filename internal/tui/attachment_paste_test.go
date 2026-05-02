package tui

import (
	"fmt"
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

	got, ok := normalizePastedPathToken("file:///C:/Users/me/file.txt")
	if !ok {
		t.Fatal("normalizePastedPathToken() = !ok, want ok")
	}
	if got != "/mnt/c/Users/me/file.txt" {
		t.Fatalf("normalizePastedPathToken() = %q, want %q", got, "/mnt/c/Users/me/file.txt")
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
