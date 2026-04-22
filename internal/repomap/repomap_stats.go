package repomap

// GetSymbolCount は保持しているシンボル数を返す。
func (pm *ProjectMap) GetSymbolCount() int {
	if pm == nil {
		return 0
	}
	total := 0
	for _, file := range pm.Files {
		total += len(file.Symbols)
	}
	return total
}

// GetFileCount は保持しているファイル数を返す。
func (pm *ProjectMap) GetFileCount() int {
	if pm == nil {
		return 0
	}
	return len(pm.Files)
}
