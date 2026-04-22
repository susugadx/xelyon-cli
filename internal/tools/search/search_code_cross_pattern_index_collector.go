package search

type crossPatternIndexEntry struct {
	ref          primaryFileRef
	patternCount int
	category     string
}

type crossPatternIndexSections struct {
	implKeys   []string
	testKeys   []string
	configKeys []string
}

type crossPatternIndexCollector struct {
	fileMap map[string]*crossPatternIndexEntry
	order   []string
}

type crossPatternIndexData struct {
	fileMap    map[string]*crossPatternIndexEntry
	order      []string
	sections   crossPatternIndexSections
	hasHotspot bool
}

func newCrossPatternIndexData(collector *crossPatternIndexCollector) crossPatternIndexData {
	fileMap, order := collector.results()
	sections, hasHotspot := summarizeCrossPatternIndex(fileMap, order)
	return crossPatternIndexData{
		fileMap:    fileMap,
		order:      order,
		sections:   sections,
		hasHotspot: hasHotspot,
	}
}

func (d crossPatternIndexData) isEmpty() bool {
	return len(d.order) == 0
}

func crossPatternIndexEntryKey(ref primaryFileRef) string {
	return ref.DisplayPath + "\x00" + ref.ResolvedPath
}

func newCrossPatternIndexCollector() *crossPatternIndexCollector {
	return &crossPatternIndexCollector{
		fileMap: make(map[string]*crossPatternIndexEntry),
	}
}

func (collector *crossPatternIndexCollector) addRef(ref primaryFileRef) {
	key := crossPatternIndexEntryKey(ref)
	if entry, ok := collector.fileMap[key]; ok {
		entry.patternCount++
		return
	}
	collector.fileMap[key] = &crossPatternIndexEntry{
		ref:          ref,
		patternCount: 1,
		category:     classifyFilePath(ref.DisplayPath),
	}
	collector.order = append(collector.order, key)
}

func (collector *crossPatternIndexCollector) results() (map[string]*crossPatternIndexEntry, []string) {
	return collector.fileMap, collector.order
}

func summarizeCrossPatternIndex(fileMap map[string]*crossPatternIndexEntry, order []string) (crossPatternIndexSections, bool) {
	sections := crossPatternIndexSections{}
	hasHotspot := false
	for _, key := range order {
		entry := fileMap[key]
		if entry.patternCount > 1 {
			hasHotspot = true
		}
		switch entry.category {
		case "test":
			sections.testKeys = append(sections.testKeys, key)
		case "config":
			sections.configKeys = append(sections.configKeys, key)
		default:
			sections.implKeys = append(sections.implKeys, key)
		}
	}
	return sections, hasHotspot
}
