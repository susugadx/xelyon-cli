package review

import (
	"sort"
)

const (
	reviewRunnerPromptCWDDisplay          = "<cwd>"
	reviewRunnerPromptProbeWorkDirDisplay = "<probe_workdir>"
)

// reviewRunnerPromptRedactor は Pass2 prompt と final ReviewReport に渡す probe result/summary を安全な表示 path に置き換える。
// raw probe results / raw trusted summaries は trace/debug/audit 用の別契約で保持し、LLM が返す summaries は信頼元にしない。
type reviewRunnerPromptRedactor struct {
	repoRoot     string
	replacements []reviewRunnerPromptPathReplacement
}

type reviewRunnerPromptPathReplacement struct {
	path    string
	display string
}

func newReviewRunnerPromptRedactor(bundle ReviewEvidenceBundle, results []ReviewProbeResult) reviewRunnerPromptRedactor {
	redactor := reviewRunnerPromptRedactor{repoRoot: normalizeReviewRunnerPromptPath(bundle.RepoRoot)}
	redactor.addReplacement(bundle.RepoRoot, reviewEvidenceRepoRootPathDisplay)
	if redactor.isOutsideRepoAbsolutePath(bundle.CWD) {
		redactor.addReplacement(bundle.CWD, reviewRunnerPromptCWDDisplay)
	}

	probeWorkDirDisplays := map[string]string{}
	for _, result := range results {
		redactor.addProbeResultReplacements(result, probeWorkDirDisplays)
	}

	sort.SliceStable(redactor.replacements, func(i, j int) bool {
		return len(redactor.replacements[i].path) > len(redactor.replacements[j].path)
	})
	return redactor
}

func (r reviewRunnerPromptRedactor) redactText(text string) string {
	redacted := text
	for _, replacement := range r.replacements {
		redacted = replaceReviewRunnerPromptPath(redacted, replacement.path, replacement.display)
	}
	return redacted
}

// RedactText は prompt/model input 向けに text 内の path を表示用 path へ置き換える。
func (r reviewRunnerPromptRedactor) RedactText(text string) string {
	return r.redactText(text)
}

func (r reviewRunnerPromptRedactor) redactPath(path string) string {
	if path == "" {
		return ""
	}
	if display := formatReviewEvidencePathDisplay(r.repoRoot, path); display != reviewEvidenceOutsideRepoPathDisplay {
		return display
	}

	redacted := r.redactText(path)
	if redacted != path {
		return reviewRunnerPromptSlashPath(redacted)
	}
	if isReviewRunnerPromptAbsolutePath(path) {
		return reviewEvidenceOutsideRepoPathDisplay
	}
	return path
}

// RedactPath は prompt/model input 向けに path を安全な表示 path へ置き換える。
func (r reviewRunnerPromptRedactor) RedactPath(path string) string {
	return r.redactPath(path)
}

func (r reviewRunnerPromptRedactor) redactPaths(paths []string) []string {
	if len(paths) == 0 {
		return []string{}
	}

	redacted := make([]string, 0, len(paths))
	for _, path := range paths {
		redacted = append(redacted, r.redactPath(path))
	}
	return redacted
}

// RedactPaths は prompt/model input 向けに path 配列を安全な表示 path 配列へ置き換える。
func (r reviewRunnerPromptRedactor) RedactPaths(paths []string) []string {
	return r.redactPaths(paths)
}

func (r reviewRunnerPromptRedactor) redactTexts(values []string) []string {
	if len(values) == 0 {
		return []string{}
	}

	redacted := make([]string, 0, len(values))
	for _, value := range values {
		redacted = append(redacted, r.redactText(value))
	}
	return redacted
}

// RedactTexts は prompt/model input 向けに text 配列内の path を表示用 path へ置き換える。
func (r reviewRunnerPromptRedactor) RedactTexts(values []string) []string {
	return r.redactTexts(values)
}
