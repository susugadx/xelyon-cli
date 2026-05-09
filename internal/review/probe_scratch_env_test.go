package review

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestBuildScratchOnlyEnv_AllowlistAndOverrides(t *testing.T) {
	scratchDir := t.TempDir()
	dirs, err := prepareScratchOnlyDirs(scratchDir)
	if err != nil {
		t.Fatalf("prepareScratchOnlyDirs() error = %v", err)
	}

	repoRoot := "/repo/root"
	env := buildScratchOnlyEnv([]string{
		"PATH=/usr/local/bin:/usr/bin",
		"LANG=ja_JP.UTF-8",
		"SECRET_TOKEN=secret",
		"OPENAI_API_KEY=sk-xxx",
		"HOME=/home/original",
	}, repoRoot, dirs)
	envMap := collectEnvMap(env)

	if envMap["PATH"] != "/usr/local/bin:/usr/bin" {
		t.Fatalf("PATH = %q, want inherited value", envMap["PATH"])
	}
	if envMap["LANG"] != "ja_JP.UTF-8" {
		t.Fatalf("LANG = %q, want inherited value", envMap["LANG"])
	}
	if _, ok := envMap["SECRET_TOKEN"]; ok {
		t.Fatalf("SECRET_TOKEN should not be inherited, env = %#v", env)
	}
	if _, ok := envMap["OPENAI_API_KEY"]; ok {
		t.Fatalf("OPENAI_API_KEY should not be inherited, env = %#v", env)
	}

	if envMap[scratchEnvRepoRoot] != repoRoot {
		t.Fatalf("%s = %q, want %q", scratchEnvRepoRoot, envMap[scratchEnvRepoRoot], repoRoot)
	}
	if envMap[scratchEnvScratchDir] != dirs.ScratchDir {
		t.Fatalf("%s = %q, want %q", scratchEnvScratchDir, envMap[scratchEnvScratchDir], dirs.ScratchDir)
	}
	if envMap["HOME"] != dirs.HomeDir {
		t.Fatalf("HOME = %q, want %q", envMap["HOME"], dirs.HomeDir)
	}
	if envMap["TMPDIR"] != dirs.TempDir {
		t.Fatalf("TMPDIR = %q, want %q", envMap["TMPDIR"], dirs.TempDir)
	}
	if envMap["GOCACHE"] != dirs.GoCacheDir {
		t.Fatalf("GOCACHE = %q, want %q", envMap["GOCACHE"], dirs.GoCacheDir)
	}
	if envMap["GOTOOLCHAIN"] != "local" {
		t.Fatalf("GOTOOLCHAIN = %q, want local", envMap["GOTOOLCHAIN"])
	}
	if envMap["GOPROXY"] != "off" {
		t.Fatalf("GOPROXY = %q, want off", envMap["GOPROXY"])
	}
	if envMap["PYTHONDONTWRITEBYTECODE"] != "1" {
		t.Fatalf("PYTHONDONTWRITEBYTECODE = %q, want 1", envMap["PYTHONDONTWRITEBYTECODE"])
	}
}

func TestBuildScratchOnlyEnv_InheritsSafeGoRootOnly(t *testing.T) {
	repoRoot := t.TempDir()
	safeGoRoot := filepath.Join(t.TempDir(), "go")
	if err := os.MkdirAll(safeGoRoot, 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", safeGoRoot, err)
	}
	repoGoRoot := filepath.Join(repoRoot, "go")
	if err := os.MkdirAll(repoGoRoot, 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", repoGoRoot, err)
	}

	scratchDir := t.TempDir()
	dirs, err := prepareScratchOnlyDirs(scratchDir)
	if err != nil {
		t.Fatalf("prepareScratchOnlyDirs() error = %v", err)
	}

	env := buildScratchOnlyEnv([]string{probeGoRootEnvKey + "=" + safeGoRoot}, repoRoot, dirs)
	envMap := collectEnvMap(env)
	if envMap[probeGoRootEnvKey] != filepath.Clean(safeGoRoot) {
		t.Fatalf("GOROOT = %q, want %q", envMap[probeGoRootEnvKey], filepath.Clean(safeGoRoot))
	}

	env = buildScratchOnlyEnv([]string{probeGoRootEnvKey + "=" + repoGoRoot}, repoRoot, dirs)
	envMap = collectEnvMap(env)
	if _, ok := envMap[probeGoRootEnvKey]; ok {
		t.Fatalf("repo-contained GOROOT should not be inherited, env = %#v", env)
	}
}

func TestScratchOnlyExecutor_ChildProcessUsesHardenedEnv(t *testing.T) {
	pathValue := os.Getenv("PATH")
	if strings.TrimSpace(pathValue) == "" {
		t.Skip("PATH is empty")
	}

	repo := newProbeTestRepo(t, withProbeTestRepoNoLargeFile())
	scratchDir := filepath.Join(t.TempDir(), "xelyon-review-scratch-env-child")

	executor := newScratchOnlyExecutor(repo)
	executor.baseEnv = []string{
		"PATH=" + pathValue,
		"SECRET_TOKEN=secret",
		"OPENAI_API_KEY=sk-xxx",
		"LANG=C.UTF-8",
	}
	executor.mktemp = func(dir, pattern string) (string, error) {
		if err := os.MkdirAll(scratchDir, 0o755); err != nil {
			return "", err
		}
		return scratchDir, nil
	}

	result := executor.run(context.Background(), ReviewProbeRequest{
		ID:             "scratch-env-child",
		Mode:           ReviewProbeScratchOnly,
		Timeout:        12 * time.Second,
		MaxOutputBytes: 8 * 1024,
		Files: []ReviewProbeFile{
			{
				Path: "print_env.go",
				Content: "package main\n" +
					"import (\n" +
					"\t\"encoding/json\"\n" +
					"\t\"os\"\n" +
					")\n" +
					"func main(){\n" +
					"\tkeys := []string{\"PATH\",\"SECRET_TOKEN\",\"OPENAI_API_KEY\",\"XELYON_REVIEW_REPO_ROOT\",\"XELYON_REVIEW_SCRATCH_DIR\",\"HOME\",\"TMPDIR\",\"GOCACHE\",\"GOMODCACHE\",\"GOTMPDIR\",\"GOTOOLCHAIN\",\"GOPROXY\",\"GOSUMDB\",\"PYTHONDONTWRITEBYTECODE\",\"PYTHONNOUSERSITE\",\"PIP_NO_INDEX\"}\n" +
					"\tm := map[string]string{}\n" +
					"\tfor _, k := range keys { m[k] = os.Getenv(k) }\n" +
					"\t_ = json.NewEncoder(os.Stdout).Encode(m)\n" +
					"}\n",
			},
		},
		Commands: []ReviewProbeCommand{
			{Command: "go", Args: []string{"run", "print_env.go"}},
		},
	})

	if result.Status != ReviewProbePassed {
		t.Fatalf("Status = %q, want %q (error=%q)", result.Status, ReviewProbePassed, result.Error)
	}
	if len(result.CommandResults) != 1 {
		t.Fatalf("len(CommandResults) = %d, want 1", len(result.CommandResults))
	}

	envMap := decodeEnvMapFromFirstCommandOutput(t, result)

	if !strings.Contains(envMap["PATH"], pathValue) {
		t.Fatalf("PATH = %q, want to contain %q", envMap["PATH"], pathValue)
	}
	assertIsolatedProbeEnv(t, envMap, isolatedProbeEnvExpectation{
		repoRootKey:   scratchEnvRepoRoot,
		repoRootValue: repo,
		modeRootKey:   scratchEnvScratchDir,
		modeRootValue: filepath.Clean(scratchDir),
		rootDir:       filepath.Clean(scratchDir),
	})
}
