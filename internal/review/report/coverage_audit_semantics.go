package report

import "strings"

type coverageTextSemantics struct {
	mentionsExternalEvidenceContext  bool
	mentionsPostPass1Evidence        bool
	mentionsExplicitAddedExternalDoc bool
	mentionsExternalSupportState     bool
	mentionsExternalGapState         bool
	claimsConfirmedExternalSpec      bool
}

func classifyCoverageText(text string) coverageTextSemantics {
	return classifyNormalizedCoverageText(strings.ToLower(text))
}

func classifyNormalizedCoverageText(normalized string) coverageTextSemantics {
	if normalized == "" {
		return coverageTextSemantics{}
	}
	return coverageTextSemantics{
		mentionsExternalEvidenceContext:  coverageTextContainsAny(normalized, coverageExternalEvidenceContextPhrases),
		mentionsPostPass1Evidence:        coverageTextContainsAny(normalized, coveragePostPass1ExternalEvidencePhrases),
		mentionsExplicitAddedExternalDoc: coverageTextContainsAny(normalized, coverageExplicitAddedExternalDocPhrases),
		mentionsExternalSupportState:     coverageTextContainsAny(normalized, coverageExternalSupportStatePhrases),
		mentionsExternalGapState:         coverageTextContainsAny(normalized, coverageExternalEvidenceGapStatePhrases),
		claimsConfirmedExternalSpec:      coverageTextClaimsConfirmedExternalSpec(normalized),
	}
}

func textReflectsExternalEvidenceRequirement(text string, requirement coverageExternalReflectionRequirement) bool {
	normalized := strings.ToLower(text)
	switch requirement.kind {
	case coverageExternalReflectionRequirementAddedDocs:
		return textReferencesExternalEvidenceValue(normalized, requirement) ||
			textReferencesExplicitAddedExternalDoc(normalized) ||
			textReferencesExternalSupportState(normalized)
	case coverageExternalReflectionRequirementSearchGap:
		return textReferencesExternalEvidenceGapState(normalized) &&
			textReferencesExternalEvidenceGapContext(normalized, requirement)
	case coverageExternalReflectionRequirementGeneric:
		return textReferencesPostPass1ExternalEvidence(normalized) ||
			textReferencesExternalSupportState(normalized) ||
			textReferencesExternalSupportDiagnostic(normalized, requirement)
	default:
		return false
	}
}

func textReferencesExternalEvidenceValue(text string, requirement coverageExternalReflectionRequirement) bool {
	for _, value := range append(append([]string(nil), requirement.docIDs...), requirement.docURLs...) {
		value = strings.ToLower(strings.TrimSpace(value))
		if value != "" && strings.Contains(text, value) {
			return true
		}
	}
	return false
}

func textReferencesExternalSupportDiagnostic(text string, requirement coverageExternalReflectionRequirement) bool {
	for _, value := range requirement.diagnostics {
		value = strings.ToLower(strings.TrimSpace(value))
		if value == "" {
			continue
		}
		if strings.Contains(text, value) || strings.Contains(text, strings.ReplaceAll(value, "_", " ")) {
			return true
		}
	}
	return false
}

func textReferencesExternalEvidenceGapState(text string) bool {
	return classifyCoverageText(text).mentionsExternalGapState
}

func textReferencesExternalEvidenceGapContext(text string, requirement coverageExternalReflectionRequirement) bool {
	return textReferencesPostPass1ExternalEvidence(text) ||
		textReferencesExternalEvidenceValue(text, requirement) ||
		textReferencesExternalSupportDiagnostic(text, requirement) ||
		textReferencesExternalEvidenceContext(text)
}

func textReferencesExplicitAddedExternalDoc(text string) bool {
	semantics := classifyCoverageText(text)
	return semantics.mentionsPostPass1Evidence || semantics.mentionsExplicitAddedExternalDoc
}

func textReferencesExternalEvidenceContext(text string) bool {
	return classifyCoverageText(text).mentionsExternalEvidenceContext
}

func textReferencesExternalSupportState(text string) bool {
	return classifyCoverageText(text).mentionsExternalSupportState
}

func textReferencesPostPass1ExternalEvidence(text string) bool {
	return classifyCoverageText(text).mentionsPostPass1Evidence
}

func textClaimsConfirmedExternalSpec(text string) bool {
	return classifyCoverageText(text).claimsConfirmedExternalSpec
}

func coverageTextClaimsConfirmedExternalSpec(text string) bool {
	for _, phrase := range coverageConfirmedExternalSpecStrongClaimPhrases {
		if coverageTextHasUnnegatedConfirmationPhrase(text, phrase) {
			return true
		}
	}
	for _, phrase := range coverageOfficialConfirmationSubjectPhrases {
		if coverageTextHasAssertedOfficialConfirmationSubject(text, phrase) {
			return true
		}
	}
	return false
}

func coverageTextHasUnnegatedConfirmationPhrase(text, phrase string) bool {
	offset := 0
	for {
		index := strings.Index(text[offset:], phrase)
		if index < 0 {
			return false
		}
		index += offset
		end := index + len(phrase)
		offset = index + 1
		if !coverageTextPhraseHasTokenBoundary(text, index, end) {
			continue
		}
		if coverageTextNegatesConfirmationAt(text, phrase, index) {
			continue
		}
		return true
	}
}

func coverageTextHasAssertedOfficialConfirmationSubject(text, phrase string) bool {
	offset := 0
	for {
		index := strings.Index(text[offset:], phrase)
		if index < 0 {
			return false
		}
		index += offset
		end := index + len(phrase)
		offset = index + 1
		if !coverageTextPhraseHasTokenBoundary(text, index, end) {
			continue
		}
		if coverageTextNegatesConfirmationAt(text, phrase, index) {
			continue
		}
		if !coverageTextOfficialConfirmationSubjectHasAssertionContext(text, index, end) {
			continue
		}
		return true
	}
}

func coverageTextOfficialConfirmationSubjectHasAssertionContext(text string, start, end int) bool {
	prefixStart := start - 40
	if prefixStart < 0 {
		prefixStart = 0
	}
	prefix := coverageTextConfirmationPrefixClause(text[prefixStart:start])

	suffixEnd := end + 80
	if suffixEnd > len(text) {
		suffixEnd = len(text)
	}
	suffix := coverageTextConfirmationSuffixClause(text[end:suffixEnd])

	return coverageTextContainsAny(prefix, coverageOfficialConfirmationAssertionPrefixPhrases) ||
		coverageTextContainsAny(suffix, coverageOfficialConfirmationAssertionSuffixPhrases)
}

func coverageTextPhraseHasTokenBoundary(text string, start, end int) bool {
	return (start == 0 || !isCoverageConfirmationTokenChar(text[start-1])) &&
		(end >= len(text) || !isCoverageConfirmationTokenChar(text[end]))
}

func isCoverageConfirmationTokenChar(value byte) bool {
	return value >= 'a' && value <= 'z' || value >= '0' && value <= '9' || value == '_'
}

func coverageTextNegatesConfirmationAt(text, phrase string, index int) bool {
	prefixStart := index - 40
	if prefixStart < 0 {
		prefixStart = 0
	}
	prefix := coverageTextConfirmationPrefixClause(text[prefixStart:index])
	for _, negation := range coverageConfirmationPrefixNonClaimPhrases {
		if strings.Contains(prefix, negation) {
			return true
		}
	}

	suffixEnd := index + len(phrase) + 80
	if suffixEnd > len(text) {
		suffixEnd = len(text)
	}
	suffix := text[index+len(phrase) : suffixEnd]
	for _, negation := range coverageConfirmationSuffixNonClaimPhrases {
		if strings.Contains(suffix, negation) {
			return true
		}
	}
	return false
}

func coverageTextConfirmationPrefixClause(prefix string) string {
	if separator := strings.LastIndexAny(prefix, ".,;:\n"); separator >= 0 {
		prefix = prefix[separator+1:]
	}
	for _, boundary := range coverageConfirmationPrefixClauseBoundaryPhrases {
		if index := strings.LastIndex(prefix, boundary); index >= 0 {
			prefix = prefix[index+len(boundary):]
		}
	}
	return prefix
}

func coverageTextConfirmationSuffixClause(suffix string) string {
	end := len(suffix)
	if separator := strings.IndexAny(suffix, ".,;:\n"); separator >= 0 {
		end = separator
	}
	for _, boundary := range coverageConfirmationPrefixClauseBoundaryPhrases {
		if index := strings.Index(suffix, boundary); index >= 0 && index < end {
			end = index
		}
	}
	return suffix[:end]
}

func coverageTextContainsAny(text string, phrases []string) bool {
	for _, phrase := range phrases {
		if strings.Contains(text, phrase) {
			return true
		}
	}
	return false
}
