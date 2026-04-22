package repomap

// Build はプロジェクトのファイル一覧とシンボル一覧を構築する。
func (pm *ProjectMap) Build() error {
	result, err := pm.runBuild(buildModeFull)
	if err != nil {
		return err
	}

	pm.applyBuildResult(result.entries)
	_ = saveMapCache(pm.RootPath, result.cache)
	return nil
}

// BuildManifest は軽量な manifest 用にファイル一覧と git status のみを構築する。
func (pm *ProjectMap) BuildManifest() error {
	result, err := pm.runBuild(buildModeManifest)
	if err != nil {
		return err
	}

	pm.applyBuildResult(result.entries)
	return nil
}
