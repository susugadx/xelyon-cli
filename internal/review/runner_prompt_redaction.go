package review

import (
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

const (
	reviewRunnerPromptCWDDisplay          = "<cwd>"
	reviewRunnerPromptProbeWorkDirDisplay = "<probe_workdir>"
)

// reviewRunnerPromptRedactor は Pass2 prompt に渡す probe result/summary を安全な表示 path に置き換える。
// 次フェーズでは final ReviewReport の trusted summaries redaction でも同じ replacement policy を使う。
// raw probe results / raw summaries は trace/debug/audit 用の別契約で保持し、user-facing report/prompt には redacted summary を使う。
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

func (r *reviewRunnerPromptRedactor) addProbeResultReplacements(result ReviewProbeResult, displays map[string]string) {
	r.addProbeResultTextReplacements(result.Error, displays)
	for _, path := range result.MutatedFiles {
		r.addIsolatedProbeRootReplacementForPath(path, displays)
	}
	for _, command := range result.CommandResults {
		r.addProbeWorkDirReplacement(command.WorkDir, displays)
		r.addProbeResultTextReplacements(command.Command, displays)
		for _, arg := range command.Args {
			r.addProbeResultTextReplacements(arg, displays)
		}
		r.addProbeResultTextReplacements(command.Output, displays)
		r.addProbeResultTextReplacements(command.Error, displays)
	}
}

func (r *reviewRunnerPromptRedactor) addProbeWorkDirReplacement(path string, displays map[string]string) {
	if !r.isOutsideRepoAbsolutePath(path) {
		return
	}
	cleaned := normalizeReviewRunnerPromptPath(path)
	if cleaned == "" {
		return
	}

	display := r.probeWorkDirDisplay(cleaned, displays)
	if root, ok := reviewRunnerPromptIsolatedProbeRoot(cleaned); ok {
		r.addReplacement(root, display)
	}
	r.addReplacement(cleaned, display)
}

func (r *reviewRunnerPromptRedactor) addProbeResultTextReplacements(text string, displays map[string]string) {
	for _, root := range reviewRunnerPromptIsolatedProbeRootsInText(text) {
		r.addIsolatedProbeRootReplacementForPath(root, displays)
	}
}

func (r *reviewRunnerPromptRedactor) addIsolatedProbeRootReplacementForPath(path string, displays map[string]string) {
	root, ok := reviewRunnerPromptIsolatedProbeRoot(path)
	if !ok || !r.isOutsideRepoAbsolutePath(root) {
		return
	}
	r.addReplacement(root, r.probeWorkDirDisplay(root, displays))
}

func (r *reviewRunnerPromptRedactor) probeWorkDirDisplay(path string, displays map[string]string) string {
	key := normalizeReviewRunnerPromptPath(path)
	if root, ok := reviewRunnerPromptIsolatedProbeRoot(key); ok {
		key = normalizeReviewRunnerPromptPath(root)
	}
	display, ok := displays[key]
	if !ok {
		display = reviewRunnerProbeWorkDirDisplay(len(displays))
		displays[key] = display
	}
	return display
}

func (r reviewRunnerPromptRedactor) redactText(text string) string {
	redacted := text
	for _, replacement := range r.replacements {
		redacted = replaceReviewRunnerPromptPath(redacted, replacement.path, replacement.display)
	}
	return redacted
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

func (r *reviewRunnerPromptRedactor) addReplacement(path, display string) {
	if display == "" {
		return
	}
	for _, variant := range reviewRunnerPromptReplacementPathVariants(path) {
		r.addReplacementVariant(variant, display)
	}
}

func (r *reviewRunnerPromptRedactor) addReplacementVariant(path, display string) {
	if path == "" {
		return
	}
	for _, replacement := range r.replacements {
		if replacement.path == path {
			return
		}
	}
	r.replacements = append(r.replacements, reviewRunnerPromptPathReplacement{
		path:    path,
		display: display,
	})
}

func (r reviewRunnerPromptRedactor) isOutsideRepoAbsolutePath(path string) bool {
	if !isReviewRunnerPromptAbsolutePath(path) {
		return false
	}
	return formatReviewEvidencePathDisplay(r.repoRoot, path) == reviewEvidenceOutsideRepoPathDisplay
}

func reviewRunnerProbeWorkDirDisplay(index int) string {
	if index == 0 {
		return reviewRunnerPromptProbeWorkDirDisplay
	}
	return reviewRunnerPromptProbeWorkDirDisplay[:len(reviewRunnerPromptProbeWorkDirDisplay)-1] + "_" + strconv.Itoa(index+1) + ">"
}

func normalizeReviewRunnerPromptPath(path string) string {
	if strings.TrimSpace(path) == "" {
		return ""
	}
	return filepath.Clean(filepath.FromSlash(path))
}

func reviewRunnerPromptReplacementPathVariants(path string) []string {
	cleaned := normalizeReviewRunnerPromptPath(path)
	if cleaned == "" {
		return nil
	}

	slashPath := reviewRunnerPromptSlashPath(cleaned)
	variants := []string{slashPath}
	if isReviewEvidenceWindowsAbsolutePath(cleaned) || isReviewEvidenceWindowsAbsolutePath(slashPath) {
		nativePath := strings.ReplaceAll(slashPath, "/", `\`)
		if nativePath != slashPath {
			variants = append(variants, nativePath)
		}
	}
	return variants
}

func reviewRunnerPromptSlashPath(path string) string {
	return strings.ReplaceAll(filepath.ToSlash(path), `\`, "/")
}

func reviewRunnerPromptIsolatedProbeRoot(path string) (string, bool) {
	cleaned := normalizeReviewRunnerPromptPath(path)
	if cleaned == "" {
		return "", false
	}

	parts := strings.Split(reviewRunnerPromptSlashPath(cleaned), "/")
	for i, part := range parts {
		for _, prefix := range reviewProbeIsolatedTempRootPrefixes {
			if strings.HasPrefix(part, prefix) {
				return filepath.FromSlash(strings.Join(parts[:i+1], "/")), true
			}
		}
	}
	return "", false
}

func reviewRunnerPromptIsolatedProbeRootsInText(text string) []string {
	if text == "" {
		return nil
	}

	seen := map[string]struct{}{}
	var roots []string
	for _, prefix := range reviewProbeIsolatedTempRootPrefixes {
		for searchStart := 0; searchStart < len(text); {
			prefixOffset := strings.Index(text[searchStart:], prefix)
			if prefixOffset < 0 {
				break
			}

			prefixStart := searchStart + prefixOffset
			candidate := text[reviewRunnerPromptFreeTextPathStart(text, prefixStart):reviewRunnerPromptFreeTextPathEnd(text, prefixStart+len(prefix))]
			candidate = strings.TrimRight(candidate, `:,.);]}"'`)
			root, ok := reviewRunnerPromptIsolatedProbeRoot(candidate)
			if ok {
				key := reviewRunnerPromptSlashPath(root)
				if _, exists := seen[key]; !exists {
					seen[key] = struct{}{}
					roots = append(roots, root)
				}
			}
			searchStart = prefixStart + len(prefix)
		}
	}
	return roots
}

func reviewRunnerPromptFreeTextPathStart(text string, prefixStart int) int {
	start := prefixStart
	for start > 0 && isReviewRunnerPromptFreeTextPathByte(text[start-1]) {
		start--
	}
	return start
}

func reviewRunnerPromptFreeTextPathEnd(text string, prefixEnd int) int {
	end := prefixEnd
	for end < len(text) && isReviewRunnerPromptFreeTextPathByte(text[end]) {
		end++
	}
	return end
}

func isReviewRunnerPromptAbsolutePath(path string) bool {
	return filepath.IsAbs(path) || isReviewEvidenceWindowsAbsolutePath(path)
}

func replaceReviewRunnerPromptPath(text, path, display string) string {
	if text == "" || path == "" {
		return text
	}

	var b strings.Builder
	last := 0
	for {
		index := strings.Index(text[last:], path)
		if index < 0 {
			b.WriteString(text[last:])
			return b.String()
		}

		start := last + index
		end := start + len(path)
		if !isReviewRunnerPromptPathBoundary(text, start, end) {
			b.WriteString(text[last:end])
			last = end
			continue
		}

		b.WriteString(text[last:start])
		b.WriteString(display)
		last = end
	}
}

func isReviewRunnerPromptPathBoundary(text string, start, end int) bool {
	beforeOK := start == 0 || !isReviewRunnerPromptPathTokenByte(text[start-1])
	afterOK := end == len(text) || text[end] == '/' || text[end] == '\\' || !isReviewRunnerPromptPathTokenByte(text[end])
	return beforeOK && afterOK
}

func isReviewRunnerPromptPathTokenByte(b byte) bool {
	return isReviewASCIIAlpha(b) || ('0' <= b && b <= '9') || b == '_' || b == '-' || b == '.'
}

func isReviewRunnerPromptFreeTextPathByte(b byte) bool {
	return isReviewRunnerPromptPathTokenByte(b) || b == '/' || b == '\\' || b == ':'
}
