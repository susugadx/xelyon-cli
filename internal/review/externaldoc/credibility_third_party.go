package externaldoc

import (
	"net/url"
	"strings"
)

func reviewExternalDocHasThirdPartySignal(sourceDomain, title, sourceURL string) bool {
	for _, thirdPartyDomain := range []string{
		"blogspot.com",
		"dev.to",
		"hashnode.com",
		"hashnode.dev",
		"medium.com",
		"reddit.com",
		"stackexchange.com",
		"stackoverflow.com",
		"substack.com",
	} {
		if reviewExternalDocSourceMatchesTrustedDomain(sourceDomain, thirdPartyDomain) {
			return true
		}
	}
	if reviewExternalDocHasGitHubThirdPartySignal(sourceDomain, title, sourceURL) {
		return true
	}
	metadata := strings.TrimSpace(sourceDomain + " " + title)
	for _, signal := range []string{
		"community guide",
		"community tutorial",
		"community blog",
		"community article",
		"not official",
		"personal blog",
		"unofficial",
	} {
		if strings.Contains(metadata, signal) {
			return true
		}
	}
	return reviewExternalDocMetadataHasThirdPartySourceType(metadata)
}

func reviewExternalDocHasGitHubThirdPartySignal(sourceDomain, title, sourceURL string) bool {
	if !reviewExternalDocSourceMatchesTrustedDomain(sourceDomain, "github.com") {
		return false
	}
	path := reviewExternalDocCredibilityURLPath(sourceURL)
	for _, signal := range []string{"/issues/", "/pull/", "/discussions/"} {
		if strings.Contains(path, signal) {
			return true
		}
	}
	metadata := " " + title + " "
	for _, signal := range []string{
		" github issue ",
		" github discussion ",
		" github pull request ",
		" issue #",
		" discussion #",
		" pull request #",
	} {
		if strings.Contains(metadata, signal) {
			return true
		}
	}
	return false
}

func reviewExternalDocCredibilityURLPath(sourceURL string) string {
	parsed, err := url.Parse(strings.TrimSpace(sourceURL))
	if err != nil {
		return ""
	}
	return strings.ToLower(parsed.Path)
}

func reviewExternalDocMetadataHasThirdPartySourceType(metadata string) bool {
	for _, sourceType := range []string{"guide", "tutorial", "blog", "article", "post"} {
		if strings.Contains(metadata, "third party "+sourceType) || strings.Contains(metadata, "third-party "+sourceType) {
			return true
		}
	}
	return false
}
