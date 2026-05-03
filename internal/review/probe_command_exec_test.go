package review

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestExecuteProbeCommand_BlocksWhenResolvedCommandPathMissing(t *testing.T) {
	result := executeProbeCommand(context.Background(), probeExecCommand{
		command: "cat",
		args:    []string{"file.txt"},
		workDir: t.TempDir(),
	}, 2*time.Second, 1024)

	if result.Status != ReviewProbeBlocked {
		t.Fatalf("Status = %q, want %q", result.Status, ReviewProbeBlocked)
	}
	if result.ExitCode != -1 {
		t.Fatalf("ExitCode = %d, want -1", result.ExitCode)
	}
	if !strings.Contains(result.Error, probeExecMissingResolvedPathError) {
		t.Fatalf("Error = %q, want to contain %q", result.Error, probeExecMissingResolvedPathError)
	}
	if result.Command != "cat" {
		t.Fatalf("Command = %q, want %q", result.Command, "cat")
	}
	if len(result.Args) != 1 || result.Args[0] != "file.txt" {
		t.Fatalf("Args = %#v, want [file.txt]", result.Args)
	}
}

func TestExecuteProbeCommand_DoesNotRunLogicalCommandWithoutResolvedPath(t *testing.T) {
	tempDir := t.TempDir()
	markerPath := filepath.Join(tempDir, "marker.txt")
	binDir := filepath.Join(tempDir, "bin")
	commandName, _ := createProbeExecMarkerCommand(t, binDir, "probe-no-fallback", markerPath)

	result := executeProbeCommand(context.Background(), probeExecCommand{
		command: commandName,
		args:    nil,
		workDir: tempDir,
		env:     prependPathEnv(os.Environ(), binDir),
	}, 2*time.Second, 1024)

	if result.Status != ReviewProbeBlocked {
		t.Fatalf("Status = %q, want %q", result.Status, ReviewProbeBlocked)
	}
	if !strings.Contains(result.Error, probeExecMissingResolvedPathError) {
		t.Fatalf("Error = %q, want to contain %q", result.Error, probeExecMissingResolvedPathError)
	}
	if _, err := os.Stat(markerPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("marker file should not be created, stat error = %v", err)
	}
}

func TestExecuteProbeCommand_RunsResolvedCommandPath(t *testing.T) {
	tempDir := t.TempDir()
	markerPath := filepath.Join(tempDir, "marker.txt")
	binDir := filepath.Join(tempDir, "bin")
	commandName, commandPath := createProbeExecMarkerCommand(t, binDir, "probe-resolved", markerPath)

	result := executeProbeCommand(context.Background(), probeExecCommand{
		command:     commandName,
		commandPath: commandPath,
		workDir:     tempDir,
	}, 2*time.Second, 1024)

	if result.Status != ReviewProbePassed {
		t.Fatalf("Status = %q, want %q (error=%q)", result.Status, ReviewProbePassed, result.Error)
	}
	if result.ExitCode != 0 {
		t.Fatalf("ExitCode = %d, want 0 (error=%q)", result.ExitCode, result.Error)
	}
	if _, err := os.Stat(markerPath); err != nil {
		t.Fatalf("marker file should be created, stat error = %v", err)
	}
}

func createProbeExecMarkerCommand(t *testing.T, dir, baseName, markerPath string) (commandName string, commandPath string) {
	t.Helper()

	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", dir, err)
	}

	if runtime.GOOS == "windows" {
		commandName = baseName + ".cmd"
		path := filepath.Join(dir, commandName)
		content := "@echo off\r\n" +
			"echo marker>\"" + markerPath + "\"\r\n" +
			"exit /b 0\r\n"
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("WriteFile(%q) error = %v", path, err)
		}
		return commandName, path
	}

	commandName = baseName
	path := filepath.Join(dir, commandName)
	content := "#!/bin/sh\nset -eu\nprintf 'marker' > \"" + markerPath + "\"\n"
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}
	return commandName, path
}

func prependPathEnv(baseEnv []string, pathPrefix string) []string {
	env := append([]string(nil), baseEnv...)
	pathKeyIndex := -1
	for i, entry := range env {
		idx := strings.IndexByte(entry, '=')
		if idx <= 0 {
			continue
		}
		if strings.EqualFold(entry[:idx], "PATH") {
			pathKeyIndex = i
			break
		}
	}

	if pathKeyIndex == -1 {
		return append(env, "PATH="+pathPrefix)
	}

	key, currentPath, _ := strings.Cut(env[pathKeyIndex], "=")
	env[pathKeyIndex] = key + "=" + pathPrefix + string(os.PathListSeparator) + currentPath
	return env
}
