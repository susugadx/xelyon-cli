package openaisubscription

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestGenerateSubscriptionPKCEUsesS256(t *testing.T) {
	pkce, err := generateSubscriptionPKCE()
	if err != nil {
		t.Fatalf("generateSubscriptionPKCE() error = %v", err)
	}
	sum := subscriptionPKCEChallenge(pkce.Verifier)
	if pkce.Challenge != sum {
		t.Fatalf("challenge = %q, want S256 verifier challenge %q", pkce.Challenge, sum)
	}
	if len(pkce.Verifier) < 43 || len(pkce.Verifier) > 128 {
		t.Fatalf("verifier length = %d, want OAuth PKCE length 43..128", len(pkce.Verifier))
	}
}

func TestValidateSubscriptionOAuthCallbackState(t *testing.T) {
	code, err := validateSubscriptionOAuthCallback(url.Values{
		"state": {"expected"},
		"code":  {"auth-code"},
	}, "expected")
	if err != nil {
		t.Fatalf("validateSubscriptionOAuthCallback() error = %v", err)
	}
	if code != "auth-code" {
		t.Fatalf("code = %q, want auth-code", code)
	}
	_, err = validateSubscriptionOAuthCallback(url.Values{
		"state": {"wrong"},
		"code":  {"auth-code"},
	}, "expected")
	if err == nil || !strings.Contains(err.Error(), "invalid OAuth state") {
		t.Fatalf("state mismatch error = %v, want invalid OAuth state", err)
	}
}

func TestExtractSubscriptionAccountIDClaims(t *testing.T) {
	cases := []struct {
		name   string
		claims map[string]any
		want   string
	}{
		{
			name:   "chatgpt account id",
			claims: map[string]any{"chatgpt_account_id": "acct_direct"},
			want:   "acct_direct",
		},
		{
			name:   "flattened claim",
			claims: map[string]any{"https://api.openai.com/auth.chatgpt_account_id": "acct_flat"},
			want:   "acct_flat",
		},
		{
			name:   "nested claim",
			claims: map[string]any{"https://api.openai.com/auth": map[string]any{"chatgpt_account_id": "acct_nested"}},
			want:   "acct_nested",
		},
		{
			name:   "organization fallback",
			claims: map[string]any{"organizations": []any{map[string]any{"id": "org_1234"}}},
			want:   "org_1234",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ExtractSubscriptionAccountID(fakeSubscriptionJWT(t, tc.claims), "")
			if got != tc.want {
				t.Fatalf("ExtractSubscriptionAccountID() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestExtractSubscriptionAccountIDInvalidJWTDoesNotPanic(t *testing.T) {
	if got := ExtractSubscriptionAccountID("not-a-jwt", "also-not-a-jwt"); got != "" {
		t.Fatalf("ExtractSubscriptionAccountID(invalid) = %q, want empty", got)
	}
}

func TestRedactSubscriptionSecrets(t *testing.T) {
	secretJWT := fakeSubscriptionJWT(t, map[string]any{"chatgpt_account_id": "acct_secretabcd", "exp": time.Now().Add(time.Hour).Unix()})
	input := strings.Join([]string{
		"Authorization: Bearer access-secret-token",
		`{"access_token":"access-secret-token","refresh_token":"refresh-secret-token","id_token":"` + secretJWT + `"}`,
		`{"token":"generic-json-token"}`,
		"ChatGPT-Account-Id: acct_secretabcd",
		"https://example.test/callback?code=auth-code&state=state-secret&device_code=device-secret&token=query-token-secret",
	}, "\n")
	redacted := RedactSubscriptionSecrets(input)
	for _, leaked := range []string{"access-secret-token", "refresh-secret-token", secretJWT, "generic-json-token", "acct_secretabcd", "auth-code", "state-secret", "device-secret", "query-token-secret"} {
		if strings.Contains(redacted, leaked) {
			t.Fatalf("redacted output leaked %q:\n%s", leaked, redacted)
		}
	}
	for _, want := range []string{"Bearer <redacted>", "<redacted>"} {
		if !strings.Contains(redacted, want) {
			t.Fatalf("redacted output missing %q:\n%s", want, redacted)
		}
	}
}

func TestSaveSubscriptionCredentialWritesSafePermissionsAndStatusMasksAccount(t *testing.T) {
	config := subscriptionAuthTestConfig(t)
	credential := SubscriptionCredential{
		AccessToken:  "access-token",
		RefreshToken: "refresh-token",
		AccountID:    "acct_123456abcd",
		ExpiresAt:    time.Now().Add(time.Hour),
	}
	if err := SaveSubscriptionCredential(config, credential); err != nil {
		t.Fatalf("SaveSubscriptionCredential() error = %v", err)
	}
	authDirMode := fileMode(t, config.AuthDir)
	if authDirMode != 0o700 {
		t.Fatalf("auth dir mode = %04o, want 0700", authDirMode)
	}
	tokenMode := fileMode(t, config.tokenStorePath())
	if tokenMode != 0o600 {
		t.Fatalf("token mode = %04o, want 0600", tokenMode)
	}
	status := ReadSubscriptionAuthStatus(config)
	if status.State != SubscriptionAuthStateLoggedIn {
		t.Fatalf("status state = %q, want logged_in; message=%s", status.State, status.Message)
	}
	if status.AccountIDMasked != "acct_****abcd" {
		t.Fatalf("masked account = %q, want acct_****abcd", status.AccountIDMasked)
	}
}

func TestSaveSubscriptionCredentialIgnoresStaleFixedTempFile(t *testing.T) {
	config := subscriptionAuthTestConfig(t)
	if err := os.MkdirAll(config.AuthDir, 0o700); err != nil {
		t.Fatalf("MkdirAll(auth dir) error = %v", err)
	}
	staleTemp := filepath.Join(config.AuthDir, "."+subscriptionTokenFileName+".tmp")
	if err := os.WriteFile(staleTemp, []byte("stale temp from crashed process"), 0o600); err != nil {
		t.Fatalf("WriteFile(stale temp) error = %v", err)
	}

	if err := SaveSubscriptionCredential(config, SubscriptionCredential{
		AccessToken:  "access-token",
		RefreshToken: "refresh-token",
		AccountID:    "acct_123456abcd",
		ExpiresAt:    time.Now().Add(time.Hour),
	}); err != nil {
		t.Fatalf("SaveSubscriptionCredential() error = %v", err)
	}

	status := ReadSubscriptionAuthStatus(config)
	if !status.LoggedIn {
		t.Fatalf("status = %+v, want logged in despite stale fixed temp file", status)
	}
	if got := fileMode(t, config.tokenStorePath()); got != 0o600 {
		t.Fatalf("auth file mode = %04o, want 0600", got)
	}
	stale, err := os.ReadFile(staleTemp)
	if err != nil {
		t.Fatalf("ReadFile(stale temp) error = %v", err)
	}
	if string(stale) != "stale temp from crashed process" {
		t.Fatalf("stale temp content = %q, want untouched stale file", stale)
	}
}

func TestSaveSubscriptionCredentialRejectsSymlinkAuthFile(t *testing.T) {
	config := subscriptionAuthTestConfig(t)
	if err := os.MkdirAll(config.AuthDir, 0o700); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	target := filepath.Join(t.TempDir(), "target.json")
	if err := os.WriteFile(target, []byte("{}"), 0o600); err != nil {
		t.Fatalf("WriteFile(target) error = %v", err)
	}
	if err := os.Symlink(target, config.tokenStorePath()); err != nil {
		t.Fatalf("Symlink() error = %v", err)
	}
	err := SaveSubscriptionCredential(config, SubscriptionCredential{
		AccessToken:  "access-token",
		RefreshToken: "refresh-token",
		ExpiresAt:    time.Now().Add(time.Hour),
	})
	if err == nil || !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("SaveSubscriptionCredential() error = %v, want symlink rejection", err)
	}
}

func TestLogoutSubscriptionAuthRejectsSymlinkAuthDir(t *testing.T) {
	config := subscriptionAuthTestConfig(t)
	targetDir := filepath.Join(t.TempDir(), "target-auth")
	if err := os.MkdirAll(targetDir, 0o700); err != nil {
		t.Fatalf("MkdirAll(targetDir) error = %v", err)
	}
	if err := os.Symlink(targetDir, config.AuthDir); err != nil {
		t.Fatalf("Symlink(auth dir) error = %v", err)
	}
	targetToken := filepath.Join(targetDir, subscriptionTokenFileName)
	if err := os.WriteFile(targetToken, []byte("{}"), 0o600); err != nil {
		t.Fatalf("WriteFile(targetToken) error = %v", err)
	}

	deleted, err := LogoutSubscriptionAuth(config)
	if err == nil || !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("LogoutSubscriptionAuth() error = %v, want symlink rejection", err)
	}
	if deleted {
		t.Fatal("LogoutSubscriptionAuth() deleted = true, want false for unsafe auth dir")
	}
	if _, err := os.Stat(targetToken); err != nil {
		t.Fatalf("target token was removed through symlink: %v", err)
	}
}

func TestReadSubscriptionAuthStatusDetectsUnsafePermission(t *testing.T) {
	config := subscriptionAuthTestConfig(t)
	if err := SaveSubscriptionCredential(config, SubscriptionCredential{
		AccessToken:  "access-token",
		RefreshToken: "refresh-token",
		AccountID:    "acct_1234abcd",
		ExpiresAt:    time.Now().Add(time.Hour),
	}); err != nil {
		t.Fatalf("SaveSubscriptionCredential() error = %v", err)
	}
	if err := os.Chmod(config.tokenStorePath(), 0o644); err != nil {
		t.Fatalf("Chmod() error = %v", err)
	}
	status := ReadSubscriptionAuthStatus(config)
	if status.State != SubscriptionAuthStatePermissionUnsafe {
		t.Fatalf("status state = %q, want permission_unsafe", status.State)
	}
	if !strings.Contains(status.Message, "0600") {
		t.Fatalf("status message = %q, want permission detail", status.Message)
	}
}

func TestReadSubscriptionAuthStatusMalformedDoesNotPrintRawBody(t *testing.T) {
	config := subscriptionAuthTestConfig(t)
	if err := os.MkdirAll(config.AuthDir, 0o700); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	raw := `{"access_token":"access-secret-token","refresh_token":"refresh-secret-token",`
	if err := os.WriteFile(config.tokenStorePath(), []byte(raw), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	status := ReadSubscriptionAuthStatus(config)
	if status.State != SubscriptionAuthStateMalformed {
		t.Fatalf("status state = %q, want token_malformed", status.State)
	}
	for _, leaked := range []string{"access-secret-token", "refresh-secret-token"} {
		if strings.Contains(status.Message, leaked) {
			t.Fatalf("status message leaked %q: %s", leaked, status.Message)
		}
	}
}

func TestReadSubscriptionAuthStatusIsLocalOnly(t *testing.T) {
	config := subscriptionAuthTestConfig(t)
	var requests int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requests, 1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()
	config.Issuer = server.URL
	config.Endpoint = server.URL + "/responses"
	status := ReadSubscriptionAuthStatus(config)
	if status.State != SubscriptionAuthStateLoginRequired {
		t.Fatalf("status state = %q, want login_required", status.State)
	}
	if atomic.LoadInt32(&requests) != 0 {
		t.Fatalf("status made %d network requests, want 0", requests)
	}
}

func TestGetSubscriptionCredentialForRequestRefreshesAndUpdatesStore(t *testing.T) {
	config := subscriptionAuthTestConfig(t)
	old := SubscriptionCredential{
		AccessToken:  fakeSubscriptionJWT(t, map[string]any{"exp": time.Now().Add(-time.Minute).Unix()}),
		RefreshToken: "old-refresh",
		AccountID:    "acct_old",
		ExpiresAt:    time.Now().Add(-time.Minute),
	}
	if err := SaveSubscriptionCredential(config, old); err != nil {
		t.Fatalf("SaveSubscriptionCredential(old) error = %v", err)
	}
	var refreshRequests int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&refreshRequests, 1)
		if r.URL.Path != "/oauth/token" {
			t.Fatalf("refresh path = %q, want /oauth/token", r.URL.Path)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatalf("ParseForm() error = %v", err)
		}
		if r.Form.Get("refresh_token") != "old-refresh" {
			t.Fatalf("refresh_token form = %q, want old-refresh", r.Form.Get("refresh_token"))
		}
		newAccess := fakeSubscriptionJWT(t, map[string]any{
			"chatgpt_account_id": "acct_new",
			"exp":                time.Now().Add(time.Hour).Unix(),
		})
		writeJSON(t, w, map[string]any{
			"access_token":  newAccess,
			"refresh_token": "new-refresh",
			"expires_in":    3600,
		})
	}))
	defer server.Close()
	config.Issuer = server.URL
	got, err := GetSubscriptionCredentialForRequest(context.Background(), config, server.Client())
	if err != nil {
		t.Fatalf("GetSubscriptionCredentialForRequest() error = %v", err)
	}
	if got.RefreshToken != "new-refresh" || got.AccountID != "acct_new" {
		t.Fatalf("credential = %#v, want refreshed token/account", got)
	}
	if refreshRequests != 1 {
		t.Fatalf("refresh requests = %d, want 1", refreshRequests)
	}
	stored, err := LoadSubscriptionCredential(config)
	if err != nil {
		t.Fatalf("LoadSubscriptionCredential() error = %v", err)
	}
	if stored.RefreshToken != "new-refresh" || stored.AccountID != "acct_new" {
		t.Fatalf("stored credential = %#v, want refreshed token/account", stored)
	}
}

func TestGetSubscriptionCredentialForRequestRefreshFailureKeepsStoreAndRedacts(t *testing.T) {
	config := subscriptionAuthTestConfig(t)
	old := SubscriptionCredential{
		AccessToken:  fakeSubscriptionJWT(t, map[string]any{"exp": time.Now().Add(-time.Minute).Unix()}),
		RefreshToken: "refresh-secret-token",
		AccountID:    "acct_oldsecret",
		ExpiresAt:    time.Now().Add(-time.Minute),
	}
	if err := SaveSubscriptionCredential(config, old); err != nil {
		t.Fatalf("SaveSubscriptionCredential(old) error = %v", err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprintf(w, `{"error":"invalid","refresh_token":"refresh-secret-token","access_token":"access-secret-token"}`)
	}))
	defer server.Close()
	config.Issuer = server.URL
	_, err := GetSubscriptionCredentialForRequest(context.Background(), config, server.Client())
	if err == nil {
		t.Fatal("GetSubscriptionCredentialForRequest() error = nil, want refresh failure")
	}
	for _, leaked := range []string{"refresh-secret-token", "access-secret-token"} {
		if strings.Contains(err.Error(), leaked) {
			t.Fatalf("refresh error leaked %q: %v", leaked, err)
		}
	}
	if !strings.Contains(err.Error(), subscriptionLoginCommand) {
		t.Fatalf("refresh error = %v, want login suggestion", err)
	}
	stored, loadErr := LoadSubscriptionCredential(config)
	if loadErr != nil {
		t.Fatalf("LoadSubscriptionCredential() error = %v", loadErr)
	}
	if stored.RefreshToken != "refresh-secret-token" {
		t.Fatalf("stored refresh token = %q, want original token retained", stored.RefreshToken)
	}
}

func TestGetSubscriptionCredentialForRequestConcurrentRefreshCollapses(t *testing.T) {
	config := subscriptionAuthTestConfig(t)
	old := SubscriptionCredential{
		AccessToken:  fakeSubscriptionJWT(t, map[string]any{"exp": time.Now().Add(-time.Minute).Unix()}),
		RefreshToken: "old-refresh",
		AccountID:    "acct_old",
		ExpiresAt:    time.Now().Add(-time.Minute),
	}
	if err := SaveSubscriptionCredential(config, old); err != nil {
		t.Fatalf("SaveSubscriptionCredential(old) error = %v", err)
	}
	var refreshRequests int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&refreshRequests, 1)
		newAccess := fakeSubscriptionJWT(t, map[string]any{
			"chatgpt_account_id": "acct_new",
			"exp":                time.Now().Add(time.Hour).Unix(),
		})
		writeJSON(t, w, map[string]any{
			"access_token":  newAccess,
			"refresh_token": "new-refresh",
			"expires_in":    3600,
		})
	}))
	defer server.Close()
	config.Issuer = server.URL
	var wg sync.WaitGroup
	errs := make(chan error, 8)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			credential, err := GetSubscriptionCredentialForRequest(context.Background(), config, server.Client())
			if err != nil {
				errs <- err
				return
			}
			if credential.RefreshToken != "new-refresh" {
				errs <- fmt.Errorf("refresh token = %q, want new-refresh", credential.RefreshToken)
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Error(err)
	}
	if refreshRequests != 1 {
		t.Fatalf("refresh requests = %d, want 1", refreshRequests)
	}
}

func TestRunSubscriptionBrowserLoginStoresTokenWithoutPrintingSecrets(t *testing.T) {
	config := subscriptionAuthTestConfig(t)
	var capturedAuthURL string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/oauth/token" {
			t.Fatalf("token path = %q, want /oauth/token", r.URL.Path)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatalf("ParseForm() error = %v", err)
		}
		if r.Form.Get("grant_type") != "authorization_code" || r.Form.Get("code") != "browser-code" {
			t.Fatalf("token form = %#v", r.Form)
		}
		writeJSON(t, w, map[string]any{
			"id_token":      fakeSubscriptionJWT(t, map[string]any{"chatgpt_account_id": "acct_browserabcd", "exp": time.Now().Add(time.Hour).Unix()}),
			"access_token":  fakeSubscriptionJWT(t, map[string]any{"exp": time.Now().Add(time.Hour).Unix()}),
			"refresh_token": "browser-refresh-secret",
		})
	}))
	defer server.Close()
	config.Issuer = server.URL
	config.OAuthPort = 0
	var out strings.Builder
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	done := make(chan error, 1)
	go func() {
		_, err := RunSubscriptionBrowserLogin(ctx, SubscriptionBrowserLoginOptions{
			Config:     config,
			Output:     &out,
			HTTPClient: server.Client(),
			OpenBrowser: func(rawURL string) error {
				capturedAuthURL = rawURL
				parsed, err := url.Parse(rawURL)
				if err != nil {
					return err
				}
				redirectURI := parsed.Query().Get("redirect_uri")
				state := parsed.Query().Get("state")
				go func() {
					callbackURL := redirectURI + "?state=" + url.QueryEscape(state) + "&code=browser-code"
					_, _ = server.Client().Get(callbackURL)
				}()
				return nil
			},
		})
		done <- err
	}()
	if err := <-done; err != nil {
		t.Fatalf("RunSubscriptionBrowserLogin() error = %v\noutput:\n%s\nauthURL=%s", err, out.String(), capturedAuthURL)
	}
	if !strings.Contains(capturedAuthURL, "originator=xelyon") {
		t.Fatalf("auth URL = %s, want originator=xelyon", capturedAuthURL)
	}
	output := out.String()
	for _, leaked := range []string{"browser-refresh-secret", "browser-code"} {
		if strings.Contains(output, leaked) {
			t.Fatalf("browser login output leaked %q:\n%s", leaked, output)
		}
	}
	status := ReadSubscriptionAuthStatus(config)
	if status.AccountIDMasked != "acct_****abcd" {
		t.Fatalf("status account = %q, want acct_****abcd", status.AccountIDMasked)
	}
}

func TestRunSubscriptionBrowserLoginFallsBackWhenDefaultCallbackPortBusy(t *testing.T) {
	config := subscriptionAuthTestConfig(t)
	config.OAuthPort = subscriptionDefaultOAuthPort
	busy := listenSubscriptionOAuthTestPort(t, subscriptionDefaultOAuthPort)
	defer busy.Close()
	requireSubscriptionOAuthTestPortAvailable(t, subscriptionFallbackOAuthPort)

	wantRedirectURI := fmt.Sprintf("http://localhost:%d%s", subscriptionFallbackOAuthPort, subscriptionOAuthCallbackPath)
	allowedRedirects := map[string]struct{}{
		fmt.Sprintf("http://localhost:%d%s", subscriptionDefaultOAuthPort, subscriptionOAuthCallbackPath):  {},
		fmt.Sprintf("http://localhost:%d%s", subscriptionFallbackOAuthPort, subscriptionOAuthCallbackPath): {},
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/oauth/token" {
			t.Fatalf("token path = %q, want /oauth/token", r.URL.Path)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatalf("ParseForm() error = %v", err)
		}
		redirectURI := r.Form.Get("redirect_uri")
		if _, ok := allowedRedirects[redirectURI]; !ok {
			t.Errorf("redirect_uri = %q, want registered OAuth redirect URI", redirectURI)
			http.Error(w, "invalid redirect_uri", http.StatusBadRequest)
			return
		}
		if redirectURI != wantRedirectURI {
			t.Fatalf("redirect_uri = %q, want fallback redirect URI %q", redirectURI, wantRedirectURI)
		}
		if r.Form.Get("grant_type") != "authorization_code" || r.Form.Get("code") != "fallback-browser-code" {
			t.Fatalf("token form = %#v", r.Form)
		}
		writeJSON(t, w, map[string]any{
			"id_token":      fakeSubscriptionJWT(t, map[string]any{"chatgpt_account_id": "acct_fallbackabcd", "exp": time.Now().Add(time.Hour).Unix()}),
			"access_token":  fakeSubscriptionJWT(t, map[string]any{"exp": time.Now().Add(time.Hour).Unix()}),
			"refresh_token": "fallback-refresh-secret",
		})
	}))
	defer server.Close()
	config.Issuer = server.URL
	var out strings.Builder
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := RunSubscriptionBrowserLogin(ctx, SubscriptionBrowserLoginOptions{
		Config:     config,
		Output:     &out,
		HTTPClient: server.Client(),
		OpenBrowser: func(rawURL string) error {
			parsed, err := url.Parse(rawURL)
			if err != nil {
				return err
			}
			redirectURI := parsed.Query().Get("redirect_uri")
			if redirectURI != wantRedirectURI {
				return fmt.Errorf("redirect_uri = %q, want fallback redirect URI %q", redirectURI, wantRedirectURI)
			}
			state := parsed.Query().Get("state")
			go func() {
				callbackURL := redirectURI + "?state=" + url.QueryEscape(state) + "&code=fallback-browser-code"
				_, _ = server.Client().Get(callbackURL)
			}()
			return nil
		},
	})
	if err != nil {
		t.Fatalf("RunSubscriptionBrowserLogin() error = %v\noutput:\n%s", err, out.String())
	}
	status := ReadSubscriptionAuthStatus(config)
	if status.AccountIDMasked != "acct_****abcd" {
		t.Fatalf("status account = %q, want acct_****abcd", status.AccountIDMasked)
	}
}

func TestStartSubscriptionOAuthCallbackServerDoesNotFallbackFromExplicitBusyPort(t *testing.T) {
	busy, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen() error = %v", err)
	}
	defer busy.Close()
	tcpAddr, ok := busy.Addr().(*net.TCPAddr)
	if !ok {
		t.Fatalf("listener addr = %T, want *net.TCPAddr", busy.Addr())
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_, _, shutdown, err := startSubscriptionOAuthCallbackServer(ctx, tcpAddr.Port, "state")
	if shutdown != nil {
		shutdown()
	}
	if err == nil {
		t.Fatalf("startSubscriptionOAuthCallbackServer() error = nil, want explicit busy port failure")
	}
	if !strings.Contains(err.Error(), fmt.Sprintf("port %d", tcpAddr.Port)) {
		t.Fatalf("error = %q, want explicit port detail", err.Error())
	}
}

func TestStartSubscriptionOAuthCallbackServerFailsWhenRegisteredFallbackPortBusy(t *testing.T) {
	busyDefault := listenSubscriptionOAuthTestPort(t, subscriptionDefaultOAuthPort)
	defer busyDefault.Close()
	busyFallback := listenSubscriptionOAuthTestPort(t, subscriptionFallbackOAuthPort)
	defer busyFallback.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_, _, shutdown, err := startSubscriptionOAuthCallbackServer(ctx, subscriptionDefaultOAuthPort, "state")
	if shutdown != nil {
		shutdown()
	}
	if err == nil {
		t.Fatal("startSubscriptionOAuthCallbackServer() error = nil, want busy registered fallback port failure")
	}
	for _, port := range []int{subscriptionDefaultOAuthPort, subscriptionFallbackOAuthPort} {
		if !strings.Contains(err.Error(), fmt.Sprintf("port %d", port)) {
			t.Fatalf("error = %q, want port %d detail", err.Error(), port)
		}
	}
}

func listenSubscriptionOAuthTestPort(t *testing.T, port int) net.Listener {
	t.Helper()
	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		t.Skipf("OAuth callback port %d is unavailable before test setup: %v", port, err)
	}
	return listener
}

func requireSubscriptionOAuthTestPortAvailable(t *testing.T, port int) {
	t.Helper()
	listener := listenSubscriptionOAuthTestPort(t, port)
	if err := listener.Close(); err != nil {
		t.Fatalf("failed to release OAuth callback port %d after availability probe: %v", port, err)
	}
}

func TestRunSubscriptionBrowserLoginRejectsNonXelyonOriginatorBeforeOpeningBrowser(t *testing.T) {
	config := subscriptionAuthTestConfig(t)
	config.Originator = "opencode"
	var opened bool

	_, err := RunSubscriptionBrowserLogin(context.Background(), SubscriptionBrowserLoginOptions{
		Config: config,
		OpenBrowser: func(string) error {
			opened = true
			return nil
		},
	})
	if err == nil || !strings.Contains(err.Error(), "subscription originator must be xelyon") {
		t.Fatalf("RunSubscriptionBrowserLogin() error = %v, want originator validation", err)
	}
	if opened {
		t.Fatal("RunSubscriptionBrowserLogin() opened browser before originator validation")
	}
}

func TestRunSubscriptionDeviceLoginStoresTokenWithoutPrintingSecrets(t *testing.T) {
	config := subscriptionAuthTestConfig(t)
	var sawOriginator bool
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("originator") == "xelyon" {
			sawOriginator = true
		}
		switch r.URL.Path {
		case "/api/accounts/deviceauth/usercode":
			writeJSON(t, w, map[string]any{
				"device_auth_id": "device-secret-id",
				"user_code":      "USER-CODE",
				"interval":       "1",
			})
		case "/api/accounts/deviceauth/token":
			writeJSON(t, w, map[string]any{
				"authorization_code": "device-auth-code-secret",
				"code_challenge":     "device-challenge",
				"code_verifier":      "device-verifier-secret",
			})
		case "/oauth/token":
			if err := r.ParseForm(); err != nil {
				t.Fatalf("ParseForm() error = %v", err)
			}
			if r.Form.Get("redirect_uri") != server.URL+"/deviceauth/callback" {
				t.Fatalf("redirect_uri = %q, want device callback", r.Form.Get("redirect_uri"))
			}
			writeJSON(t, w, map[string]any{
				"id_token":      fakeSubscriptionJWT(t, map[string]any{"chatgpt_account_id": "acct_deviceabcd", "exp": time.Now().Add(time.Hour).Unix()}),
				"access_token":  fakeSubscriptionJWT(t, map[string]any{"exp": time.Now().Add(time.Hour).Unix()}),
				"refresh_token": "device-refresh-secret",
			})
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()
	config.Issuer = server.URL
	var out strings.Builder
	_, err := RunSubscriptionDeviceLogin(context.Background(), SubscriptionDeviceLoginOptions{
		Config:     config,
		Output:     &out,
		HTTPClient: server.Client(),
	})
	if err != nil {
		t.Fatalf("RunSubscriptionDeviceLogin() error = %v\noutput:\n%s", err, out.String())
	}
	if !sawOriginator {
		t.Fatal("device auth did not send originator=xelyon header")
	}
	output := out.String()
	if !strings.Contains(output, "USER-CODE") || !strings.Contains(output, server.URL+"/codex/device") {
		t.Fatalf("device login output missing user-facing code/url:\n%s", output)
	}
	for _, leaked := range []string{"device-secret-id", "device-auth-code-secret", "device-verifier-secret", "device-refresh-secret"} {
		if strings.Contains(output, leaked) {
			t.Fatalf("device login output leaked %q:\n%s", leaked, output)
		}
	}
	status := ReadSubscriptionAuthStatus(config)
	if status.AccountIDMasked != "acct_****abcd" {
		t.Fatalf("status account = %q, want acct_****abcd", status.AccountIDMasked)
	}
}

func TestSubscriptionDiagnosticsPreviewUsesConfiguredOriginator(t *testing.T) {
	t.Setenv(subscriptionAuthDirEnv, t.TempDir())
	t.Setenv(subscriptionOriginatorEnv, "xelyon-test")

	report := DiagnoseOpenAISubscription(context.Background(), SubscriptionDiagnosticOptions{
		Config:       config.DefaultConfig(),
		Model:        "gpt-5.4-mini",
		CatalogModel: "gpt-5.4-mini",
		PrintRequest: true,
		ToolSmoke:    true,
	})

	if report.Originator != "xelyon-test" {
		t.Fatalf("Originator = %q, want configured originator", report.Originator)
	}
	if report.RequestPreview == nil || len(report.RequestPreview.Requests) != 1 {
		t.Fatalf("RequestPreview = %#v, want one preview request", report.RequestPreview)
	}
	headers := report.RequestPreview.Requests[0].Headers
	if headers["originator"] != "xelyon-test" {
		t.Fatalf("preview originator = %q, want configured originator", headers["originator"])
	}
}

func TestDefaultSubscriptionAuthConfigUsesLiveVerifiedCompactEndpoint(t *testing.T) {
	t.Setenv(subscriptionAuthDirEnv, t.TempDir())

	cfg := DefaultSubscriptionAuthConfig()
	if cfg.CompactEndpoint != subscriptionDefaultCompactEndpoint {
		t.Fatalf("CompactEndpoint = %q, want default %q", cfg.CompactEndpoint, subscriptionDefaultCompactEndpoint)
	}
	if subscriptionCompactEndpointForbidden(cfg.CompactEndpoint) {
		t.Fatalf("default compact endpoint must not be OpenAI Platform API: %q", cfg.CompactEndpoint)
	}
}

func TestSubscriptionDiagnosticsCompactSmokeUsesOAuthTransportAndRedactsOutput(t *testing.T) {
	t.Setenv(subscriptionAuthDirEnv, t.TempDir())
	t.Setenv("OPENAI_API_KEY", "platform-key-must-not-be-used")

	var authorization string
	var originator string
	var raw map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authorization = r.Header.Get("Authorization")
		originator = r.Header.Get("originator")
		if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
			t.Fatalf("decode compact request: %v", err)
		}
		writeJSON(t, w, map[string]any{
			"model": "gpt-5.4-mini",
			"output": []map[string]any{{
				"type": "compacted",
				"data": "opaque-secret-provider-state",
			}},
			"usage": map[string]any{
				"input_tokens":  7,
				"output_tokens": 3,
				"total_tokens":  10,
			},
		})
	}))
	defer server.Close()
	t.Setenv(subscriptionCompactEndpointEnv, server.URL)
	if err := SaveSubscriptionCredential(DefaultSubscriptionAuthConfig(), SubscriptionCredential{
		AccessToken:  "oauth-access-token",
		RefreshToken: "oauth-refresh-token",
		AccountID:    "acct_1234abcd",
		ExpiresAt:    time.Now().Add(time.Hour),
	}); err != nil {
		t.Fatalf("SaveSubscriptionCredential() error = %v", err)
	}

	report := DiagnoseOpenAISubscription(context.Background(), SubscriptionDiagnosticOptions{
		Config:       config.DefaultConfig(),
		Model:        "gpt-5.4-mini",
		CatalogModel: "gpt-5.4-mini",
		RunSmoke:     true,
		CompactSmoke: true,
	})

	if authorization != "Bearer oauth-access-token" || strings.Contains(authorization, "platform-key") {
		t.Fatalf("Authorization = %q, want OAuth bearer and no OPENAI_API_KEY", authorization)
	}
	if originator != "xelyon" {
		t.Fatalf("originator = %q, want xelyon", originator)
	}
	if raw["model"] != "gpt-5.4-mini" {
		t.Fatalf("compact model = %#v, want gpt-5.4-mini", raw["model"])
	}
	compactSmoke := subscriptionDiagnosticTestCheck(t, report.Checks, "compact_smoke")
	if compactSmoke.Status != DiagnosticStatusOK {
		t.Fatalf("compact_smoke check = %+v, want ok", compactSmoke)
	}
	if report.Smoke == nil || len(report.Smoke.Requests) != 1 || !report.Smoke.Requests[0].UsageObserved {
		t.Fatalf("Smoke = %#v, want compact request with observed usage", report.Smoke)
	}
	if report.Smoke.Route != diagnosticRouteSubscriptionCompact {
		t.Fatalf("compact smoke summary route = %q, want %q", report.Smoke.Route, diagnosticRouteSubscriptionCompact)
	}
	if report.Smoke.Requests[0].Route != diagnosticRouteSubscriptionCompact {
		t.Fatalf("compact smoke route = %q, want %q", report.Smoke.Requests[0].Route, diagnosticRouteSubscriptionCompact)
	}
	encoded, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("Marshal(report) error = %v", err)
	}
	if strings.Contains(string(encoded), "opaque-secret-provider-state") {
		t.Fatalf("diagnostic report leaked compacted opaque state:\n%s", string(encoded))
	}
}

func TestSubscriptionDiagnosticsCompactSmokeDisabledEndpointWarnsAndSkips(t *testing.T) {
	t.Setenv(subscriptionAuthDirEnv, t.TempDir())
	t.Setenv(subscriptionCompactEndpointEnv, "")
	if err := SaveSubscriptionCredential(DefaultSubscriptionAuthConfig(), SubscriptionCredential{
		AccessToken:  "oauth-access-token",
		RefreshToken: "oauth-refresh-token",
		ExpiresAt:    time.Now().Add(time.Hour),
	}); err != nil {
		t.Fatalf("SaveSubscriptionCredential() error = %v", err)
	}

	report := DiagnoseOpenAISubscription(context.Background(), SubscriptionDiagnosticOptions{
		Config:       config.DefaultConfig(),
		Model:        "gpt-5.4-mini",
		CatalogModel: "gpt-5.4-mini",
		RunSmoke:     true,
		CompactSmoke: true,
	})

	compactAPI := subscriptionDiagnosticTestCheck(t, report.Checks, "compact_api")
	if compactAPI.Status != DiagnosticStatusWarn {
		t.Fatalf("compact_api check = %+v, want warn", compactAPI)
	}
	compactSmoke := subscriptionDiagnosticTestCheck(t, report.Checks, "compact_smoke")
	if compactSmoke.Status != DiagnosticStatusWarn {
		t.Fatalf("compact_smoke check = %+v, want warn", compactSmoke)
	}
	if report.Smoke == nil || len(report.Smoke.Requests) != 1 || !report.Smoke.Requests[0].Skipped {
		t.Fatalf("Smoke = %#v, want skipped compact request", report.Smoke)
	}
}

func TestSubscriptionDiagnosticsCompactInvalidEndpointFailsAndPreviewSkips(t *testing.T) {
	t.Setenv(subscriptionAuthDirEnv, t.TempDir())
	t.Setenv(subscriptionCompactEndpointEnv, "ftp://chatgpt.example.test/backend-api/codex/responses/compact")

	report := DiagnoseOpenAISubscription(context.Background(), SubscriptionDiagnosticOptions{
		Config:       config.DefaultConfig(),
		Model:        "gpt-5.4-mini",
		CatalogModel: "gpt-5.4-mini",
		CompactSmoke: true,
		PrintRequest: true,
	})

	compactAPI := subscriptionDiagnosticTestCheck(t, report.Checks, "compact_api")
	if compactAPI.Status != DiagnosticStatusFail {
		t.Fatalf("compact_api check = %+v, want fail", compactAPI)
	}
	if !strings.Contains(compactAPI.Detail, "must use http or https") {
		t.Fatalf("compact_api detail = %q, want scheme validation", compactAPI.Detail)
	}
	if report.RequestPreview == nil || len(report.RequestPreview.Requests) != 1 {
		t.Fatalf("RequestPreview = %#v, want one skipped compact preview", report.RequestPreview)
	}
	request := report.RequestPreview.Requests[0]
	if !request.Skipped || !strings.Contains(request.SkipReason, "must use http or https") {
		t.Fatalf("compact preview = %+v, want skipped invalid endpoint reason", request)
	}
}

func TestSubscriptionDiagnosticsCompactPrintRequestUsesCompactRoute(t *testing.T) {
	t.Setenv(subscriptionAuthDirEnv, t.TempDir())
	t.Setenv(subscriptionCompactEndpointEnv, "https://chatgpt.example.test/backend-api/codex/responses/compact")

	report := DiagnoseOpenAISubscription(context.Background(), SubscriptionDiagnosticOptions{
		Config:       config.DefaultConfig(),
		Model:        "gpt-5.4-mini",
		CatalogModel: "gpt-5.4-mini",
		PrintRequest: true,
		CompactSmoke: true,
	})

	if report.RequestPreview == nil || len(report.RequestPreview.Requests) != 1 {
		t.Fatalf("RequestPreview = %#v, want compact preview", report.RequestPreview)
	}
	request := report.RequestPreview.Requests[0]
	if request.Route != diagnosticRouteSubscriptionCompact || request.URL != "https://chatgpt.example.test/backend-api/codex/responses/compact" {
		t.Fatalf("compact preview request = %+v, want subscription compact route/url", request)
	}
}

func TestSubscriptionRequestUsesCurrentConfigOriginatorNotTokenFile(t *testing.T) {
	t.Setenv(subscriptionAuthDirEnv, t.TempDir())
	t.Setenv(subscriptionOriginatorEnv, "xelyon")

	saveConfig := DefaultSubscriptionAuthConfig()
	saveConfig.Originator = "stale-token-originator"
	if err := SaveSubscriptionCredential(saveConfig, SubscriptionCredential{
		AccessToken:  "oauth-access-token",
		RefreshToken: "oauth-refresh-token",
		AccountID:    "acct_1234abcd",
		ExpiresAt:    time.Now().Add(time.Hour),
		Originator:   "stale-token-originator",
	}); err != nil {
		t.Fatalf("SaveSubscriptionCredential() error = %v", err)
	}

	provider := NewSubscription()
	req, err := provider.prepareSubscriptionResponsesRequest(context.Background(), "https://chatgpt.example.test/backend-api/codex/responses", []byte(`{}`))
	if err != nil {
		t.Fatalf("prepareSubscriptionResponsesRequest() error = %v", err)
	}
	if got := req.Header.Get("originator"); got != "xelyon" {
		t.Fatalf("request originator = %q, want current config originator xelyon", got)
	}

	status := ReadSubscriptionAuthStatus(DefaultSubscriptionAuthConfig())
	if status.Originator != "xelyon" {
		t.Fatalf("status originator = %q, want current config originator xelyon", status.Originator)
	}
	loaded, err := LoadSubscriptionCredential(DefaultSubscriptionAuthConfig())
	if err != nil {
		t.Fatalf("LoadSubscriptionCredential() error = %v", err)
	}
	if loaded.Originator != "xelyon" {
		t.Fatalf("loaded credential originator = %q, want current config originator xelyon", loaded.Originator)
	}
}

func TestSubscriptionRequestRejectsConfiguredNonXelyonOriginatorBeforeAuth(t *testing.T) {
	t.Setenv(subscriptionAuthDirEnv, t.TempDir())
	t.Setenv(subscriptionOriginatorEnv, "opencode")

	provider := NewSubscription()
	req, err := provider.prepareSubscriptionResponsesRequest(context.Background(), "https://chatgpt.example.test/backend-api/codex/responses", []byte(`{}`))
	if err == nil || !strings.Contains(err.Error(), "subscription originator must be xelyon") {
		t.Fatalf("prepareSubscriptionResponsesRequest() error = %v, want originator validation", err)
	}
	if req != nil {
		t.Fatalf("prepareSubscriptionResponsesRequest() request = %#v, want nil", req)
	}
	if strings.Contains(err.Error(), "not logged in") {
		t.Fatalf("prepareSubscriptionResponsesRequest() error = %v, want originator validation before auth", err)
	}
}

func TestSubscriptionDiagnosticsEndpointRejectsUnsupportedSchemeAndSkipsSmoke(t *testing.T) {
	t.Setenv(subscriptionAuthDirEnv, t.TempDir())
	t.Setenv(subscriptionEndpointEnv, "ftp://chatgpt.example.test/backend-api/codex/responses")
	if err := SaveSubscriptionCredential(DefaultSubscriptionAuthConfig(), SubscriptionCredential{
		AccessToken:  "oauth-access-token",
		RefreshToken: "oauth-refresh-token",
		ExpiresAt:    time.Now().Add(time.Hour),
	}); err != nil {
		t.Fatalf("SaveSubscriptionCredential() error = %v", err)
	}

	report := DiagnoseOpenAISubscription(context.Background(), SubscriptionDiagnosticOptions{
		Config:       config.DefaultConfig(),
		Model:        "gpt-5.4-mini",
		CatalogModel: "gpt-5.4-mini",
		RunSmoke:     true,
	})

	endpoint := subscriptionDiagnosticTestCheck(t, report.Checks, "endpoint")
	if endpoint.Status != DiagnosticStatusFail {
		t.Fatalf("endpoint check = %+v, want fail", endpoint)
	}
	if !strings.Contains(endpoint.Detail, "must use http or https") {
		t.Fatalf("endpoint detail = %q, want scheme validation", endpoint.Detail)
	}
	smoke := subscriptionDiagnosticTestCheck(t, report.Checks, "smoke")
	if smoke.Status != DiagnosticStatusWarn || !strings.Contains(smoke.Detail, "endpoint") {
		t.Fatalf("smoke check = %+v, want readiness skip for endpoint", smoke)
	}
	if report.Smoke != nil {
		t.Fatalf("Smoke = %#v, want nil when readiness failed", report.Smoke)
	}
}

func TestSubscriptionDiagnosticsSmokeSkipsOnOriginatorFailure(t *testing.T) {
	t.Setenv(subscriptionAuthDirEnv, t.TempDir())
	t.Setenv(subscriptionOriginatorEnv, "opencode")
	var requests int
	server := mockAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		requests++
		w.WriteHeader(http.StatusOK)
	})
	t.Setenv(subscriptionEndpointEnv, server.URL)
	if err := SaveSubscriptionCredential(DefaultSubscriptionAuthConfig(), SubscriptionCredential{
		AccessToken:  "oauth-access-token",
		RefreshToken: "oauth-refresh-token",
		ExpiresAt:    time.Now().Add(time.Hour),
	}); err != nil {
		t.Fatalf("SaveSubscriptionCredential() error = %v", err)
	}

	report := DiagnoseOpenAISubscription(context.Background(), SubscriptionDiagnosticOptions{
		Config:       config.DefaultConfig(),
		Model:        "gpt-5.4-mini",
		CatalogModel: "gpt-5.4-mini",
		RunSmoke:     true,
	})

	originator := subscriptionDiagnosticTestCheck(t, report.Checks, "originator")
	if originator.Status != DiagnosticStatusFail {
		t.Fatalf("originator check = %+v, want fail", originator)
	}
	smoke := subscriptionDiagnosticTestCheck(t, report.Checks, "smoke")
	if smoke.Status != DiagnosticStatusWarn || !strings.Contains(smoke.Detail, "originator") {
		t.Fatalf("smoke check = %+v, want readiness skip for originator", smoke)
	}
	if report.Smoke != nil {
		t.Fatalf("Smoke = %#v, want nil when readiness failed", report.Smoke)
	}
	if requests != 0 {
		t.Fatalf("smoke sent %d requests, want 0 when originator readiness failed", requests)
	}
}

func TestSubscriptionDiagnosticsUnsafeAuthPermissionFailsAndSkipsSmoke(t *testing.T) {
	t.Setenv(subscriptionAuthDirEnv, t.TempDir())
	var requests int
	server := mockAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		requests++
		w.WriteHeader(http.StatusOK)
	})
	t.Setenv(subscriptionEndpointEnv, server.URL)
	authConfig := DefaultSubscriptionAuthConfig()
	if err := SaveSubscriptionCredential(authConfig, SubscriptionCredential{
		AccessToken:  "oauth-access-token",
		RefreshToken: "oauth-refresh-token",
		ExpiresAt:    time.Now().Add(time.Hour),
	}); err != nil {
		t.Fatalf("SaveSubscriptionCredential() error = %v", err)
	}
	if err := os.Chmod(authConfig.tokenStorePath(), 0o644); err != nil {
		t.Fatalf("Chmod() error = %v", err)
	}

	report := DiagnoseOpenAISubscription(context.Background(), SubscriptionDiagnosticOptions{
		Config:       config.DefaultConfig(),
		Model:        "gpt-5.4-mini",
		CatalogModel: "gpt-5.4-mini",
		RunSmoke:     true,
	})

	auth := subscriptionDiagnosticTestCheck(t, report.Checks, "auth")
	if auth.Status != DiagnosticStatusFail {
		t.Fatalf("auth check = %+v, want fail for unsafe permissions", auth)
	}
	if !report.HasFailures() {
		t.Fatal("report.HasFailures() = false, want true for unsafe auth permissions")
	}
	smoke := subscriptionDiagnosticTestCheck(t, report.Checks, "smoke")
	if smoke.Status != DiagnosticStatusWarn || !strings.Contains(smoke.Detail, "auth") {
		t.Fatalf("smoke check = %+v, want readiness skip for auth", smoke)
	}
	if report.Smoke != nil {
		t.Fatalf("Smoke = %#v, want nil when auth readiness failed", report.Smoke)
	}
	if requests != 0 {
		t.Fatalf("smoke sent %d requests, want 0 when auth permissions are unsafe", requests)
	}
}

func TestSubscriptionDiagnosticsSmokeRefreshesExpiredToken(t *testing.T) {
	t.Setenv(subscriptionAuthDirEnv, t.TempDir())
	t.Setenv("OPENAI_API_KEY", "platform-key-must-not-be-used")

	refreshedAccessToken := fakeSubscriptionJWT(t, map[string]any{
		"chatgpt_account_id": "acct_new",
		"exp":                time.Now().Add(time.Hour).Unix(),
	})
	var refreshRequests int32
	var responsesRequests int32
	var authorization string
	server := mockAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/oauth/token":
			atomic.AddInt32(&refreshRequests, 1)
			if err := r.ParseForm(); err != nil {
				t.Fatalf("ParseForm() error = %v", err)
			}
			if r.Form.Get("refresh_token") != "old-refresh" {
				t.Fatalf("refresh_token form = %q, want old-refresh", r.Form.Get("refresh_token"))
			}
			writeJSON(t, w, map[string]any{
				"access_token":  refreshedAccessToken,
				"refresh_token": "new-refresh",
				"expires_in":    3600,
			})
		case "/responses":
			atomic.AddInt32(&responsesRequests, 1)
			authorization = r.Header.Get("Authorization")
			w.Header().Set("Content-Type", "text/event-stream")
			_, _ = w.Write([]byte(strings.Join([]string{
				`data: {"type":"response.created","response":{"id":"resp_subscription_smoke"}}`,
				``,
				`data: {"type":"response.output_text.delta","delta":"refreshed smoke ok"}`,
				``,
				`data: {"type":"response.completed","response":{"usage":{"input_tokens":3,"output_tokens":2}}}`,
				``,
				`data: [DONE]`,
			}, "\n")))
		default:
			t.Fatalf("unexpected path = %q", r.URL.Path)
		}
	})
	t.Setenv(subscriptionIssuerEnv, server.URL)
	t.Setenv(subscriptionEndpointEnv, server.URL+"/responses")
	if err := SaveSubscriptionCredential(DefaultSubscriptionAuthConfig(), SubscriptionCredential{
		AccessToken:  fakeSubscriptionJWT(t, map[string]any{"exp": time.Now().Add(-time.Minute).Unix()}),
		RefreshToken: "old-refresh",
		AccountID:    "acct_old",
		ExpiresAt:    time.Now().Add(-time.Minute),
	}); err != nil {
		t.Fatalf("SaveSubscriptionCredential() error = %v", err)
	}

	report := DiagnoseOpenAISubscription(context.Background(), SubscriptionDiagnosticOptions{
		Config:       config.DefaultConfig(),
		Model:        "gpt-5.4-mini",
		CatalogModel: "gpt-5.4-mini",
		RunSmoke:     true,
	})

	auth := subscriptionDiagnosticTestCheck(t, report.Checks, "auth")
	if auth.Status != DiagnosticStatusWarn {
		t.Fatalf("auth check = %+v, want warn for refreshable expired token", auth)
	}
	smoke := subscriptionDiagnosticTestCheck(t, report.Checks, "smoke")
	if smoke.Status != DiagnosticStatusOK {
		t.Fatalf("smoke check = %+v, want ok after refresh", smoke)
	}
	if report.Smoke == nil || len(report.Smoke.Requests) != 1 {
		t.Fatalf("Smoke = %#v, want one smoke request", report.Smoke)
	}
	if atomic.LoadInt32(&refreshRequests) != 1 {
		t.Fatalf("refresh requests = %d, want 1", refreshRequests)
	}
	if atomic.LoadInt32(&responsesRequests) != 1 {
		t.Fatalf("responses requests = %d, want 1", responsesRequests)
	}
	if authorization != "Bearer "+refreshedAccessToken {
		t.Fatalf("Authorization = %q, want refreshed OAuth bearer", authorization)
	}
	if strings.Contains(authorization, "platform-key") {
		t.Fatalf("Authorization used OPENAI_API_KEY: %q", authorization)
	}
	status := ReadSubscriptionAuthStatus(DefaultSubscriptionAuthConfig())
	if status.State != SubscriptionAuthStateLoggedIn {
		t.Fatalf("status state = %q, want logged_in after smoke refresh", status.State)
	}
}

func TestSubscriptionDiagnosticsCompactSmokeSkipsOnCompactEndpointFailure(t *testing.T) {
	t.Setenv(subscriptionAuthDirEnv, t.TempDir())
	t.Setenv(subscriptionCompactEndpointEnv, "ftp://chatgpt.example.test/backend-api/codex/responses/compact")
	if err := SaveSubscriptionCredential(DefaultSubscriptionAuthConfig(), SubscriptionCredential{
		AccessToken:  "oauth-access-token",
		RefreshToken: "oauth-refresh-token",
		ExpiresAt:    time.Now().Add(time.Hour),
	}); err != nil {
		t.Fatalf("SaveSubscriptionCredential() error = %v", err)
	}

	report := DiagnoseOpenAISubscription(context.Background(), SubscriptionDiagnosticOptions{
		Config:       config.DefaultConfig(),
		Model:        "gpt-5.4-mini",
		CatalogModel: "gpt-5.4-mini",
		RunSmoke:     true,
		CompactSmoke: true,
	})

	compactAPI := subscriptionDiagnosticTestCheck(t, report.Checks, "compact_api")
	if compactAPI.Status != DiagnosticStatusFail {
		t.Fatalf("compact_api check = %+v, want fail", compactAPI)
	}
	smoke := subscriptionDiagnosticTestCheck(t, report.Checks, "smoke")
	if smoke.Status != DiagnosticStatusWarn || !strings.Contains(smoke.Detail, "compact_api") {
		t.Fatalf("smoke check = %+v, want readiness skip for compact_api", smoke)
	}
	if report.Smoke != nil {
		t.Fatalf("Smoke = %#v, want nil when readiness failed", report.Smoke)
	}
}

func subscriptionDiagnosticTestCheck(t *testing.T, checks []DiagnosticCheck, name string) DiagnosticCheck {
	t.Helper()
	for _, check := range checks {
		if check.Name == name {
			return check
		}
	}
	t.Fatalf("check %q not found in %#v", name, checks)
	return DiagnosticCheck{}
}

func subscriptionAuthTestConfig(t *testing.T) SubscriptionAuthConfig {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "auth")
	return SubscriptionAuthConfig{
		Issuer:     "https://auth.example.test",
		Endpoint:   "https://chatgpt.example.test/backend-api/codex/responses",
		ClientID:   "test-client",
		Originator: "xelyon",
		AuthDir:    dir,
		OAuthPort:  0,
	}
}

func fakeSubscriptionJWT(t *testing.T, claims map[string]any) string {
	t.Helper()
	header, err := json.Marshal(map[string]any{"alg": "none", "typ": "JWT"})
	if err != nil {
		t.Fatalf("marshal header: %v", err)
	}
	payload, err := json.Marshal(claims)
	if err != nil {
		t.Fatalf("marshal claims: %v", err)
	}
	return base64.RawURLEncoding.EncodeToString(header) + "." +
		base64.RawURLEncoding.EncodeToString(payload) + ".signature"
}

func subscriptionPKCEChallenge(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func fileMode(t *testing.T, path string) os.FileMode {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat(%s) error = %v", path, err)
	}
	return info.Mode().Perm()
}

func writeJSON(t *testing.T, w http.ResponseWriter, value any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(value); err != nil {
		t.Fatalf("Encode() error = %v", err)
	}
}
