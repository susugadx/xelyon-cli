package repomap

// Generate はプロジェクト構造マップを文字列に変換する。
func (pm *ProjectMap) Generate() string {
	if pm == nil {
		return ""
	}
	if len(pm.Files) == 0 && len(pm.GitStatus) == 0 {
		return ""
	}
	return newProjectMapBudgetReducer(pm).reduce()
}

// GenerateManifest は root manifest 寄りの軽量な Project Map を生成する。
func (pm *ProjectMap) GenerateManifest(prioritizedPaths []string) string {
	if pm == nil || len(pm.Files) == 0 {
		return ""
	}

	topDirs, topFiles := pm.collectTopLevelEntries()
	priorityFiles := pm.collectPriorityFiles(prioritizedPaths)
	return newProjectMapManifestBudgetReducer(pm, topDirs, topFiles, priorityFiles).reduce()
}
