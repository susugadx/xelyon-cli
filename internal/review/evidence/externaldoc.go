package evidence

import (
	"net/http"

	"github.com/susugadx/xelyon-cli/internal/review/externaldoc"
)

// NewHTTPReviewExternalDocFetcher は external doc fetcher を構築する。
func NewHTTPReviewExternalDocFetcher(client *http.Client) *HTTPReviewExternalDocFetcher {
	return externaldoc.NewHTTPFetcher(client)
}
