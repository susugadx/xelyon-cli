package review

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestHTTPReviewExternalDocFetcherRequiresHTTPS(t *testing.T) {
	fetcher := NewHTTPReviewExternalDocFetcher(nil)

	got := fetcher.FetchExternalDoc(context.Background(), "http://example.test/spec", "external-doc-1")

	if got.Error == "" || !strings.Contains(got.Error, "HTTPS") {
		t.Fatalf("Error = %q, want HTTPS rejection", got.Error)
	}
	if len(got.Snippets) != 0 {
		t.Fatalf("Snippets = %#v, want none for rejected URL", got.Snippets)
	}
}

func TestHTTPReviewExternalDocFetcherRejectsLoopbackHost(t *testing.T) {
	fetcher := NewHTTPReviewExternalDocFetcher(nil)

	got := fetcher.FetchExternalDoc(context.Background(), "https://127.0.0.1:8443/spec", "external-doc-1")

	if got.Error == "" || !strings.Contains(got.Error, "public routable") {
		t.Fatalf("Error = %q, want public routable host rejection", got.Error)
	}
	if got.StatusCode != 0 {
		t.Fatalf("StatusCode = %d, want 0 for rejected URL before fetch", got.StatusCode)
	}
}

func TestHTTPReviewExternalDocFetcherRejectsPrivateDNSResolution(t *testing.T) {
	fetcher := NewHTTPReviewExternalDocFetcher(nil)
	fetcher.networkGuard = reviewExternalDocNetworkGuard{
		lookupIPAddr: func(context.Context, string) ([]net.IPAddr, error) {
			return []net.IPAddr{{IP: net.ParseIP("10.0.0.10")}}, nil
		},
	}

	got := fetcher.FetchExternalDoc(context.Background(), "https://docs.example.test/spec", "external-doc-1")

	if got.Error == "" || !strings.Contains(got.Error, "public routable") {
		t.Fatalf("Error = %q, want private DNS rejection", got.Error)
	}
}

func TestHTTPReviewExternalDocFetcherRejectsPrivateRedirectHost(t *testing.T) {
	fetcher := NewHTTPReviewExternalDocFetcher(nil)
	fetcher.networkGuard = reviewExternalDocNetworkGuard{
		lookupIPAddr: func(context.Context, string) ([]net.IPAddr, error) {
			return []net.IPAddr{{IP: net.ParseIP("172.16.0.4")}}, nil
		},
	}
	client := fetcher.boundedClient("docs.example.test")
	req := httptest.NewRequest(http.MethodGet, "https://docs.example.test/redirected", nil)

	err := client.CheckRedirect(req, []*http.Request{httptest.NewRequest(http.MethodGet, "https://docs.example.test/start", nil)})

	if err == nil || !strings.Contains(err.Error(), "public routable") {
		t.Fatalf("CheckRedirect error = %v, want private redirect host rejection", err)
	}
}

func TestHTTPReviewExternalDocFetcherRejectsNonTextContentType(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		_, _ = w.Write([]byte{0, 1, 2})
	}))
	defer server.Close()

	fetcher := newLocalReviewExternalDocFetcherForTest(server.Client())
	got := fetcher.FetchExternalDoc(context.Background(), server.URL, "external-doc-1")

	if got.Error == "" || !strings.Contains(got.Error, "non-text") {
		t.Fatalf("Error = %q, want non-text rejection", got.Error)
	}
	if got.StatusCode != http.StatusOK {
		t.Fatalf("StatusCode = %d, want 200", got.StatusCode)
	}
}

func TestHTTPReviewExternalDocFetcherRejectsCrossHostRedirect(t *testing.T) {
	target := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("target"))
	}))
	defer target.Close()
	redirector := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL, http.StatusFound)
	}))
	defer redirector.Close()

	fetcher := newLocalReviewExternalDocFetcherForTest(redirector.Client())
	got := fetcher.FetchExternalDoc(context.Background(), redirector.URL, "external-doc-1")

	if got.Error == "" || !strings.Contains(got.Error, "cross-host redirect") {
		t.Fatalf("Error = %q, want cross-host redirect rejection", got.Error)
	}
}

func TestHTTPReviewExternalDocFetcherFollowsSameHostRedirect(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/redirect" {
			http.Redirect(w, r, "/spec", http.StatusFound)
			return
		}
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte("external spec text"))
	}))
	defer server.Close()

	fetcher := newLocalReviewExternalDocFetcherForTest(server.Client())
	got := fetcher.FetchExternalDoc(context.Background(), server.URL+"/redirect", "external-doc-1")

	if got.Error != "" {
		t.Fatalf("Error = %q, want nil for same-host redirect", got.Error)
	}
	if got.URL != server.URL+"/spec" {
		t.Fatalf("URL = %q, want final redirected URL", got.URL)
	}
	if len(got.Snippets) == 0 {
		t.Fatal("Snippets empty, want fetched text")
	}
}

func TestHTTPReviewExternalDocFetcherRejectsSameHostDowngradeRedirect(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/redirect" {
			http.Redirect(w, r, "http://"+r.Host+"/spec", http.StatusFound)
			return
		}
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte("external spec text"))
	}))
	defer server.Close()

	fetcher := newLocalReviewExternalDocFetcherForTest(server.Client())
	got := fetcher.FetchExternalDoc(context.Background(), server.URL+"/redirect", "external-doc-1")

	if got.Error == "" || !strings.Contains(got.Error, "non-HTTPS redirect rejected") {
		t.Fatalf("Error = %q, want non-HTTPS redirect rejection", got.Error)
	}
	if strings.HasPrefix(got.URL, "http://") {
		t.Fatalf("URL = %q, want evidence to keep HTTPS URL on downgrade rejection", got.URL)
	}
	if len(got.Snippets) != 0 {
		t.Fatalf("Snippets = %#v, want none for downgrade rejection", got.Snippets)
	}
}

func TestHTTPReviewExternalDocFetcherSanitizesHTMLAndBoundsSnippets(t *testing.T) {
	body := `<html><head><style>.x{}</style><script>alert("x")</script></head><body><h1>Spec</h1><p>` +
		strings.Repeat("External contract text ", 120) +
		`</p></body></html>`
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(body))
	}))
	defer server.Close()

	fetcher := newLocalReviewExternalDocFetcherForTest(server.Client())
	got := fetcher.FetchExternalDoc(context.Background(), server.URL, "external-doc-1")

	if got.Error != "" {
		t.Fatalf("Error = %q, want nil", got.Error)
	}
	if len(got.Snippets) == 0 {
		t.Fatal("Snippets empty, want sanitized snippet")
	}
	if strings.Contains(got.Snippets[0].Content, "<script") || strings.Contains(got.Snippets[0].Content, "alert") {
		t.Fatalf("snippet leaked script content: %q", got.Snippets[0].Content)
	}
	if len(got.Snippets[0].Content) > reviewExternalDocMaxSnippetBytes {
		t.Fatalf("snippet bytes = %d, want <= %d", len(got.Snippets[0].Content), reviewExternalDocMaxSnippetBytes)
	}
	if !strings.HasPrefix(got.ContentHash, "sha256:") || !strings.HasPrefix(got.Snippets[0].ContentHash, "sha256:") {
		t.Fatalf("hashes = (%q, %q), want sha256 hashes", got.ContentHash, got.Snippets[0].ContentHash)
	}
}

func TestHTTPReviewExternalDocFetcherTruncatesLargeResponses(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte(strings.Repeat("x", reviewExternalDocMaxResponseBytes+100)))
	}))
	defer server.Close()

	fetcher := newLocalReviewExternalDocFetcherForTest(server.Client())
	got := fetcher.FetchExternalDoc(context.Background(), server.URL, "external-doc-1")

	if got.Error != "" {
		t.Fatalf("Error = %q, want nil", got.Error)
	}
	if !got.Truncated {
		t.Fatal("Truncated = false, want true for response over max bytes")
	}
	if len(got.Snippets) != reviewExternalDocMaxSnippets {
		t.Fatalf("Snippets = %d, want max %d snippets", len(got.Snippets), reviewExternalDocMaxSnippets)
	}
	for _, snippet := range got.Snippets {
		if len(snippet.Content) > reviewExternalDocMaxSnippetBytes {
			t.Fatalf("snippet bytes = %d, want <= %d", len(snippet.Content), reviewExternalDocMaxSnippetBytes)
		}
		if !snippet.Truncated {
			t.Fatalf("snippet %#v Truncated = false, want true", snippet.SnippetID)
		}
	}
}

func TestHTTPReviewExternalDocFetcherUsesFixedTimeout(t *testing.T) {
	fetcher := NewHTTPReviewExternalDocFetcher(&http.Client{Timeout: time.Minute})

	client := fetcher.boundedClient("docs.example.test")

	if client.Timeout != reviewExternalDocFetchTimeout {
		t.Fatalf("Timeout = %s, want %s", client.Timeout, reviewExternalDocFetchTimeout)
	}
}

func newLocalReviewExternalDocFetcherForTest(client *http.Client) *HTTPReviewExternalDocFetcher {
	fetcher := NewHTTPReviewExternalDocFetcher(client)
	fetcher.networkGuard = reviewExternalDocNetworkGuard{allowPrivateAddr: true}
	return fetcher
}
