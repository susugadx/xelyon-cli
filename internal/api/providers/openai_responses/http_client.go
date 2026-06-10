package openairesponses

import "net/http"

// NewLongRunningHTTPClient は長時間の Responses 系 request 用に header timeout を外した HTTP client を返します。
func NewLongRunningHTTPClient(base *http.Client) *http.Client {
	if base == nil {
		base = &http.Client{}
	}
	client := *base

	transport := base.Transport
	if transport == nil {
		transport = http.DefaultTransport
	}
	if httpTransport, ok := transport.(*http.Transport); ok {
		cloned := httpTransport.Clone()
		cloned.ResponseHeaderTimeout = 0
		client.Transport = cloned
		return &client
	}

	client.Transport = transport
	return &client
}
