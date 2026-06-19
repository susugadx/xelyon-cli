package externaldoc

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
)

func (f *HTTPFetcher) boundedClient(initialHost string) *http.Client {
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
