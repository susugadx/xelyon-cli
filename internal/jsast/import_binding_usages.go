package jsast

import (
	"sort"
	"strings"

	"github.com/odvcencio/gotreesitter"
	codeast "github.com/susugadx/xelyon-cli/internal/ast"
)

// ImportBindingUsage は import binding の local name を参照する usage を表す。
type ImportBindingUsage struct {
	Name    string
	Line    int
	Snippet string
	Class   codeast.MatchClass
}

// ImportBindingUsageVisitor は import binding usage の visitor を表す。
type ImportBindingUsageVisitor func(ImportBindingUsage) bool

// ImportBindingUsagesWithParsed は import binding の local usage を抽出する。
func ImportBindingUsagesWithParsed(parsed *ParsedFile, binding ImportBinding) []ImportBindingUsage {
	var usages []ImportBindingUsage
	VisitImportBindingUsagesWithParsed(parsed, binding, func(usage ImportBindingUsage) bool {
		usages = append(usages, usage)
		return true
	})
	return usages
}

// VisitImportBindingUsagesWithParsed は import binding の local usage を走査する。
func VisitImportBindingUsagesWithParsed(parsed *ParsedFile, binding ImportBinding, visit ImportBindingUsageVisitor) {
	if parsed == nil || parsed.tree == nil || visit == nil {
		return
	}
	localName := strings.TrimSpace(binding.Local)
	if localName == "" {
		return
	}

	root := parsed.tree.RootNode()
	shadowScopes := importBindingShadowScopes(parsed, binding)
	candidatesByLine := make(map[int]importBindingUsageCandidate)
	walkNamed(root, func(node *gotreesitter.Node) {
		if !importBindingLocalNameNode(parsed, node, localName) {
			return
		}
		if node.StartByte() == binding.localStartByte && node.EndByte() == binding.localEndByte {
			return
		}
		if hasAncestorKind(parsed, node, "import_statement") {
			return
		}
		if jsxLocalNameShadowedByScopes(node, shadowScopes) {
			return
		}

		class := classifyNode(parsed, root, node, node.StartByte(), node.EndByte(), localName)
		if !importBindingUsageClassVisible(class) {
			return
		}
		line := int(node.StartPoint().Row) + 1
		candidate := importBindingUsageCandidate{
			usage: ImportBindingUsage{
				Name:    localName,
				Line:    line,
				Snippet: parsedLineSnippet(parsed, line),
				Class:   class,
			},
			startByte: node.StartByte(),
		}
		if existing, ok := candidatesByLine[line]; ok && importBindingUsageCandidateBetter(existing, candidate) {
			return
		}
		candidatesByLine[line] = candidate
	})

	candidates := make([]importBindingUsageCandidate, 0, len(candidatesByLine))
	for _, candidate := range candidatesByLine {
		candidates = append(candidates, candidate)
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].usage.Line != candidates[j].usage.Line {
			return candidates[i].usage.Line < candidates[j].usage.Line
		}
		return candidates[i].startByte < candidates[j].startByte
	})
	for _, candidate := range candidates {
		if !visit(candidate.usage) {
			return
		}
	}
}

type importBindingUsageCandidate struct {
	usage     ImportBindingUsage
	startByte uint32
}

func importBindingUsageCandidateBetter(existing importBindingUsageCandidate, next importBindingUsageCandidate) bool {
	existingPriority := matchClassPriority(existing.usage.Class)
	nextPriority := matchClassPriority(next.usage.Class)
	if existingPriority != nextPriority {
		return existingPriority > nextPriority
	}
	return existing.startByte <= next.startByte
}

func importBindingLocalNameNode(parsed *ParsedFile, node *gotreesitter.Node, localName string) bool {
	switch nodeKind(parsed, node) {
	case "identifier", "shorthand_property_identifier", "type_identifier":
	default:
		return false
	}
	return strings.TrimSpace(nodeText(parsed, node)) == localName
}

func importBindingUsageClassVisible(class codeast.MatchClass) bool {
	switch class {
	case codeast.ClassUnknown, codeast.ClassDef, codeast.ClassImport, codeast.ClassString, codeast.ClassComment, ClassIgnored:
		return false
	default:
		return true
	}
}

func importBindingShadowScopes(parsed *ParsedFile, binding ImportBinding) []localBindingScopeRange {
	return localBindingShadowScopes(parsed, binding.Local, binding.localStartByte, binding.localEndByte)
}
