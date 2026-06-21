package externaldoc

import (
	"regexp"
	"strings"
)

var reviewExternalDocCredibilityTokenRE = regexp.MustCompile(`[a-z0-9]+`)

func classifySourceCredibility(fetchReq FetchRequest, doc Evidence, content string) (SourceCredibility, string) {
	source := reviewExternalDocNormalizeCredibilityDomain(doc.SourceDomain)
	title := strings.ToLower(strings.Join(strings.Fields(fetchReq.SearchResultTitle), " "))
	body := strings.ToLower(strings.Join(strings.Fields(content), " "))
	subjectTokens := reviewExternalDocSubjectCredibilityTokens(fetchReq.QuerySubjectHint)
	combined := strings.TrimSpace(source + " " + title + " " + body)

	if reviewExternalDocHasThirdPartySignal(source, title, fetchReq.URL) {
		return SourceCredibilityThirdParty, "third_party: known third-party host, source URL, or source metadata signal is present"
	}
	if combined == "" {
		return SourceCredibilityUnknown, "unknown: source metadata and content are unavailable"
	}

	trustedDomains := reviewExternalDocTrustedDomainsForSubject(fetchReq.QuerySubjectHint)
	if len(trustedDomains) == 0 {
		return SourceCredibilityUnknown, "unknown: query subject has no trusted domain mapping for official documentation"
	}
	if !reviewExternalDocSourceMatchesTrustedDomains(source, trustedDomains) {
		return SourceCredibilityUnknown, "unknown: source domain does not match trusted domains for the query subject"
	}
	if !reviewExternalDocSubjectMatchesText(title, body, subjectTokens) {
		return SourceCredibilityUnknown, "unknown: trusted domain matched but title or content lacks a query subject text signal"
	}
	if !reviewExternalDocContentHasReferenceSignal(body) {
		return SourceCredibilityUnknown, "unknown: trusted domain and subject text matched but reference content signal is absent"
	}
	return SourceCredibilityOfficialCandidate, "official_candidate: trusted source domain, query subject text signal, and reference content signal are present"
}

func normalizeSourceCredibility(value SourceCredibility) SourceCredibility {
	switch value {
	case SourceCredibilityOfficialCandidate, SourceCredibilityThirdParty, SourceCredibilityUnknown:
		return value
	default:
		return SourceCredibilityUnknown
	}
}

func normalizeSourceCredibilityReason(value SourceCredibility, reason string) string {
	reason = strings.Join(strings.Fields(strings.TrimSpace(reason)), " ")
	if reason != "" {
		return reason
	}
	switch normalizeSourceCredibility(value) {
	case SourceCredibilityOfficialCandidate:
		return "official_candidate: source signals matched official documentation heuristics"
	case SourceCredibilityThirdParty:
		return "third_party: source signals matched unofficial or community heuristics"
	default:
		return "unknown: source credibility was not established"
	}
}
