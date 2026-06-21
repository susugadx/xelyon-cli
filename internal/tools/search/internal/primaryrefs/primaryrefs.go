package primaryrefs

import (
	"path/filepath"
	"strings"
)

type Ref struct {
	DisplayPath  string
	ResolvedPath string
}

type Source int

const (
	SourceText Source = iota
	SourceStructuredSymbol
	SourceInvocationRelative
)

type Candidate struct {
	DisplayPath string
	Source      Source
}

type Resolver struct {
	Text               func(string) string
	StructuredSymbol   func(string) string
	InvocationRelative func(string) string
}

type resolverState struct {
	resolver Resolver
	seen     map[string]bool
	refs     []Ref
}

type candidateCollector struct {
	candidates []Candidate
}

type lineParser func(line string) (Candidate, bool)

var numberedCandidateKindTokens = []string{
	" function ",
	" method ",
	" type ",
	" interface ",
	" const ",
	" var ",
}

var defaultLineParsers = []lineParser{
	parseTextCandidate,
	parseStructuredHeaderCandidate,
	parseNumberedCandidate,
}

func ExtractPaths(output string) []string {
	refs := ExtractRefs(output, Resolver{})
	paths := make([]string, 0, len(refs))
	for _, ref := range refs {
		paths = append(paths, ref.DisplayPath)
	}
	return paths
}

func ExtractRefs(output string, resolver Resolver) []Ref {
	candidates := CollectCandidates(output)
	return ResolveCandidates(candidates, resolver)
}

func CollectCandidates(output string) []Candidate {
	collector := newCandidateCollector()
	for _, line := range strings.Split(output, "\n") {
		collector.addLine(line)
	}
	return collector.results()
}

func ParseCandidateLine(line string) (Candidate, bool) {
	trimmed := strings.TrimSpace(line)
	for _, parser := range defaultLineParsers {
		if candidate, ok := parser(trimmed); ok {
			return candidate, true
		}
	}
	return Candidate{}, false
}

func parseTextCandidate(line string) (Candidate, bool) {
	if !strings.HasPrefix(line, "📄 ") {
		return Candidate{}, false
	}
	rest := strings.TrimPrefix(line, "📄 ")
	idx := strings.Index(rest, " (")
	if idx <= 0 {
		return Candidate{}, false
	}
	return Candidate{
		DisplayPath: strings.TrimSpace(rest[:idx]),
		Source:      SourceText,
	}, true
}

func parseStructuredHeaderCandidate(line string) (Candidate, bool) {
	if !strings.HasPrefix(line, "── ") || !strings.Contains(line, " in ") || !strings.HasSuffix(line, "──") {
		return Candidate{}, false
	}
	inIdx := strings.LastIndex(line, " in ")
	rest := line[inIdx+4:]
	rest = strings.TrimSuffix(rest, "──")
	return Candidate{
		DisplayPath: trimRenderedPath(rest),
		Source:      SourceStructuredSymbol,
	}, true
}

func parseNumberedCandidate(line string) (Candidate, bool) {
	rest, ok := SplitNumberedListLinePrefix(line)
	if !ok {
		return Candidate{}, false
	}
	if numbered, ok := ParseNumberedCandidateFilePathWithPolicy(rest, numberedCandidateKindTokens); ok {
		return Candidate{
			DisplayPath: trimRenderedPath(numbered),
			Source:      SourceStructuredSymbol,
		}, true
	}
	if idx := strings.LastIndex(rest, " in "); idx > 0 {
		return Candidate{
			DisplayPath: trimRenderedPath(rest[idx+4:]),
			Source:      SourceInvocationRelative,
		}, true
	}
	return Candidate{}, false
}

func newCandidateCollector() *candidateCollector {
	return &candidateCollector{
		candidates: make([]Candidate, 0),
	}
}

func (collector *candidateCollector) addLine(line string) {
	candidate, ok := ParseCandidateLine(line)
	if !ok {
		return
	}
	collector.addCandidate(candidate)
}

func (collector *candidateCollector) addCandidate(candidate Candidate) {
	candidate.DisplayPath = strings.TrimSpace(candidate.DisplayPath)
	if candidate.DisplayPath == "" {
		return
	}
	collector.candidates = append(collector.candidates, candidate)
}

func (collector *candidateCollector) results() []Candidate {
	return collector.candidates
}

func ResolveCandidates(candidates []Candidate, resolver Resolver) []Ref {
	state := newResolverState(resolver)
	for _, candidate := range candidates {
		state.addCandidate(candidate)
	}
	return state.results()
}

func newResolverState(resolver Resolver) *resolverState {
	return &resolverState{
		resolver: resolver,
		seen:     make(map[string]bool),
	}
}

func (state *resolverState) addCandidate(candidate Candidate) {
	ref, ok := state.resolveCandidate(candidate)
	if !ok {
		return
	}
	key := ref.DisplayPath + "\x00" + ref.ResolvedPath
	if state.seen[key] {
		return
	}
	state.seen[key] = true
	state.refs = append(state.refs, ref)
}

func (state *resolverState) resolveCandidate(candidate Candidate) (Ref, bool) {
	displayPath := strings.TrimSpace(candidate.DisplayPath)
	if displayPath == "" {
		return Ref{}, false
	}
	return Ref{
		DisplayPath:  displayPath,
		ResolvedPath: cleanResolvedPath(state.resolvePath(displayPath, candidate.Source)),
	}, true
}

func (state *resolverState) resolvePath(displayPath string, source Source) string {
	switch source {
	case SourceStructuredSymbol:
		if state.resolver.StructuredSymbol != nil {
			return state.resolver.StructuredSymbol(displayPath)
		}
	case SourceInvocationRelative:
		if state.resolver.InvocationRelative != nil {
			return state.resolver.InvocationRelative(displayPath)
		}
	default:
		if state.resolver.Text != nil {
			return state.resolver.Text(displayPath)
		}
	}
	return ""
}

func (state *resolverState) results() []Ref {
	return state.refs
}

func HasNumericListPrefix(line string) bool {
	_, ok := SplitNumberedListLinePrefix(line)
	return ok
}

func SplitNumberedListLinePrefix(line string) (string, bool) {
	line = strings.TrimSpace(line)
	if line == "" {
		return "", false
	}
	dotIdx := strings.Index(line, ".")
	if dotIdx <= 0 {
		return "", false
	}
	for _, r := range line[:dotIdx] {
		if r < '0' || r > '9' {
			return "", false
		}
	}
	rest := strings.TrimSpace(line[dotIdx+1:])
	if rest == "" {
		return "", false
	}
	return rest, true
}

func ParseNumberedCandidateFilePath(line string) (string, bool) {
	rest, ok := SplitNumberedListLinePrefix(line)
	if !ok {
		return "", false
	}
	return ParseNumberedCandidateFilePathWithPolicy(rest, numberedCandidateKindTokens)
}

func ParseNumberedCandidateFilePathWithPolicy(rest string, kindTokens []string) (string, bool) {
	for _, token := range kindTokens {
		if idx := strings.Index(rest, token); idx > 0 {
			return strings.TrimSpace(rest[:idx]), true
		}
	}
	return "", false
}

func cleanResolvedPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	if abs, err := filepath.Abs(path); err == nil {
		path = abs
	}
	return filepath.Clean(path)
}

func trimRenderedPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	if atIdx := strings.LastIndex(path, " @"); atIdx > 0 {
		path = strings.TrimSpace(path[:atIdx])
	}
	for {
		lIdx := strings.LastIndex(path, " [L")
		if lIdx <= 0 || !strings.HasSuffix(path, "]") {
			break
		}
		path = strings.TrimSpace(path[:lIdx])
	}
	return path
}
