package evidence

func (c *reviewContextEvidenceCollector) collectChangedFileContext(changedFiles []ReviewChangedFile) []ReviewContextFileEvidence {
	files := make([]ReviewContextFileEvidence, 0, minReviewEvidenceInt(len(changedFiles), c.limits.MaxContextFiles))
	for _, file := range changedFiles {
		if c.contextErr() != nil {
			break
		}
		evidence, ok := c.collectChangedFile(file)
		if !ok {
			continue
		}
		files = append(files, evidence)
	}
	return files
}

func (c *reviewContextEvidenceCollector) collectChangedFile(file ReviewChangedFile) (ReviewContextFileEvidence, bool) {
	if reviewStatusHasPrefix(file.Status, "D") {
		return ReviewContextFileEvidence{}, false
	}
	return c.collectContextFile(file.Path, reviewContextFileRoleChanged)
}

func (c *reviewContextEvidenceCollector) collectContextFile(path, role string) (ReviewContextFileEvidence, bool) {
	absPath, relPath, err := resolveReviewEvidenceRepoPathLexically(c.repoRoot, path)
	evidence := ReviewContextFileEvidence{
		Path: normalizeReviewEvidenceDisplayPath(path),
		Role: role,
	}
	if err != nil {
		return c.skipContextFile(evidence, reviewContextSkipInvalidPath)
	}
	evidence.Path = relPath

	if isReviewContextGeneratedPath(relPath) {
		return c.skipContextFile(evidence, reviewContextSkipGenerated)
	}
	if isReviewContextVendorPath(relPath) {
		return c.skipContextFile(evidence, reviewContextSkipVendor)
	}

	return c.readContextFile(absPath, evidence)
}

func (c *reviewContextEvidenceCollector) skipContextFile(evidence ReviewContextFileEvidence, reason string) (ReviewContextFileEvidence, bool) {
	reserved, ok := c.reserveContextFileSlot(evidence)
	if !ok {
		return reserved, false
	}
	return markReviewContextFileSkipped(reserved, reason), true
}

func (c *reviewContextEvidenceCollector) readContextFile(absPath string, evidence ReviewContextFileEvidence) (ReviewContextFileEvidence, bool) {
	reserved, ok := c.reserveContextFileSlot(evidence)
	if !ok {
		return reserved, false
	}
	evidence = reserved

	remainingTotal := c.limits.MaxTotalContextBytes - c.totalContextRead
	maxBytes := minReviewEvidenceInt64(c.limits.MaxContextFileBytes, remainingTotal)
	file := readReviewEvidenceRegularFile(reviewEvidenceRegularFileReadInput{
		repoRoot:       c.repoRoot,
		absPath:        absPath,
		relPath:        evidence.Path,
		maxBytes:       maxBytes,
		maxFileBytes:   c.limits.MaxContextFileBytes,
		enforceFileMax: true,
		maxBudgetBytes: remainingTotal,
		enforceBudget:  true,
	})
	evidence.SizeBytes = file.sizeBytes
	evidence.ReadBytes = file.readBytes
	evidence.Truncated = file.truncated

	switch file.status {
	case reviewEvidenceRegularFileReadOK:
		c.totalContextRead += evidence.ReadBytes
		if file.binary {
			return markReviewContextFileSkipped(evidence, reviewContextSkipBinary), true
		}
		evidence.Content = string(file.data)
		return evidence, true
	case reviewEvidenceRegularFileReadMissing, reviewEvidenceRegularFileReadStatFailed:
		return markReviewContextFileSkipped(evidence, reviewContextSkipStatFailed), true
	case reviewEvidenceRegularFileReadSymlink:
		return markReviewContextFileSkipped(evidence, reviewContextSkipSymlink), true
	case reviewEvidenceRegularFileReadDir, reviewEvidenceRegularFileReadNonRegular:
		return markReviewContextFileSkipped(evidence, reviewContextSkipNonRegular), true
	case reviewEvidenceRegularFileReadInvalidPath:
		return markReviewContextFileSkipped(evidence, reviewContextSkipInvalidPath), true
	case reviewEvidenceRegularFileReadFileTooLarge:
		return markReviewContextFileSkipped(evidence, reviewContextSkipTooLarge), true
	case reviewEvidenceRegularFileReadBudgetExceeded:
		return markReviewContextFileSkipped(evidence, reviewContextSkipTotalBudgetExceeded), true
	case reviewEvidenceRegularFileReadFailed:
		return markReviewContextFileSkipped(evidence, reviewContextSkipReadFailed), true
	}
	return markReviewContextFileSkipped(evidence, reviewContextSkipReadFailed), true
}

func (c *reviewContextEvidenceCollector) reserveContextFileSlot(evidence ReviewContextFileEvidence) (ReviewContextFileEvidence, bool) {
	if c.contextFileCount < c.limits.MaxContextFiles {
		c.contextFileCount++
		return evidence, true
	}
	if c.maxContextFilesExceededLogged {
		return evidence, false
	}
	c.maxContextFilesExceededLogged = true
	return markReviewContextFileSkipped(evidence, reviewContextSkipMaxFilesExceeded), true
}

func markReviewContextFileSkipped(evidence ReviewContextFileEvidence, reason string) ReviewContextFileEvidence {
	evidence.Content = ""
	evidence.Skipped = true
	evidence.SkipReason = reason
	return evidence
}
