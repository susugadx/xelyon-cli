package search

import (
	"os"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/jsast"
	"github.com/susugadx/xelyon-cli/internal/repomap"
)

type jsFamilyLSPReferenceBuilder struct {
	symbol   string
	files    map[string]*jsFamilyLSPReferenceFile
	loadFile func(absPath string) *jsFamilyLSPReferenceFile
}

type jsFamilyLSPReferenceFile struct {
	src    []byte
	lines  []string
	parsed *jsast.ParsedFile
}

func newJSFamilyLSPReferenceBuilder(symbol string) *jsFamilyLSPReferenceBuilder {
	return &jsFamilyLSPReferenceBuilder{
		symbol:   symbol,
		files:    make(map[string]*jsFamilyLSPReferenceFile),
		loadFile: loadJSFamilyLSPReferenceFile,
	}
}

func loadJSFamilyLSPReferenceFile(absPath string) *jsFamilyLSPReferenceFile {
	src, err := os.ReadFile(absPath)
	if err != nil {
		return &jsFamilyLSPReferenceFile{}
	}
	file := &jsFamilyLSPReferenceFile{
		src: src,
	}
	if parsed, err := jsast.ParseBytes(absPath, src); err == nil {
		file.parsed = parsed
	}
	return file
}

func (b *jsFamilyLSPReferenceBuilder) SummaryRef(candidate jsFamilyLSPReferenceCandidate) genericSymbolRef {
	file := b.file(candidate.absPath)
	loc := candidate.loc
	ref := genericSymbolRef{
		File:   candidate.displayPath,
		Line:   loc.Line,
		IsTest: repomap.IsTestFile(candidate.displayPath),
	}
	if file.parsed != nil {
		if info, err := jsast.ClassifyRangeWithParsed(file.parsed, loc.Line, loc.Character, loc.EndLine, loc.EndChar, b.symbol); err == nil && info != nil {
			ref.Class = info.Class
		}
	}
	return ref
}

func (b *jsFamilyLSPReferenceBuilder) Ref(candidate jsFamilyLSPReferenceCandidate) genericSymbolRef {
	return b.RefWithSummary(candidate, b.SummaryRef(candidate))
}

func (b *jsFamilyLSPReferenceBuilder) RefWithSummary(candidate jsFamilyLSPReferenceCandidate, summary genericSymbolRef) genericSymbolRef {
	file := b.file(candidate.absPath)
	ref := summary
	ref.Snippet = file.snippet(candidate.loc.Line)
	return ref
}

func (b *jsFamilyLSPReferenceBuilder) file(absPath string) *jsFamilyLSPReferenceFile {
	if b.files == nil {
		b.files = make(map[string]*jsFamilyLSPReferenceFile)
	}
	if file, ok := b.files[absPath]; ok {
		return file
	}
	if b.loadFile == nil {
		b.loadFile = loadJSFamilyLSPReferenceFile
	}
	file := b.loadFile(absPath)
	if file == nil {
		file = &jsFamilyLSPReferenceFile{}
	}
	b.files[absPath] = file
	return file
}

func (b *jsFamilyLSPReferenceBuilder) Close() {
	for _, file := range b.files {
		if file != nil && file.parsed != nil {
			file.parsed.Close()
		}
	}
}

func (file *jsFamilyLSPReferenceFile) snippet(line int) string {
	if file == nil || line <= 0 {
		return ""
	}
	if file.lines == nil && file.src != nil {
		file.lines = strings.Split(string(file.src), "\n")
	}
	if line > len(file.lines) {
		return ""
	}
	return strings.TrimSpace(file.lines[line-1])
}
