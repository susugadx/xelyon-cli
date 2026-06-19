package toolmeta

import "sort"

// SafetyLevel はツールの安全性レベル。
type SafetyLevel int

const (
	SafetyHigh SafetyLevel = iota
	SafetyMedium
	SafetyLow
)

// Spec はビルトインツールの表示・安全性メタデータ。
type Spec struct {
	Name        string
	Description string
	Safety      SafetyLevel
	HelpSummary string
	HelpOrder   int
}

var builtinSpecs = []Spec{
	{
		Name:        "gather_context",
		Description: "Primary repository investigation tool. Use it to retrieve the files, ranges, symbols, callers, tests, or directory context needed for the next decision. Prefer it when the target is not already available as exact content.",
		Safety:      SafetyHigh,
		HelpOrder:   10,
	},
	{
		Name:        "read_file",
		Description: "Read exact content from known files or line ranges. Use when precise text is required for an edit or verification; avoid repeating content already available in the current context.",
		Safety:      SafetyHigh,
		HelpOrder:   30,
	},
	{
		Name:        "write_file",
		Description: "Create or overwrite a file. Uses 0644 for new files and preserves permissions on overwrite. Prefer the primary edit tool for partial edits to existing files.",
		Safety:      SafetyMedium,
		HelpSummary: "Legacy edit tool to create or overwrite a file",
		HelpOrder:   110,
	},
	{
		Name:        "str_replace",
		Description: "Precise edits to existing files. Copy exact old_str and existing context from actual gather_context, read_file, or search_code output; do not invent old_str. Write new_str as the intended replacement based on that verified context. For multiple edits in the same file, prefer same-file edits=[{old_str,new_str},...] based on actual content. Line-range mode (old_str empty + start_line/end_line) is an advanced fallback only when fresh tool output provides an exact range; do not guess line ranges or use it to avoid evidence.",
		Safety:      SafetyMedium,
		HelpSummary: "Precise same-file replacements from actual tool output",
		HelpOrder:   100,
	},
	{
		Name:        "apply_patch",
		Description: "Primary edit tool for precise patch-based file changes.",
		Safety:      SafetyLow,
		HelpSummary: "Primary edit tool for precise patch-based file changes",
		HelpOrder:   105,
	},
	{
		Name:        "delete_file",
		Description: "Delete a file permanently.",
		Safety:      SafetyLow,
		HelpSummary: "Legacy edit tool to delete a file",
		HelpOrder:   120,
	},
	{
		Name:        "list_dir",
		Description: "Low-level directory summary tool. gather_context is the default front door for directory investigation and direct routing. list_dir stays hidden on current gather_context-first agent surfaces; when directly exposed, treat it as a low-level/internal override. Returns a compact summary with representative names, counts, and types. Ignores .git, node_modules, vendor by default. Use depth parameter (1-3) for recursive listing.",
		Safety:      SafetyHigh,
		HelpOrder:   40,
	},
	{
		Name:        "search_code",
		Description: "Low-level repository search for exact symbols, literals, regex patterns, references, or impact analysis. Use when gather_context is insufficient or exact search control is needed. Combine related independent patterns when practical.",
		Safety:      SafetyHigh,
		HelpOrder:   20,
	},
	{
		Name:        "web_search",
		Description: "Search the web and return summarized findings plus source URLs, not full page contents. For deeper coverage, run multiple targeted searches with narrower queries.",
		Safety:      SafetyMedium,
		HelpSummary: "Search the web and return summarized findings with source URLs",
		HelpOrder:   50,
	},
	{
		Name:        "bash",
		Description: "Run local commands for build, test, format, lint, git, package tooling, and tasks without a dedicated tool. Runtime policy determines approval or denial. Prefer dedicated repository tools for reading and searching when available, but use shell fallbacks when they are unavailable or insufficient.",
		Safety:      SafetyLow,
		HelpSummary: "Execute shell commands for build, test, git, and local tooling",
		HelpOrder:   130,
	},
	{
		Name:        "ask_user_question",
		Description: "Ask the user a clarification question before planning. Use only when requirements are ambiguous.",
		Safety:      SafetyHigh,
		HelpSummary: "Ask the user a clarification question during plan investigation",
		HelpOrder:   80,
	},
	{
		Name:        "spawn_agent",
		Description: "Start a bounded specialist task for exploration, implementation, review, or verification. Provide the objective, relevant scope, constraints, and expected deliverable. The parent agent remains responsible for integration and final decisions.",
		Safety:      SafetyHigh,
		HelpSummary: "Spawn a sub-agent for explore/edit/verify tasks",
		HelpOrder:   140,
	},
	{
		Name:        "wait_agent",
		Description: "Collect results from one or more spawned agents. Respect the configured timeout and concurrency policy.",
		Safety:      SafetyHigh,
		HelpSummary: "Wait for sub-agents to complete",
		HelpOrder:   150,
	},
	{
		Name:        "activate_skill",
		Description: "Load one discovered Agent Skill by name and return a JSON payload with name, skill_directory, scripts/references/assets, and skill_md. This is read-only and does not execute scripts automatically.",
		Safety:      SafetyHigh,
		HelpSummary: "Load full SKILL.md content for one discovered skill on demand",
		HelpOrder:   60,
	},
	{
		Name:        "run_skill_script",
		Description: "Resolve and execute one script under a discovered skill's scripts/ directory. Uses existing bash safety/confirmation/deny policy and does not bypass execution controls.",
		Safety:      SafetyLow,
		HelpSummary: "Run one whitelisted skill script via existing bash safety and confirmation policy",
		HelpOrder:   70,
	},
}

var specsByName map[string]Spec

func init() {
	specsByName = make(map[string]Spec, len(builtinSpecs))
	for _, spec := range builtinSpecs {
		specsByName[spec.Name] = spec
	}
}

// BuiltinSpecs はビルトインツール仕様のスナップショットを返す。
func BuiltinSpecs() []Spec {
	result := make([]Spec, len(builtinSpecs))
	copy(result, builtinSpecs)
	return result
}

// Lookup はツール名から仕様を返す。
func Lookup(name string) (Spec, bool) {
	spec, ok := specsByName[name]
	return spec, ok
}

// DescriptionMap は description マップのコピーを返す。
func DescriptionMap() map[string]string {
	out := make(map[string]string, len(builtinSpecs))
	for _, spec := range builtinSpecs {
		out[spec.Name] = spec.Description
	}
	return out
}

// HelpDisplayOrder は help 表示順を返す。
func HelpDisplayOrder() []string {
	ordered := BuiltinSpecs()
	sort.SliceStable(ordered, func(i, j int) bool {
		if ordered[i].HelpOrder == ordered[j].HelpOrder {
			return ordered[i].Name < ordered[j].Name
		}
		return ordered[i].HelpOrder < ordered[j].HelpOrder
	})
	names := make([]string, 0, len(ordered))
	for _, spec := range ordered {
		names = append(names, spec.Name)
	}
	return names
}

// HelpSummary は help summary を返す。
func HelpSummary(name string) (string, bool) {
	spec, ok := Lookup(name)
	if !ok {
		return "", false
	}
	if spec.HelpSummary == "" {
		return "", false
	}
	return spec.HelpSummary, true
}

// SafetyByName は safety を返す。
func SafetyByName(name string) (SafetyLevel, bool) {
	spec, ok := Lookup(name)
	if !ok {
		return SafetyMedium, false
	}
	return spec.Safety, true
}
