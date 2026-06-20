package review

import (
	"strconv"

	reviewevidence "github.com/susugadx/xelyon-cli/internal/review/evidence"
	reviewprobe "github.com/susugadx/xelyon-cli/internal/review/probe"
)

func (r *reviewRunnerPromptRedactor) addProbeResultReplacements(result reviewprobe.ReviewProbeResult, displays map[string]string) {
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

func (r *reviewRunnerPromptRedactor) addReplacement(path, display string) {
	if display == "" {
		return
	}
	for _, variant := range reviewevidence.ReviewEvidencePathReplacementVariants(path) {
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
	return reviewevidence.FormatReviewEvidencePathDisplay(r.repoRoot, path) == reviewevidence.OutsideRepoPathDisplay
}

func reviewRunnerProbeWorkDirDisplay(index int) string {
	if index == 0 {
		return reviewRunnerPromptProbeWorkDirDisplay
	}
	return reviewRunnerPromptProbeWorkDirDisplay[:len(reviewRunnerPromptProbeWorkDirDisplay)-1] + "_" + strconv.Itoa(index+1) + ">"
}
