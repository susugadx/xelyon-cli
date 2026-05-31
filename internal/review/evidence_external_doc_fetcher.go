package review

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
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

var (
	reviewExternalDocBlockedIPPrefixes = mustReviewExternalDocBlockedIPPrefixes(
		"0.0.0.0/8",
		"10.0.0.0/8",
		"100.64.0.0/10",
		"127.0.0.0/8",
		"169.254.0.0/16",
		"172.16.0.0/12",
		"192.0.0.0/24",
		"192.0.2.0/24",
		"192.168.0.0/16",
		"198.18.0.0/15",
		"198.51.100.0/24",
		"203.0.113.0/24",
		"224.0.0.0/4",
		"240.0.0.0/4",
		"255.255.255.255/32",
		"::/128",
		"::1/128",
		"::ffff:0:0/96",
		"64:ff9b::/96",
		"100::/64",
		"2001::/32",
		"2001:2::/48",
		"2001:db8::/32",
		"fc00::/7",
		"fe80::/10",
		"ff00::/8",
	)
)

// HTTPReviewExternalDocFetcher は HTTPS URL から bounded text snippet を取得する。
type HTTPReviewExternalDocFetcher struct {
	client       *http.Client
	networkGuard reviewExternalDocNetworkGuard
}

type reviewExternalDocNetworkGuard struct {
	lookupIPAddr     func(context.Context, string) ([]net.IPAddr, error)
	dialContext      func(context.Context, string, string) (net.Conn, error)
	allowPrivateAddr bool
}

// NewHTTPReviewExternalDocFetcher は external doc fetcher を構築する。
func NewHTTPReviewExternalDocFetcher(client *http.Client) *HTTPReviewExternalDocFetcher {
	if client == nil {
		client = http.DefaultClient
	}
	return &HTTPReviewExternalDocFetcher{
		client:       client,
		networkGuard: newReviewExternalDocNetworkGuard(),
	}
}

// FetchExternalDoc は検索結果 URL 由来の external_doc evidence を取得する。
func (f *HTTPReviewExternalDocFetcher) FetchExternalDoc(ctx context.Context, fetchReq ReviewExternalDocFetchRequest) ReviewExternalDocEvidence {
	doc := ReviewExternalDocEvidence{
		DocID:                   fetchReq.DocID,
		URL:                     strings.TrimSpace(fetchReq.URL),
		SourceDomain:            reviewExternalDocSourceDomain(fetchReq.URL),
		SourceCredibility:       ReviewExternalDocSourceCredibilityUnknown,
		SourceCredibilityReason: normalizeReviewExternalDocSourceCredibilityReason(ReviewExternalDocSourceCredibilityUnknown, ""),
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
	doc.SourceCredibility, doc.SourceCredibilityReason = classifyReviewExternalDocSourceCredibility(fetchReq, doc, sanitized)
	doc.SourceCredibility = normalizeReviewExternalDocSourceCredibility(doc.SourceCredibility)
	doc.SourceCredibilityReason = normalizeReviewExternalDocSourceCredibilityReason(doc.SourceCredibility, doc.SourceCredibilityReason)
	doc.Snippets = buildReviewExternalDocSnippets(doc.DocID, sanitized, truncated, fetchReq.FocusTerms)
	if len(doc.Snippets) == 0 {
		doc.Error = "fetched document produced no snippets"
	}
	return doc
}

func (f *HTTPReviewExternalDocFetcher) boundedClient(initialHost string) *http.Client {
	base := http.DefaultClient
	if f != nil && f.client != nil {
		base = f.client
	}
	guard := f.effectiveNetworkGuard()
	return &http.Client{
		Transport: reviewExternalDocBoundedTransport(base.Transport, guard),
		Jar:       base.Jar,
		Timeout:   reviewExternalDocFetchTimeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return validateReviewExternalDocRedirect(req, via, initialHost, guard)
		},
	}
}

func validateReviewExternalDocRedirect(req *http.Request, via []*http.Request, initialHost string, guard reviewExternalDocNetworkGuard) error {
	if len(via) >= 3 {
		return fmt.Errorf("stopped after 3 redirects")
	}
	if req == nil || !reviewExternalDocURLIsHTTPS(req.URL) {
		return errors.New(reviewExternalDocNonHTTPSRedirectError)
	}
	if req.URL.Host != initialHost {
		return fmt.Errorf("cross-host redirect rejected")
	}
	return guard.validateHost(req.Context(), req.URL.Hostname())
}

func reviewExternalDocURLIsHTTPS(u *url.URL) bool {
	return u != nil && strings.EqualFold(u.Scheme, "https")
}

func (f *HTTPReviewExternalDocFetcher) effectiveNetworkGuard() reviewExternalDocNetworkGuard {
	if f != nil && (f.networkGuard.lookupIPAddr != nil || f.networkGuard.dialContext != nil || f.networkGuard.allowPrivateAddr) {
		guard := f.networkGuard
		guard.setDefaults()
		return guard
	}
	return newReviewExternalDocNetworkGuard()
}

func newReviewExternalDocNetworkGuard() reviewExternalDocNetworkGuard {
	guard := reviewExternalDocNetworkGuard{}
	guard.setDefaults()
	return guard
}

func (g *reviewExternalDocNetworkGuard) setDefaults() {
	if g.lookupIPAddr == nil {
		g.lookupIPAddr = net.DefaultResolver.LookupIPAddr
	}
	if g.dialContext == nil {
		dialer := &net.Dialer{}
		g.dialContext = dialer.DialContext
	}
}

func reviewExternalDocBoundedTransport(base http.RoundTripper, guard reviewExternalDocNetworkGuard) http.RoundTripper {
	defaultTransport, _ := http.DefaultTransport.(*http.Transport)
	if defaultTransport == nil {
		return reviewExternalDocUnsupportedTransport{}
	}
	clone := defaultTransport.Clone()
	if transport, ok := base.(*http.Transport); ok && transport.TLSClientConfig != nil {
		clone.TLSClientConfig = transport.TLSClientConfig.Clone()
	}
	clone.Proxy = nil
	clone.DialContext = guard.dialContextForPublicHost
	clone.DialTLSContext = nil
	return clone
}

type reviewExternalDocUnsupportedTransport struct{}

func (reviewExternalDocUnsupportedTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, fmt.Errorf("external doc fetch requires an http transport")
}

func (g reviewExternalDocNetworkGuard) validateHost(ctx context.Context, host string) error {
	if g.allowPrivateAddr {
		return nil
	}
	if strings.TrimSpace(host) == "" {
		return fmt.Errorf("external doc host is required")
	}
	_, err := g.publicIPsForHost(ctx, host)
	return err
}

func (g reviewExternalDocNetworkGuard) publicIPsForHost(ctx context.Context, host string) ([]netip.Addr, error) {
	g.setDefaults()
	if ip, err := netip.ParseAddr(strings.Trim(host, "[]")); err == nil {
		ip = ip.Unmap()
		if !reviewExternalDocIsPublicRoutableIP(ip) {
			return nil, fmt.Errorf("external doc host must resolve to public routable IPs")
		}
		return []netip.Addr{ip}, nil
	}

	addrs, err := g.lookupIPAddr(ctx, host)
	if err != nil {
		return nil, fmt.Errorf("external doc host lookup failed: %w", err)
	}
	if len(addrs) == 0 {
		return nil, fmt.Errorf("external doc host lookup returned no addresses")
	}

	ips := make([]netip.Addr, 0, len(addrs))
	for _, resolved := range addrs {
		ip, ok := netip.AddrFromSlice(resolved.IP)
		if !ok {
			return nil, fmt.Errorf("external doc host lookup returned an invalid address")
		}
		ip = ip.Unmap()
		if !reviewExternalDocIsPublicRoutableIP(ip) {
			return nil, fmt.Errorf("external doc host must resolve to public routable IPs")
		}
		ips = append(ips, ip)
	}
	return ips, nil
}

func (g reviewExternalDocNetworkGuard) dialContextForPublicHost(ctx context.Context, network, address string) (net.Conn, error) {
	if g.allowPrivateAddr {
		g.setDefaults()
		return g.dialContext(ctx, network, address)
	}
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, err
	}
	ips, err := g.publicIPsForHost(ctx, host)
	if err != nil {
		return nil, err
	}

	var lastErr error
	for _, ip := range ips {
		if network == "tcp4" && !ip.Is4() {
			continue
		}
		if network == "tcp6" && !ip.Is6() {
			continue
		}
		conn, err := g.dialContext(ctx, network, net.JoinHostPort(ip.String(), port))
		if err == nil {
			return conn, nil
		}
		lastErr = err
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, fmt.Errorf("external doc host has no address for network %s", network)
}

func reviewExternalDocIsPublicRoutableIP(ip netip.Addr) bool {
	if !ip.IsValid() || !ip.IsGlobalUnicast() || ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsMulticast() || ip.IsUnspecified() {
		return false
	}
	for _, prefix := range reviewExternalDocBlockedIPPrefixes {
		if prefix.Contains(ip) {
			return false
		}
	}
	return true
}

func mustReviewExternalDocBlockedIPPrefixes(values ...string) []netip.Prefix {
	prefixes := make([]netip.Prefix, 0, len(values))
	for _, value := range values {
		prefixes = append(prefixes, netip.MustParsePrefix(value))
	}
	return prefixes
}

func readReviewExternalDocBody(body io.Reader) ([]byte, bool, error) {
	var buf bytes.Buffer
	limit := int64(reviewExternalDocMaxResponseBytes + 1)
	if _, err := io.Copy(&buf, io.LimitReader(body, limit)); err != nil {
		return nil, false, err
	}
	data := buf.Bytes()
	if len(data) > reviewExternalDocMaxResponseBytes {
		return data[:reviewExternalDocMaxResponseBytes], true, nil
	}
	return data, false, nil
}

func reviewExternalDocSourceDomain(rawURL string) string {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return ""
	}
	return strings.ToLower(parsed.Hostname())
}
