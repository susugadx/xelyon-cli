package search

import (
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/susugadx/xelyon-cli/internal/ast"
)

type goMethodCallMatchContext struct {
	targetPackageDir     string
	receiver             string
	methodName           string
	currentPath          string
	currentSrc           []byte
	opts                 SearchOptions
	requireReceiver      bool
	alternativeReceivers map[string]struct{}
	localTypeNames       map[string]struct{}
	dependencies         *structuredGoImpactProbeDeps
}

type methodCallProbeLineStat struct {
	Count            int
	HasSelectorChain bool
	SelectorTail     string
}

func (ctx goMethodTestProbeContext) matchContext(currentPath string, currentSrc []byte) goMethodCallMatchContext {
	return goMethodCallMatchContext{
		targetPackageDir:     ctx.targetPackageDir,
		receiver:             ctx.receiver,
		methodName:           ctx.symbol.Name,
		currentPath:          currentPath,
		currentSrc:           currentSrc,
		opts:                 ctx.opts,
		requireReceiver:      ctx.requireReceiver,
		alternativeReceivers: ctx.alternativeReceivers,
		localTypeNames:       ctx.localTypeNames,
		dependencies:         ctx.dependencies,
	}
}

func methodTestBodyMatchesSymbol(ctx goMethodCallMatchContext, parsed *ast.ParsedFile, test ast.Symbol, allowNameProbeOnly bool) bool {
	if strings.TrimSpace(ctx.methodName) == "" {
		return false
	}
	lines := strings.Split(string(ctx.currentSrc), "\n")
	start := test.Line - 1
	if start < 0 {
		start = 0
	}
	end := min(len(lines), test.EndLine)
	if start >= end {
		return false
	}

	body := strings.Join(lines[start:end], "\n")
	callLines := findMethodCallProbeLineStats(body, ctx.methodName, test.Line)
	if len(callLines) == 0 {
		return allowNameProbeOnly
	}

	sawDirectMismatch := false
	sawAmbiguous := false
	for line, stat := range callLines {
		if stat.Count > 1 {
			sawAmbiguous = true
			continue
		}
		info, err := ast.ClassifyLineWithParsed(parsed, line, ctx.methodName)
		if err != nil || info == nil {
			sawAmbiguous = true
			continue
		}
		if info.Class != ast.ClassCall {
			sawAmbiguous = true
			continue
		}
		if info.SelectorKind == "package" {
			sawDirectMismatch = true
			continue
		}
		if info.SelectorKind != "method" && info.NodeType != "field_identifier" {
			sawAmbiguous = true
			continue
		}

		candidateReceiver := canonicalProbeReceiver(info.ReceiverType)
		if ctx.receiver != "" && candidateReceiver == ctx.receiver {
			return true
		}
		if candidateReceiver == "" {
			if ctx.requireReceiver && stat.HasSelectorChain {
				tail := strings.TrimSpace(stat.SelectorTail)
				if tail == "" {
					sawAmbiguous = true
					continue
				}
				if tail == ctx.receiver {
					return true
				}
				if isLikelyWrapperSelectorTail(tail) {
					sawAmbiguous = true
					continue
				}
				if _, ok := ctx.localTypeNames[tail]; ok {
					sawDirectMismatch = true
					continue
				}
				if _, ok := ctx.alternativeReceivers[tail]; ok {
					sawDirectMismatch = true
					continue
				}
				sawDirectMismatch = true
				continue
			}
			sawAmbiguous = true
			continue
		}

		if !ctx.requireReceiver {
			if probeReceiverIsQualified(info.ReceiverType) {
				qctx := newQualifiedReceiverProbeContextWithDeps(info.ReceiverType, ctx.currentPath, ctx.currentSrc, ctx.opts, ctx.dependencies)
				switch qctx.role() {
				case methodProbeReceiverRoleInterface, methodProbeReceiverRoleUnknown:
					sawAmbiguous = true
					continue
				default:
					if qctx.hasDirectMethod(ctx.methodName) {
						sawDirectMismatch = true
						continue
					}
					sawAmbiguous = true
					continue
				}
			}
			if _, ok := ctx.localTypeNames[candidateReceiver]; ok {
				sawAmbiguous = true
				continue
			}
			sawDirectMismatch = true
			continue
		}

		if _, ok := ctx.alternativeReceivers[candidateReceiver]; !ok {
			sawAmbiguous = true
			continue
		}
		if probeReceiverIsQualified(info.ReceiverType) {
			qctx := newQualifiedReceiverProbeContextWithDeps(info.ReceiverType, ctx.currentPath, ctx.currentSrc, ctx.opts, ctx.dependencies)
			switch qctx.role() {
			case methodProbeReceiverRoleInterface, methodProbeReceiverRoleUnknown:
				sawAmbiguous = true
				continue
			default:
				if !qctx.hasDirectMethod(ctx.methodName) {
					sawAmbiguous = true
					continue
				}
			}
		}
		sawDirectMismatch = true
	}

	if sawAmbiguous {
		return true
	}
	return !sawDirectMismatch
}

func findMethodCallProbeLineStats(body, methodName string, startLine int) map[int]methodCallProbeLineStat {
	methodName = strings.TrimSpace(methodName)
	if body == "" || methodName == "" || startLine <= 0 {
		return nil
	}

	re := regexp.MustCompile(`\.\s*(` + regexp.QuoteMeta(methodName) + `)\s*\(`)
	matches := re.FindAllStringSubmatchIndex(body, -1)
	if len(matches) == 0 {
		return nil
	}

	lines := make(map[int]methodCallProbeLineStat, len(matches))
	for _, match := range matches {
		if len(match) < 4 {
			continue
		}
		line := startLine + strings.Count(body[:match[2]], "\n")
		stat := lines[line]
		stat.Count++
		if methodCallHasSelectorChain(body, match[0]) {
			stat.HasSelectorChain = true
			stat.SelectorTail = methodCallSelectorChainTail(body, match[0])
		}
		lines[line] = stat
	}
	return lines
}

func methodCallSelectorChainTail(body string, matchStart int) string {
	if matchStart <= 0 || matchStart > len(body) {
		return ""
	}

	i := matchStart - 1
	for i >= 0 {
		r, size := utf8.DecodeLastRuneInString(body[:i+1])
		if unicode.IsSpace(r) {
			i -= size
			continue
		}
		break
	}
	if i < 0 {
		return ""
	}

	end := i + 1
	for i >= 0 {
		r, size := utf8.DecodeLastRuneInString(body[:i+1])
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' {
			i -= size
			continue
		}
		break
	}
	start := i + 1
	if start >= end {
		return ""
	}
	return strings.TrimSpace(body[start:end])
}

func isLikelyWrapperSelectorTail(tail string) bool {
	tail = strings.TrimSpace(tail)
	if tail == "" {
		return false
	}
	r, _ := utf8.DecodeRuneInString(tail)
	return unicode.IsLower(r)
}

func methodCallHasSelectorChain(body string, matchStart int) bool {
	if matchStart <= 0 || matchStart > len(body) {
		return false
	}

	for i := matchStart - 1; i >= 0; {
		r, size := utf8.DecodeLastRuneInString(body[:i+1])
		if unicode.IsSpace(r) {
			i -= size
			continue
		}
		switch {
		case r == '.':
			return true
		case unicode.IsLetter(r), unicode.IsDigit(r), r == '_', r == '*', r == ')', r == '(', r == ']', r == '[', r == '}', r == '{':
			i -= size
			continue
		default:
			return false
		}
	}
	return false
}
