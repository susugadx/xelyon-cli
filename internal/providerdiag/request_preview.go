package providerdiag

import "strings"

// RequestPreviewTransport は doctor request preview の transport/body field を表す。
type RequestPreviewTransport struct {
	Method             string
	URL                string
	Headers            map[string]string
	PreviousResponseID string
	Body               any
}

// JSONHeaders は request preview の JSON body 用 Content-Type header を返す。
func JSONHeaders() map[string]string {
	return map[string]string{
		"Content-Type": "application/json",
	}
}

// RedactedAPIKeyHeaders は provider 固有 API key header を redacted 値で含む JSON headers を返す。
func RedactedAPIKeyHeaders(headerName string) map[string]string {
	headers := JSONHeaders()
	if header := strings.TrimSpace(headerName); header != "" {
		headers[header] = "<redacted>"
	}
	return headers
}

// RedactedSigV4Headers は AWS SigV4 request preview 用の redacted headers を返す。
func RedactedSigV4Headers() map[string]string {
	headers := JSONHeaders()
	headers["Accept"] = "application/json"
	headers["Authorization"] = "<redacted: AWS SigV4>"
	return headers
}
