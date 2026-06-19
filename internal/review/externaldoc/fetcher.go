package externaldoc

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	reviewExternalDocFetchTimeout     = 5 * time.Second
	reviewExternalDocMaxResponseBytes = 256 * 1024
	reviewExternalDocMaxSnippets      = 3
	reviewExternalDocMaxSnippetBytes  = 1200

	reviewExternalDocNonHTTPSRedirectError = "non-HTTPS redirect rejected"
)

// HTTPFetcher は HTTPS URL から bounded text snippet を取得する。
type HTTPFetcher struct {
	client       *http.Client
	networkGuard reviewExternalDocNetworkGuard
}

// NewHTTPFetcher は external doc fetcher を構築する。
func NewHTTPFetcher(client *http.Client) *HTTPFetcher {
	if client == nil {
		client = http.DefaultClient
	}
	return &HTTPFetcher{
		client:       client,
		networkGuard: newReviewExternalDocNetworkGuard(),
	}
}

// FetchExternalDoc は検索結果 URL 由来の external_doc evidence を取得する。
func (f *HTTPFetcher) FetchExternalDoc(ctx context.Context, fetchReq FetchRequest) Evidence {
	doc := Evidence{
		DocID:                   fetchReq.DocID,
		URL:                     strings.TrimSpace(fetchReq.URL),
		SourceDomain:            reviewExternalDocSourceDomain(fetchReq.URL),
		SourceCredibility:       SourceCredibilityUnknown,
		SourceCredibilityReason: normalizeSourceCredibilityReason(SourceCredibilityUnknown, ""),
		FetchedAt:               time.Now().UTC(),
	}
	parsed, err := url.Parse(doc.URL)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" {
		doc.Error = "external doc fetch requires an HTTPS URL"
		return doc
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := f.effectiveNetworkGuard().validateHost(ctx, parsed.Hostname()); err != nil {
		doc.Error = err.Error()
		return doc
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, parsed.String(), nil)
	if err != nil {
		doc.Error = err.Error()
		return doc
	}
	client := f.boundedClient(parsed.Host)
	resp, err := client.Do(httpReq)
	if err != nil {
		doc.Error = err.Error()
		return doc
	}
	defer resp.Body.Close()

	if resp.Request == nil || !reviewExternalDocURLIsHTTPS(resp.Request.URL) {
		doc.Error = reviewExternalDocNonHTTPSRedirectError
		return doc
	}
	doc.URL = resp.Request.URL.String()
	doc.SourceDomain = reviewExternalDocSourceDomain(doc.URL)
	doc.StatusCode = resp.StatusCode
	doc.ContentType = resp.Header.Get("Content-Type")
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		doc.Error = fmt.Sprintf("unexpected status code %d", resp.StatusCode)
		return doc
	}
	if !reviewExternalDocAllowedContentType(doc.ContentType) {
		doc.Error = "non-text content type rejected"
		return doc
	}

	body, truncated, err := readReviewExternalDocBody(resp.Body)
	if err != nil {
		doc.Error = err.Error()
		return doc
	}
	sanitized := sanitizeReviewExternalDocText(body, doc.ContentType)
	if sanitized == "" {
		doc.Error = "fetched document has no text content"
		return doc
	}
	doc.Truncated = truncated
	doc.ContentHash = reviewExternalDocContentHash(sanitized)
	doc.SourceCredibility, doc.SourceCredibilityReason = classifySourceCredibility(fetchReq, doc, sanitized)
	doc.SourceCredibility = normalizeSourceCredibility(doc.SourceCredibility)
	doc.SourceCredibilityReason = normalizeSourceCredibilityReason(doc.SourceCredibility, doc.SourceCredibilityReason)
	doc.Snippets = buildReviewExternalDocSnippets(doc.DocID, sanitized, truncated, fetchReq.FocusTerms)
	if len(doc.Snippets) == 0 {
		doc.Error = "fetched document produced no snippets"
	}
	return doc
}
