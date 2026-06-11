package openaisubscription

import (
	"bytes"
	"context"
	"net/http"
	"strings"
)

func (p *SubscriptionProvider) prepareSubscriptionResponsesRequest(ctx context.Context, url string, payload []byte) (*http.Request, error) {
	authCfg := DefaultSubscriptionAuthConfig()
	originator, err := validateSubscriptionOriginatorForRequest(authCfg.Originator)
	if err != nil {
		return nil, err
	}
	credential, err := GetSubscriptionCredentialForRequest(ctx, authCfg, p.HTTPClient)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+credential.AccessToken)
	if strings.TrimSpace(credential.AccountID) != "" {
		req.Header.Set("ChatGPT-Account-Id", credential.AccountID)
	}
	req.Header.Set("originator", originator)
	req.Header.Set("User-Agent", subscriptionUserAgent())
	return req, nil
}
