package fragments

import (
	"fmt"
	"strings"
)

// InvestigationToolingOptions は共有 investigation tool guidance block の構成を表す。
type InvestigationToolingOptions struct {
	AllowLowLevelOverrides bool
	SearchOverrideLabel    string
	SearchOverrideExtra    string
	ReadOverrideExtra      string
	IncludeBatchRead       bool
	BatchReadOverrideLabel string
	IncludeMultiPattern    bool
	MultiPatternExtra      string
}

// GatherContextDefaultInvestigationLine returns the shared default gather_context guidance.
func GatherContextDefaultInvestigationLine() string {
	return promptBullet("gather_context: default investigation tool. It routes direct reads, directory listing, auto search, or structured impact and may prefetch compact evidence automatically.")
}

// GatherContextFirstLine returns the shared gather_context-first investigation guidance.
func GatherContextFirstLine(extra string) string {
	core := "Start normal investigation with gather_context unless exact low-level control is clearly necessary."
	return promptBulletWithExtra(core, extra)
}

// ReviewInvestigationSentence returns the shared review/investigation tracing sentence.
func ReviewInvestigationSentence(allowLowLevelOverrides bool) string {
	if allowLowLevelOverrides {
		return "Trace callers, shared contracts, deletion paths, and error paths with gather_context first. Use search_code/read_file only when exact low-level control is clearly necessary."
	}
	return "Trace callers, shared contracts, deletion paths, and error paths with gather_context first. Refine gather_context queries before escalating to any lower-level path."
}

// GatherContextDirectMultiReadLine returns the shared direct multi-read guidance.
func GatherContextDirectMultiReadLine() string {
	return promptBullet("If every comma-separated gather_context item is an exact file or range, the runtime treats it as one direct multi-read instead of search.")
}

// GatherContextPathDisambiguationLine returns the shared path-vs-search precedence guidance.
func GatherContextPathDisambiguationLine() string {
	return promptBullet(`Bare no-extension files like Makefile can read directly via gather_context(query="Makefile"). If a name may collide with a symbol or you need scoped search, use an explicit path like gather_context(query="./Makefile").`)
}

// SharedChangeGatherContextLine returns the shared shared-change investigation guidance.
func SharedChangeGatherContextLine(extra string) string {
	core := `For shared changes, gather_context(query="SymbolName") is the default path. Prefer one combined multi-pattern gather_context query before any low-level override search.`
	return promptBulletWithExtra(core, extra)
}

// ProjectMapExactReadLine returns the shared Project Map exact-read guidance.
func ProjectMapExactReadLine(allowLowLevelOverrides bool) string {
	if allowLowLevelOverrides {
		return promptBullet(`read_file with range syntax (e.g. paths=["agent.go:161-328"]) is the expert override when you need exact manual control.`)
	}
	return promptBullet(`For exact Project Map ranges, keep using gather_context(query="agent.go:161-328") directly.`)
}

// ProjectMapKnownSymbolLine returns the shared Project Map known-symbol guidance.
func ProjectMapKnownSymbolLine(allowLowLevelOverrides bool) string {
	if allowLowLevelOverrides {
		return promptBullet("Do NOT call search_code to find symbols already listed in Project Map.")
	}
	return promptBullet("Do NOT re-search symbols already listed in Project Map.")
}

// InvestigationFollowUpLine returns the shared follow-up read guidance.
func InvestigationFollowUpLine(allowLowLevelOverrides bool, extra string) string {
	core := "After the initial gather_context, prefer another targeted gather_context query when exact manual follow-up is needed."
	if allowLowLevelOverrides {
		core = "After the initial gather_context/search_code, prefer moving to read_file only when exact manual follow-up is needed."
	}
	return promptBulletWithExtra(core, extra)
}

// LowLevelOverridesWhenExposedLine returns the shared low-level override positioning guidance.
func LowLevelOverridesWhenExposedLine() string {
	return promptBullet("When search_code/read_file are exposed, keep them as low-level expert overrides only when gather_context is clearly insufficient.")
}

// SearchCodeOverrideLine returns the shared search_code override guidance with caller-specific detail.
func SearchCodeOverrideLine(overrideLabel string, extra string) string {
	if strings.TrimSpace(overrideLabel) == "" {
		overrideLabel = "an expert override"
	}
	core := fmt.Sprintf("search_code: code discovery tool for %s. Uses language-aware routing across symbol-aware resolution, literal search, and regex search. Prefer mode=auto.", strings.TrimSpace(overrideLabel))
	return promptBulletWithExtra(core, extra)
}

// ReadFileOverrideLine returns the shared read_file override guidance with caller-specific detail.
func ReadFileOverrideLine(extra string) string {
	core := "read_file: low-level exact-content reader for expert override."
	return promptBulletWithExtra(core, extra)
}

// ReadFileBatchOverrideLine returns the shared batch read guidance.
func ReadFileBatchOverrideLine(overrideLabel string) string {
	if strings.TrimSpace(overrideLabel) == "" {
		overrideLabel = "an expert override"
	}
	return promptBullet(fmt.Sprintf("Reading 2+ independent exact files -> read_file can batch them as %s.", strings.TrimSpace(overrideLabel)))
}

// InvestigationMultiPatternLine returns the shared multi-pattern investigation guidance.
func InvestigationMultiPatternLine(allowLowLevelOverrides bool, extra string) string {
	core := "Searching multiple patterns -> prefer one gather_context query with comma-separated patterns instead of serial narrow searches."
	if allowLowLevelOverrides {
		core = "Searching multiple patterns -> prefer one gather_context query or one search_code call with comma-separated patterns instead of serial searches."
	}
	return promptBulletWithExtra(core, extra)
}

// InvestigationAllowedToolsLine は investigation prompt 用の Allowed 行を返す。
func InvestigationAllowedToolsLine(allowLowLevelOverrides bool) string {
	if allowLowLevelOverrides {
		return "Allowed: gather_context, search_code, read_file, web_search, bash (read-only git commands only: git status, git diff, git log)"
	}
	return "Allowed: gather_context, web_search, bash (read-only git commands only: git status, git diff, git log)"
}

// BuildInvestigationToolingBlock は共有 investigation tooling guidance block を返す。
func BuildInvestigationToolingBlock(opts InvestigationToolingOptions) string {
	lines := []string{
		GatherContextDefaultInvestigationLine(),
		GatherContextDirectMultiReadLine(),
		GatherContextPathDisambiguationLine(),
	}
	if opts.IncludeMultiPattern {
		lines = append(lines, InvestigationMultiPatternLine(opts.AllowLowLevelOverrides, opts.MultiPatternExtra))
	}
	if !opts.AllowLowLevelOverrides {
		return strings.Join(lines, "\n")
	}
	if strings.TrimSpace(opts.SearchOverrideLabel) != "" || strings.TrimSpace(opts.SearchOverrideExtra) != "" {
		lines = append(lines, SearchCodeOverrideLine(opts.SearchOverrideLabel, opts.SearchOverrideExtra))
	}
	if strings.TrimSpace(opts.ReadOverrideExtra) != "" {
		lines = append(lines, ReadFileOverrideLine(opts.ReadOverrideExtra))
	}
	if opts.IncludeBatchRead {
		lines = append(lines, ReadFileBatchOverrideLine(opts.BatchReadOverrideLabel))
	}
	return strings.Join(lines, "\n")
}

// DedicatedToolUsageSentence は dedicated tools を bash より優先する共有文言を返す。
func DedicatedToolUsageSentence() string {
	return "Always use dedicated tools (gather_context first; lower-level investigation tools only when they are exposed as expert overrides) instead of bash equivalents; tools provide caching, range tracking, and structured output"
}

// NoBashSubstituteSentence は code discovery で bash 代用を禁止する共有文言を返す。
func NoBashSubstituteSentence() string {
	return "During code discovery, do not use bash as a substitute for gather_context or other exposed investigation tools. Repository exploration, symbol lookup, related-test discovery, and dependency tracing must use the dedicated tools first."
}

// InvestigationContextSourceLine returns the shared exact-context rule.
func InvestigationContextSourceLine(allowLowLevelOverrides bool) string {
	if allowLowLevelOverrides {
		return "Use exact context from actual gather_context/read_file/search_code output when constructing edit instructions; never reconstruct it from memory."
	}
	return "Use exact context from actual visible investigation tool output in this session, especially gather_context output; never reconstruct it from memory."
}

// DelegatedInvestigationWaitLine returns the shared no-duplicate-after-delegation rule.
func DelegatedInvestigationWaitLine(allowLowLevelOverrides bool) string {
	if allowLowLevelOverrides {
		return "- After spawning, do NOT use gather_context/search_code/read_file yourself for the same delegated task. Wait for results first."
	}
	return "- After spawning, do NOT repeat the same investigation yourself with gather_context or other visible read-only tools for the same delegated task. Wait for results first."
}

// InvestigationCoverageLine returns the shared re-read coverage rule.
func InvestigationCoverageLine(allowLowLevelOverrides bool) string {
	if allowLowLevelOverrides {
		return "- NEVER re-read a file already returned in full. Avoid re-reading files already covered by gather_context/search_code or earlier read_file calls in this session."
	}
	return "- NEVER re-read a file already returned in full. Avoid re-reading files already covered by gather_context in this session."
}

// CombinedInvestigationQueryLine returns the shared combined-query efficiency rule.
func CombinedInvestigationQueryLine(allowLowLevelOverrides bool) string {
	if allowLowLevelOverrides {
		return "- One gather_context query or one search_code call with comma-separated patterns is better than multiple narrow searches."
	}
	return "- One combined gather_context query is better than multiple narrow investigation queries."
}

// LegacyStrReplaceContextSourceLine returns the shared str_replace provenance rule.
func LegacyStrReplaceContextSourceLine() string {
	return promptBullet("str_replace old_str must come from actual gather_context, read_file, or search_code output in this session.")
}

// LegacyRetryReadLine returns the shared retry-after-read rule.
func LegacyRetryReadLine() string {
	return promptBullet("After str_replace fails, read the target section once, then retry. Do not loop read-fail-read-fail.")
}

// LegacyEditToolRulesBlock は legacy edit tool 用の共有 rules block を返す。
func LegacyEditToolRulesBlock() string {
	return strings.Join([]string{
		promptBullet("Prefer str_replace for partial edits after targeted reads or searches."),
		promptBullet("Use write_file for full-file creation or replacement."),
		promptBullet("Use delete_file only for intentional removals."),
		LegacyStrReplaceContextSourceLine(),
		LegacyRetryReadLine(),
	}, "\n")
}

func promptBullet(text string) string {
	return "- " + strings.TrimSpace(text)
}

func promptBulletWithExtra(core string, extra string) string {
	core = strings.TrimSpace(core)
	extra = strings.TrimSpace(extra)
	if extra == "" {
		return promptBullet(core)
	}
	return promptBullet(core + " " + extra)
}
