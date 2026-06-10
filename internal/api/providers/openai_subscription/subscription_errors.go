package openaisubscription

import (
	"fmt"
	"io"
	"net/http"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

func handleSubscriptionHTTPError(resp *http.Response, spinner *ui.Spinner, providerName string) error {
	if spinner != nil {
		spinner.Stop()
	}
	if rateLimitErr := api.HandleRateLimit(resp); rateLimitErr != nil {
		return rateLimitErr
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, subscriptionMaxHTTPBodyBytes+1))
	if err != nil {
		return fmt.Errorf("%s API error (status %d): unable to read response body - %v", providerName, resp.StatusCode, err)
	}
	if len(body) > subscriptionMaxHTTPBodyBytes {
		return fmt.Errorf("%s API error (status %d): response body exceeded %d bytes", providerName, resp.StatusCode, subscriptionMaxHTTPBodyBytes)
	}
	return api.SanitizeErrorMessage([]byte(RedactSubscriptionSecrets(string(body))), resp.StatusCode)
}
