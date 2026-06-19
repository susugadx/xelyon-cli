package externaldoc

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

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
