package evidence

var reviewEvidenceGoLanguage = reviewEvidenceLanguageSpec{
	fileExtension:             ".go",
	testFileSuffix:            "_test.go",
	relatedCandidatePathspec:  "*.go",
	relatedSourceContextRole:  reviewContextFileRoleRelatedGo,
	relatedTestContextRole:    reviewContextFileRoleRelatedTest,
	extractRelatedSearchTerms: extractReviewGoRelatedSearchTerms,
}
