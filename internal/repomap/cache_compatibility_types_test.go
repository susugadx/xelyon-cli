package repomap

import "time"

type oldSymbol struct {
	Line      int    `json:"line"`
	Signature string `json:"signature"`
}

type oldCacheFile struct {
	ModTime   time.Time   `json:"mod_time"`
	LineCount int         `json:"line_count"`
	Symbols   []oldSymbol `json:"symbols"`
}

type oldMapCache struct {
	RootPath  string                   `json:"root_path"`
	UpdatedAt time.Time                `json:"updated_at"`
	Files     map[string]*oldCacheFile `json:"files"`
}
