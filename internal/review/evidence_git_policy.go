package review

import "strings"

type reviewEvidenceGitConfigOverride struct {
	key   string
	value string
}

type reviewEvidenceGitConfigPolicy struct {
	sideEffectSuppression []reviewEvidenceGitConfigOverride
	outputDeterminism     []reviewEvidenceGitConfigOverride
}

type reviewEvidenceGitEnvPolicy struct {
	repoSelectionDenylist     []string
	sideEffectDenylist        []string
	sideEffectPrefixDenylist  []string
	outputDeterminismDenylist []string
}

func (c reviewEvidenceGitConfigOverride) argValue() string {
	return c.key + "=" + c.value
}

func (p reviewEvidenceGitConfigPolicy) overrides() []reviewEvidenceGitConfigOverride {
	total := len(p.sideEffectSuppression) + len(p.outputDeterminism)
	overrides := make([]reviewEvidenceGitConfigOverride, 0, total)
	overrides = append(overrides, p.sideEffectSuppression...)
	overrides = append(overrides, p.outputDeterminism...)
	return overrides
}

var reviewEvidenceGitConcreteConfigPolicy = reviewEvidenceGitConfigPolicy{
	sideEffectSuppression: []reviewEvidenceGitConfigOverride{
		{key: "core.fsmonitor", value: "false"},
		{key: "core.untrackedCache", value: "false"},
		{key: "diff.external", value: ""},
	},
	outputDeterminism: []reviewEvidenceGitConfigOverride{
		{key: "color.ui", value: "false"},
		{key: "color.diff", value: "false"},
		{key: "color.status", value: "false"},
		{key: "diff.renames", value: "true"},
	},
}

var reviewEvidenceGitConcreteEnvPolicy = reviewEvidenceGitEnvPolicy{
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

func buildReviewEvidenceGitArgs(repoRoot string, commandArgs []string) []string {
	configOverrides := reviewEvidenceGitConcreteConfigPolicy.overrides()
	args := make([]string, 0, len(configOverrides)*2+2+len(commandArgs))
	for _, override := range configOverrides {
		args = append(args, "-c", override.argValue())
	}
	args = append(args, "-C", repoRoot)
	args = append(args, commandArgs...)
	return args
}

func cleanReviewEvidenceGitEnv(environ []string) []string {
	cleaned := make([]string, 0, len(environ))
	for _, entry := range environ {
		key, _, _ := strings.Cut(entry, "=")
		if reviewEvidenceGitConcreteEnvPolicy.denies(key) {
			continue
		}
		cleaned = append(cleaned, entry)
	}
	return cleaned
}

func (p reviewEvidenceGitEnvPolicy) denies(key string) bool {
	normalized := strings.ToUpper(key)
	return p.deniesExact(normalized, p.repoSelectionDenylist) ||
		p.deniesExact(normalized, p.sideEffectDenylist) ||
		p.deniesPrefix(normalized, p.sideEffectPrefixDenylist) ||
		p.deniesExact(normalized, p.outputDeterminismDenylist)
}

func (p reviewEvidenceGitEnvPolicy) deniesExact(normalized string, denylist []string) bool {
	for _, denied := range denylist {
		if normalized == strings.ToUpper(denied) {
			return true
		}
	}
	return false
}

func (p reviewEvidenceGitEnvPolicy) deniesPrefix(normalized string, denylist []string) bool {
	for _, deniedPrefix := range denylist {
		if strings.HasPrefix(normalized, strings.ToUpper(deniedPrefix)) {
			return true
		}
	}
	return false
}

func buildReviewEvidenceGitEnv(environ []string) []string {
	cleaned := cleanReviewEvidenceGitEnv(environ)
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
