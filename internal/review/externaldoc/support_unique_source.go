package externaldoc

type externalSupportUniqueSourceCounter struct {
	keyToSource map[string]int
	parents     []int
}

func newExternalSupportUniqueSourceCounter() *externalSupportUniqueSourceCounter {
	return &externalSupportUniqueSourceCounter{
		keyToSource: make(map[string]int),
	}
}

func (c *externalSupportUniqueSourceCounter) add(keys []string) {
	keys = externalSupportUniqueKeys(keys)
	if len(keys) == 0 {
		return
	}

	source := -1
	for _, key := range keys {
		if existing, ok := c.keyToSource[key]; ok {
			root := c.find(existing)
			if source == -1 {
				source = root
				continue
			}
			source = c.union(source, root)
		}
	}
	if source == -1 {
		source = len(c.parents)
		c.parents = append(c.parents, source)
	}
	for _, key := range keys {
		c.keyToSource[key] = source
	}
}

func (c *externalSupportUniqueSourceCounter) count() int {
	roots := make(map[int]struct{}, len(c.parents))
	for index := range c.parents {
		roots[c.find(index)] = struct{}{}
	}
	return len(roots)
}

func (c *externalSupportUniqueSourceCounter) find(index int) int {
	for c.parents[index] != index {
		c.parents[index] = c.parents[c.parents[index]]
		index = c.parents[index]
	}
	return index
}

func (c *externalSupportUniqueSourceCounter) union(left, right int) int {
	leftRoot := c.find(left)
	rightRoot := c.find(right)
	if leftRoot == rightRoot {
		return leftRoot
	}
	c.parents[rightRoot] = leftRoot
	return leftRoot
}

func externalSupportUniqueSourceKeys(doc Evidence, snippetHashes []string) []string {
	keys := []string{}
	if normalizedURL := externalSupportNormalizedURL(doc.URL); normalizedURL != "" {
		keys = append(keys, "url:"+normalizedURL)
	}
	if contentHash := externalSupportNormalizedHash(doc.ContentHash); contentHash != "" {
		return append(keys, "doc_hash:"+contentHash)
	}
	for _, hash := range snippetHashes {
		if normalizedHash := externalSupportNormalizedHash(hash); normalizedHash != "" {
			keys = append(keys, "snippet_hash:"+normalizedHash)
		}
	}
	return keys
}
