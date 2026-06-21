package evidence

func (b *reviewGenericImpactCandidateBuilder) collectPathCandidates() {
	b.collectSameStemTestOrSpecCandidates()
	b.collectNearbyTestCandidates()
	b.collectNearbyProjectConfigCandidates()
}

func (b *reviewGenericImpactCandidateBuilder) collectSameStemTestOrSpecCandidates() {
	stems := reviewGenericImpactStringSet(b.changedStems)
	for _, path := range b.repoPaths {
		if b.isChangedPath(path) || !isReviewGenericImpactTestOrSpecPath(path) {
			continue
		}
		stem := reviewGenericImpactPathStem(path)
		if _, ok := stems[stem]; !ok {
			continue
		}
		b.addCandidate(ReviewGenericImpactCandidate{
			Path:   path,
			Role:   ReviewGenericImpactRoleSameStemTestOrSpec,
			Reason: "test/spec file shares changed path stem",
			Token:  stem,
		})
	}
}

func (b *reviewGenericImpactCandidateBuilder) collectNearbyTestCandidates() {
	for _, path := range b.repoPaths {
		if b.isChangedPath(path) || !isReviewGenericImpactNearbyTestPath(path) {
			continue
		}
		if !b.isNearChangedDir(path) {
			continue
		}
		b.addCandidate(ReviewGenericImpactCandidate{
			Path:   path,
			Role:   ReviewGenericImpactRoleNearbyTestOrTestsDir,
			Reason: "test/spec file is in the same directory or a nearby test directory",
			Token:  reviewGenericImpactPathStem(path),
		})
	}
}

func (b *reviewGenericImpactCandidateBuilder) collectNearbyProjectConfigCandidates() {
	ancestorDirs := reviewGenericImpactAncestorDirSet(b.changedDirs)
	for _, path := range b.repoPaths {
		if b.isChangedPath(path) || !isReviewGenericImpactProjectConfigPath(path) {
			continue
		}
		if _, ok := ancestorDirs[reviewGenericImpactPathDir(path)]; !ok {
			continue
		}
		b.addCandidate(ReviewGenericImpactCandidate{
			Path:   path,
			Role:   ReviewGenericImpactRoleNearbyProjectConfig,
			Reason: "project config is in the changed path directory or one of its ancestors",
			Token:  reviewGenericImpactConfigToken(path),
		})
	}
}

func (b *reviewGenericImpactCandidateBuilder) isChangedPath(path string) bool {
	_, ok := b.changedPaths[path]
	return ok
}

func (b *reviewGenericImpactCandidateBuilder) isNearChangedDir(path string) bool {
	dir := reviewGenericImpactPathDir(path)
	for _, changedDir := range b.changedDirs {
		if dir == changedDir {
			return true
		}
		for _, testDir := range reviewGenericImpactNearbyTestDirNames {
			if dir == reviewGenericImpactJoinPath(changedDir, testDir) ||
				dir == reviewGenericImpactJoinPath(reviewGenericImpactPathDir(changedDir), testDir) {
				return true
			}
		}
	}
	return false
}
