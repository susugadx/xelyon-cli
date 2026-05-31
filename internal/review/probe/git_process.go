package probe

import (
	"context"
	"os"
	"os/exec"
	"strings"
)

type reviewProbeGitProcessRequest struct {
	repoRoot       string
	args           []string
	maxOutputBytes int64
}

type reviewProbeGitProcessResult struct {
	parsedOutput string
	diagnostics  string
	truncated    bool
}

func runReviewProbeGitProcess(ctx context.Context, req reviewProbeGitProcessRequest) (reviewProbeGitProcessResult, error) {
	proc, err := newReviewProbeGitProcess(ctx, req)
	if err != nil {
		return reviewProbeGitProcessResult{}, err
	}
	streams := newReviewProbeGitProcessStreams(req.maxOutputBytes)
	streams.attach(proc)

	err = proc.Run()
	return streams.result(), err
}

func newReviewProbeGitProcess(ctx context.Context, req reviewProbeGitProcessRequest) (*exec.Cmd, error) {
	env := buildReviewProbeGitEnv(os.Environ())
	gitPath, err := ResolveCommandPath("git", CommandResolutionContext{
		RepoRoot: req.repoRoot,
		WorkDir:  req.repoRoot,
		Env:      env,
	})
	if err != nil {
		return nil, err
	}

	proc := exec.CommandContext(ctx, gitPath, buildReviewProbeGitArgs(req.repoRoot, req.args)...)
	proc.Dir = req.repoRoot
	proc.Env = env
	return proc, nil
}

type reviewProbeGitProcessStreams struct {
	stdout *cappedOutput
	stderr *cappedOutput
}

func newReviewProbeGitProcessStreams(maxOutputBytes int64) reviewProbeGitProcessStreams {
	return reviewProbeGitProcessStreams{
		stdout: newCappedOutput(maxOutputBytes),
		stderr: newCappedOutput(maxOutputBytes),
	}
}

func (s reviewProbeGitProcessStreams) attach(proc *exec.Cmd) {
	proc.Stdout = s.stdout
	proc.Stderr = s.stderr
}

func (s reviewProbeGitProcessStreams) result() reviewProbeGitProcessResult {
	stdout := s.stdout.String()
	stderr := s.stderr.String()
	return reviewProbeGitProcessResult{
		parsedOutput: stdout,
		diagnostics:  combineReviewProbeGitDiagnostics(stderr, stdout),
		truncated:    s.stdout.Truncated(),
	}
}

func combineReviewProbeGitDiagnostics(stderr, stdout string) string {
	if stderr == "" {
		return stdout
	}
	if stdout == "" {
		return stderr
	}
	return stderr + "\n" + stdout
}

type reviewProbeGitConfigOverride struct {
	key   string
	value string
}

type reviewProbeGitConfigPolicy struct {
	sideEffectSuppression []reviewProbeGitConfigOverride
	outputDeterminism     []reviewProbeGitConfigOverride
}

type reviewProbeGitEnvPolicy struct {
	repoSelectionDenylist     []string
	sideEffectDenylist        []string
	sideEffectPrefixDenylist  []string
	outputDeterminismDenylist []string
}

func (c reviewProbeGitConfigOverride) argValue() string {
	return c.key + "=" + c.value
}

func (p reviewProbeGitConfigPolicy) overrides() []reviewProbeGitConfigOverride {
	total := len(p.sideEffectSuppression) + len(p.outputDeterminism)
	overrides := make([]reviewProbeGitConfigOverride, 0, total)
	overrides = append(overrides, p.sideEffectSuppression...)
	overrides = append(overrides, p.outputDeterminism...)
	return overrides
}

var reviewProbeGitConcreteConfigPolicy = reviewProbeGitConfigPolicy{
	sideEffectSuppression: []reviewProbeGitConfigOverride{
		{key: "core.fsmonitor", value: "false"},
		{key: "core.untrackedCache", value: "false"},
		{key: "diff.external", value: ""},
	},
	outputDeterminism: []reviewProbeGitConfigOverride{
		{key: "color.ui", value: "false"},
		{key: "color.diff", value: "false"},
		{key: "color.status", value: "false"},
		{key: "diff.renames", value: "true"},
	},
}

var reviewProbeGitConcreteEnvPolicy = reviewProbeGitEnvPolicy{
	repoSelectionDenylist: []string{
		"GIT_DIR",
		"GIT_WORK_TREE",
		"GIT_COMMON_DIR",
		"GIT_INDEX_FILE",
		"GIT_OBJECT_DIRECTORY",
		"GIT_ALTERNATE_OBJECT_DIRECTORIES",
		"GIT_IMPLICIT_WORK_TREE",
		"GIT_PREFIX",
	},
	sideEffectPrefixDenylist: []string{
		"GIT_TRACE",
	},
	outputDeterminismDenylist: []string{
		"GIT_DIFF_OPTS",
		"GIT_LITERAL_PATHSPECS",
		"GIT_GLOB_PATHSPECS",
		"GIT_NOGLOB_PATHSPECS",
		"GIT_ICASE_PATHSPECS",
	},
}

func buildReviewProbeGitArgs(repoRoot string, commandArgs []string) []string {
	configOverrides := reviewProbeGitConcreteConfigPolicy.overrides()
	args := make([]string, 0, len(configOverrides)*2+2+len(commandArgs))
	for _, override := range configOverrides {
		args = append(args, "-c", override.argValue())
	}
	args = append(args, "-C", repoRoot)
	args = append(args, commandArgs...)
	return args
}

// BuildGitArgs は review git 実行に共通する deterministic / side-effect suppression args を構築する。
func BuildGitArgs(repoRoot string, commandArgs []string) []string {
	return buildReviewProbeGitArgs(repoRoot, commandArgs)
}

func cleanReviewProbeGitEnv(environ []string) []string {
	cleaned := make([]string, 0, len(environ))
	for _, entry := range environ {
		key, _, _ := strings.Cut(entry, "=")
		if reviewProbeGitConcreteEnvPolicy.denies(key) {
			continue
		}
		cleaned = append(cleaned, entry)
	}
	return cleaned
}

// CleanGitEnv は review git 実行で拒否する GIT_* 環境変数を除去する。
func CleanGitEnv(environ []string) []string {
	return cleanReviewProbeGitEnv(environ)
}

func (p reviewProbeGitEnvPolicy) denies(key string) bool {
	normalized := strings.ToUpper(key)
	return p.deniesExact(normalized, p.repoSelectionDenylist) ||
		p.deniesExact(normalized, p.sideEffectDenylist) ||
		p.deniesPrefix(normalized, p.sideEffectPrefixDenylist) ||
		p.deniesExact(normalized, p.outputDeterminismDenylist)
}

func (p reviewProbeGitEnvPolicy) deniesExact(normalized string, denylist []string) bool {
	for _, denied := range denylist {
		if normalized == strings.ToUpper(denied) {
			return true
		}
	}
	return false
}

func (p reviewProbeGitEnvPolicy) deniesPrefix(normalized string, denylist []string) bool {
	for _, deniedPrefix := range denylist {
		if strings.HasPrefix(normalized, strings.ToUpper(deniedPrefix)) {
			return true
		}
	}
	return false
}

func buildReviewProbeGitEnv(environ []string) []string {
	cleaned := cleanReviewProbeGitEnv(environ)
	env := make([]string, 0, len(cleaned)+1)
	for _, entry := range cleaned {
		key, _, _ := strings.Cut(entry, "=")
		if strings.EqualFold(key, "GIT_OPTIONAL_LOCKS") {
			continue
		}
		env = append(env, entry)
	}
	return append(env, "GIT_OPTIONAL_LOCKS=0")
}

// BuildGitEnv は review git 実行に使う環境変数を構築する。
func BuildGitEnv(environ []string) []string {
	return buildReviewProbeGitEnv(environ)
}
