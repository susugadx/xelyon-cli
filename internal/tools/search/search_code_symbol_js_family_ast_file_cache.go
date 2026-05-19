package search

import (
	"path/filepath"

	"github.com/susugadx/xelyon-cli/internal/jsast"
)

type jsFamilyASTParsedFileCache struct {
	files map[string]*jsast.ParsedFile
}

func newJSFamilyASTParsedFileCache() *jsFamilyASTParsedFileCache {
	return &jsFamilyASTParsedFileCache{
		files: make(map[string]*jsast.ParsedFile),
	}
}

func (c *jsFamilyASTParsedFileCache) Parsed(absPath string) *jsast.ParsedFile {
	if c == nil {
		return nil
	}
	if c.files == nil {
		c.files = make(map[string]*jsast.ParsedFile)
	}
	key := filepath.Clean(absPath)
	if parsed, ok := c.files[key]; ok {
		return parsed
	}

	parsed, ok := parseJSFamilyFileForSearch(key)
	if !ok {
		c.files[key] = nil
		return nil
	}
	c.files[key] = parsed
	return parsed
}

func (c *jsFamilyASTParsedFileCache) Close() {
	if c == nil {
		return
	}
	for _, parsed := range c.files {
		if parsed != nil {
			parsed.Close()
		}
	}
}
