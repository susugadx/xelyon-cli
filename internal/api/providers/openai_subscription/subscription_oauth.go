package openaisubscription

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"net"
	"net/http"
	"net/url"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

// SubscriptionBrowserLoginOptions は browser OAuth login の実行設定です。
type SubscriptionBrowserLoginOptions struct {
	Config      SubscriptionAuthConfig
	HTTPClient  *http.Client
	Output      io.Writer
	OpenBrowser func(string) error
	Timeout     time.Duration
}

// SubscriptionDeviceLoginOptions は device OAuth login の実行設定です。
type SubscriptionDeviceLoginOptions struct {
	Config     SubscriptionAuthConfig
	HTTPClient *http.Client
	Output     io.Writer
	Timeout    time.Duration
}

type subscriptionOAuthCallbackResult struct {
	Code string
	Err  error
}

type subscriptionDeviceCode struct {
	VerificationURL string
	UserCode        string
	DeviceAuthID    string
	Interval        time.Duration
}

type subscriptionDeviceTokenPollResponse struct {
	AuthorizationCode string `json:"authorization_code"`
	CodeChallenge     string `json:"code_challenge"`
	CodeVerifier      string `json:"code_verifier"`
}

// RunSubscriptionBrowserLogin は localhost callback を使って OAuth login を実行します。
func RunSubscriptionBrowserLogin(ctx context.Context, options SubscriptionBrowserLoginOptions) (SubscriptionAuthStatus, error) {
	config := options.Config.normalized()
	if _, err := validateSubscriptionOriginatorForRequest(config.Originator); err != nil {
		return ReadSubscriptionAuthStatus(config), err
	}
	timeout := options.Timeout
	if timeout <= 0 {
		timeout = subscriptionLoginTimeout
	}
	loginCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	pkce, err := generateSubscriptionPKCE()
	if err != nil {
		return ReadSubscriptionAuthStatus(config), err
	}
	state, err := randomSubscriptionBase64URL(32)
	if err != nil {
		return ReadSubscriptionAuthStatus(config), err
	}
	redirectURI, callbacks, shutdown, err := startSubscriptionOAuthCallbackServer(loginCtx, config.OAuthPort, state)
	if err != nil {
		return ReadSubscriptionAuthStatus(config), err
	}
	defer shutdown()

	authURL, err := buildSubscriptionAuthorizeURL(config, redirectURI, pkce, state)
	if err != nil {
		return ReadSubscriptionAuthStatus(config), err
	}
	output := options.Output
	if output == nil {
		output = io.Discard
	}
	openBrowser := options.OpenBrowser
	if openBrowser == nil {
		openBrowser = openSubscriptionBrowser
	}
	if err := openBrowser(authURL); err != nil {
		fmt.Fprintln(output, "Open this URL in your browser to sign in with ChatGPT/Codex:")
		fmt.Fprintln(output, authURL)
	} else {
		fmt.Fprintln(output, "Waiting for OAuth callback...")
	}

	select {
	case <-loginCtx.Done():
		return ReadSubscriptionAuthStatus(config), subscriptionAuthError(SubscriptionAuthStateLoginRequired, "openai_subscription OAuth login timed out", loginCtx.Err())
	case result := <-callbacks:
		if result.Err != nil {
			return ReadSubscriptionAuthStatus(config), result.Err
		}
		credential, err := exchangeSubscriptionAuthorizationCode(loginCtx, config, redirectURI, pkce, result.Code, options.HTTPClient)
		if err != nil {
			return ReadSubscriptionAuthStatus(config), err
		}
		if err := SaveSubscriptionCredential(config, credential); err != nil {
			return ReadSubscriptionAuthStatus(config), err
		}
		return ReadSubscriptionAuthStatus(config), nil
	}
}

// RunSubscriptionDeviceLogin は device code OAuth login を実行します。
func RunSubscriptionDeviceLogin(ctx context.Context, options SubscriptionDeviceLoginOptions) (SubscriptionAuthStatus, error) {
	config := options.Config.normalized()
	if _, err := validateSubscriptionOriginatorForRequest(config.Originator); err != nil {
		return ReadSubscriptionAuthStatus(config), err
	}
	timeout := options.Timeout
	if timeout <= 0 {
		timeout = subscriptionDeviceLoginTimeout
	}
	loginCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	output := options.Output
	if output == nil {
		output = io.Discard
	}
	device, err := requestSubscriptionDeviceCode(loginCtx, config, options.HTTPClient)
	if err != nil {
		return ReadSubscriptionAuthStatus(config), err
	}
	fmt.Fprintln(output, "Open this URL in your browser to sign in with ChatGPT/Codex:")
	fmt.Fprintln(output, device.VerificationURL)
	fmt.Fprintln(output, "Enter this one-time code:")
	fmt.Fprintln(output, device.UserCode)

	codeResp, err := pollSubscriptionDeviceAuthorization(loginCtx, config, device, options.HTTPClient)
	if err != nil {
		return ReadSubscriptionAuthStatus(config), err
	}
	pkce := subscriptionPKCE{Verifier: codeResp.CodeVerifier, Challenge: codeResp.CodeChallenge}
	redirectURI := config.Issuer + "/deviceauth/callback"
	credential, err := exchangeSubscriptionAuthorizationCode(loginCtx, config, redirectURI, pkce, codeResp.AuthorizationCode, options.HTTPClient)
	if err != nil {
		return ReadSubscriptionAuthStatus(config), err
	}
	if err := SaveSubscriptionCredential(config, credential); err != nil {
		return ReadSubscriptionAuthStatus(config), err
	}
	return ReadSubscriptionAuthStatus(config), nil
}

func startSubscriptionOAuthCallbackServer(ctx context.Context, port int, expectedState string) (string, <-chan subscriptionOAuthCallbackResult, func(), error) {
	callbacks := make(chan subscriptionOAuthCallbackResult, 1)
	listener, err := listenSubscriptionOAuthCallback(port)
	if err != nil {
		return "", nil, nil, err
	}
	actualPort := port
	if tcpAddr, ok := listener.Addr().(*net.TCPAddr); ok {
		actualPort = tcpAddr.Port
	}
	redirectURI := fmt.Sprintf("http://localhost:%d%s", actualPort, subscriptionOAuthCallbackPath)
	var once sync.Once
	send := func(result subscriptionOAuthCallbackResult) {
		once.Do(func() {
			select {
			case callbacks <- result:
			case <-ctx.Done():
			}
		})
	}
	mux := http.NewServeMux()
	mux.HandleFunc(subscriptionOAuthCallbackPath, func(w http.ResponseWriter, r *http.Request) {
		code, err := validateSubscriptionOAuthCallback(r.URL.Query(), expectedState)
		if err != nil {
			send(subscriptionOAuthCallbackResult{Err: err})
			writeSubscriptionCallbackHTML(w, http.StatusBadRequest, "Authorization failed", err.Error())
			return
		}
		send(subscriptionOAuthCallbackResult{Code: code})
		writeSubscriptionCallbackHTML(w, http.StatusOK, "Authorization successful", "You can close this window and return to XELYON.")
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		http.NotFound(w, nil)
	})
	server := &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() {
		if serveErr := server.Serve(listener); serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			send(subscriptionOAuthCallbackResult{Err: fmt.Errorf("OAuth callback server failed: %w", serveErr)})
		}
	}()
	shutdown := func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}
	go func() {
		<-ctx.Done()
		shutdown()
	}()
	return redirectURI, callbacks, shutdown, nil
}

func listenSubscriptionOAuthCallback(port int) (net.Listener, error) {
	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err == nil {
		return listener, nil
	}
	if port != subscriptionDefaultOAuthPort {
		return nil, fmt.Errorf("failed to listen on OAuth callback port %d: %w", port, err)
	}
	fallback, fallbackErr := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", subscriptionFallbackOAuthPort))
	if fallbackErr != nil {
		return nil, fmt.Errorf("failed to listen on OAuth callback port %d: %w; fallback OAuth callback port %d also failed: %v", port, err, subscriptionFallbackOAuthPort, fallbackErr)
	}
	return fallback, nil
}

func validateSubscriptionOAuthCallback(query url.Values, expectedState string) (string, error) {
	if errParam := strings.TrimSpace(query.Get("error")); errParam != "" {
		description := strings.TrimSpace(query.Get("error_description"))
		if description == "" {
			description = errParam
		}
		return "", subscriptionAuthError(SubscriptionAuthStateLoginRequired, "OAuth returned error: "+RedactSubscriptionSecrets(description), nil)
	}
	if query.Get("state") != expectedState {
		return "", subscriptionAuthError(SubscriptionAuthStateLoginRequired, "invalid OAuth state", nil)
	}
	code := strings.TrimSpace(query.Get("code"))
	if code == "" {
		return "", subscriptionAuthError(SubscriptionAuthStateLoginRequired, "missing authorization code", nil)
	}
	return code, nil
}

func writeSubscriptionCallbackHTML(w http.ResponseWriter, status int, title, message string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	fmt.Fprintf(
		w,
		"<!doctype html><html><head><meta charset=\"utf-8\"><title>%s</title></head><body><h1>%s</h1><p>%s</p></body></html>",
		html.EscapeString(title),
		html.EscapeString(title),
		html.EscapeString(RedactSubscriptionSecrets(message)),
	)
}

func buildSubscriptionAuthorizeURL(config SubscriptionAuthConfig, redirectURI string, pkce subscriptionPKCE, state string) (string, error) {
	originator, err := validateSubscriptionOriginatorForRequest(config.Originator)
	if err != nil {
		return "", err
	}
	base, err := url.Parse(config.oauthAuthorizeURLBase())
	if err != nil {
		return "", err
	}
	query := url.Values{}
	query.Set("response_type", "code")
	query.Set("client_id", config.ClientID)
	query.Set("redirect_uri", redirectURI)
	query.Set("scope", "openid profile email offline_access")
	query.Set("code_challenge", pkce.Challenge)
	query.Set("code_challenge_method", "S256")
	query.Set("id_token_add_organizations", "true")
	query.Set("codex_cli_simplified_flow", "true")
	query.Set("state", state)
	query.Set("originator", originator)
	base.RawQuery = query.Encode()
	return base.String(), nil
}

func exchangeSubscriptionAuthorizationCode(ctx context.Context, config SubscriptionAuthConfig, redirectURI string, pkce subscriptionPKCE, code string, client *http.Client) (SubscriptionCredential, error) {
	originator, err := validateSubscriptionOriginatorForRequest(config.Originator)
	if err != nil {
		return SubscriptionCredential{}, err
	}
	if client == nil {
		client = &http.Client{Timeout: subscriptionHTTPTimeout}
	}
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("redirect_uri", redirectURI)
	form.Set("client_id", config.ClientID)
	form.Set("code_verifier", pkce.Verifier)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, config.oauthTokenURL(), strings.NewReader(form.Encode()))
	if err != nil {
		return SubscriptionCredential{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", subscriptionUserAgent())
	resp, err := client.Do(req)
	if err != nil {
		return SubscriptionCredential{}, err
	}
	defer resp.Body.Close()
	body, err := readSubscriptionLimitedBody(resp.Body)
	if err != nil {
		return SubscriptionCredential{}, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return SubscriptionCredential{}, subscriptionAuthError(SubscriptionAuthStateLoginRequired, fmt.Sprintf("token exchange failed: HTTP %d: %s", resp.StatusCode, RedactSubscriptionSecrets(string(body))), nil)
	}
	var tokens subscriptionTokenResponse
	if err := json.Unmarshal(body, &tokens); err != nil {
		return SubscriptionCredential{}, err
	}
	if strings.TrimSpace(tokens.AccessToken) == "" {
		return SubscriptionCredential{}, subscriptionAuthError(SubscriptionAuthStateMalformed, "token exchange response did not include access_token", nil)
	}
	if strings.TrimSpace(tokens.RefreshToken) == "" {
		return SubscriptionCredential{}, subscriptionAuthError(SubscriptionAuthStateMalformed, "token exchange response did not include refresh_token", nil)
	}
	expiresAt := subscriptionTokenResponseExpiresAt(tokens, time.Now())
	if expiresAt.IsZero() {
		return SubscriptionCredential{}, subscriptionAuthError(SubscriptionAuthStateMalformed, "token exchange response did not include token expiry", nil)
	}
	return SubscriptionCredential{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		AccountID:    ExtractSubscriptionAccountID(tokens.IDToken, tokens.AccessToken),
		ExpiresAt:    expiresAt,
		Issuer:       config.Issuer,
		ClientID:     config.ClientID,
		Originator:   originator,
	}, nil
}

func requestSubscriptionDeviceCode(ctx context.Context, config SubscriptionAuthConfig, client *http.Client) (subscriptionDeviceCode, error) {
	originator, err := validateSubscriptionOriginatorForRequest(config.Originator)
	if err != nil {
		return subscriptionDeviceCode{}, err
	}
	if client == nil {
		client = &http.Client{Timeout: subscriptionHTTPTimeout}
	}
	body, err := json.Marshal(map[string]string{"client_id": config.ClientID})
	if err != nil {
		return subscriptionDeviceCode{}, err
	}
	endpoint := config.Issuer + "/api/accounts/deviceauth/usercode"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return subscriptionDeviceCode{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", subscriptionUserAgent())
	req.Header.Set("originator", originator)
	resp, err := client.Do(req)
	if err != nil {
		return subscriptionDeviceCode{}, err
	}
	defer resp.Body.Close()
	responseBody, err := readSubscriptionLimitedBody(resp.Body)
	if err != nil {
		return subscriptionDeviceCode{}, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return subscriptionDeviceCode{}, subscriptionAuthError(SubscriptionAuthStateLoginRequired, fmt.Sprintf("device code request failed: HTTP %d: %s", resp.StatusCode, RedactSubscriptionSecrets(string(responseBody))), nil)
	}
	return parseSubscriptionDeviceCode(config, responseBody)
}

func parseSubscriptionDeviceCode(config SubscriptionAuthConfig, body []byte) (subscriptionDeviceCode, error) {
	var raw map[string]any
	if err := json.Unmarshal(body, &raw); err != nil {
		return subscriptionDeviceCode{}, err
	}
	deviceAuthID := stringFromMap(raw, "device_auth_id")
	userCode := firstNonEmptyString(stringFromMap(raw, "user_code"), stringFromMap(raw, "usercode"))
	if deviceAuthID == "" || userCode == "" {
		return subscriptionDeviceCode{}, subscriptionAuthError(SubscriptionAuthStateMalformed, "device code response is missing required fields", nil)
	}
	interval := durationSecondsFromMap(raw, "interval")
	if interval <= 0 {
		interval = 5 * time.Second
	}
	verificationURL := firstNonEmptyString(
		stringFromMap(raw, "verification_uri_complete"),
		stringFromMap(raw, "verification_url"),
		stringFromMap(raw, "verification_uri"),
		config.Issuer+"/codex/device",
	)
	return subscriptionDeviceCode{
		VerificationURL: verificationURL,
		UserCode:        userCode,
		DeviceAuthID:    deviceAuthID,
		Interval:        interval,
	}, nil
}

func pollSubscriptionDeviceAuthorization(ctx context.Context, config SubscriptionAuthConfig, device subscriptionDeviceCode, client *http.Client) (subscriptionDeviceTokenPollResponse, error) {
	if client == nil {
		client = &http.Client{Timeout: subscriptionHTTPTimeout}
	}
	endpoint := config.Issuer + "/api/accounts/deviceauth/token"
	for {
		poll, err := pollSubscriptionDeviceAuthorizationOnce(ctx, client, endpoint, config, device)
		if err == nil {
			return poll, nil
		}
		var pending subscriptionDevicePendingError
		if !errors.As(err, &pending) {
			return subscriptionDeviceTokenPollResponse{}, err
		}
		timer := time.NewTimer(device.Interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return subscriptionDeviceTokenPollResponse{}, subscriptionAuthError(SubscriptionAuthStateLoginRequired, "device auth timed out or was cancelled", ctx.Err())
		case <-timer.C:
		}
	}
}

type subscriptionDevicePendingError struct{}

func (subscriptionDevicePendingError) Error() string {
	return "device auth pending"
}

func pollSubscriptionDeviceAuthorizationOnce(ctx context.Context, client *http.Client, endpoint string, config SubscriptionAuthConfig, device subscriptionDeviceCode) (subscriptionDeviceTokenPollResponse, error) {
	originator, err := validateSubscriptionOriginatorForRequest(config.Originator)
	if err != nil {
		return subscriptionDeviceTokenPollResponse{}, err
	}
	body, err := json.Marshal(map[string]string{
		"device_auth_id": device.DeviceAuthID,
		"user_code":      device.UserCode,
	})
	if err != nil {
		return subscriptionDeviceTokenPollResponse{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return subscriptionDeviceTokenPollResponse{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", subscriptionUserAgent())
	req.Header.Set("originator", originator)
	resp, err := client.Do(req)
	if err != nil {
		return subscriptionDeviceTokenPollResponse{}, err
	}
	defer resp.Body.Close()
	responseBody, err := readSubscriptionLimitedBody(resp.Body)
	if err != nil {
		return subscriptionDeviceTokenPollResponse{}, err
	}
	if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusNotFound {
		return subscriptionDeviceTokenPollResponse{}, subscriptionDevicePendingError{}
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return subscriptionDeviceTokenPollResponse{}, subscriptionAuthError(SubscriptionAuthStateLoginRequired, fmt.Sprintf("device auth failed: HTTP %d: %s", resp.StatusCode, RedactSubscriptionSecrets(string(responseBody))), nil)
	}
	var parsed subscriptionDeviceTokenPollResponse
	if err := json.Unmarshal(responseBody, &parsed); err != nil {
		return subscriptionDeviceTokenPollResponse{}, err
	}
	if strings.TrimSpace(parsed.AuthorizationCode) == "" || strings.TrimSpace(parsed.CodeVerifier) == "" {
		return subscriptionDeviceTokenPollResponse{}, subscriptionAuthError(SubscriptionAuthStateMalformed, "device auth response is missing authorization_code or code_verifier", nil)
	}
	return parsed, nil
}

func openSubscriptionBrowser(rawURL string) error {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", rawURL).Start()
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", rawURL).Start()
	default:
		return exec.Command("xdg-open", rawURL).Start()
	}
}

func stringFromMap(values map[string]any, key string) string {
	if value, ok := values[key].(string); ok {
		return strings.TrimSpace(value)
	}
	return ""
}

func durationSecondsFromMap(values map[string]any, key string) time.Duration {
	value, ok := values[key]
	if !ok {
		return 0
	}
	switch typed := value.(type) {
	case float64:
		return time.Duration(typed) * time.Second
	case string:
		seconds, err := strconv.ParseFloat(strings.TrimSpace(typed), 64)
		if err == nil {
			return time.Duration(seconds) * time.Second
		}
	}
	return 0
}
