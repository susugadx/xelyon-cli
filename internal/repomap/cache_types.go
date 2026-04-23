package repomap

import "time"

// MapCache はプロジェクトマップのキャッシュ。
type MapCache struct {
	RootPath  string                `json:"root_path"`
	UpdatedAt time.Time             `json:"updated_at"`
	Files     map[string]*CacheFile `json:"files"`
}

// CacheFile はファイルごとのキャッシュ。
type CacheFile struct {
	ModTime   time.Time `json:"mod_time"`
	LineCount int       `json:"line_count"`
	Symbols   []Symbol  `json:"symbols,omitempty"`
}

func cloneCacheFile(file *CacheFile) *CacheFile {
	if file == nil {
		return nil
	}
	cloned := &CacheFile{
		ModTime:   file.ModTime,
		LineCount: file.LineCount,
	}
	if len(file.Symbols) > 0 {
		cloned.Symbols = append([]Symbol(nil), file.Symbols...)
	}
	return cloned
}
