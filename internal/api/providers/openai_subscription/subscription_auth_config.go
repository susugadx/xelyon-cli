package openaisubscription

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/susugadx/xelyon-cli/internal/version"
)

const (
	subscriptionDefaultIssuerURL       = "https://auth.openai.com"
	subscriptionDefaultEndpointURL     = "https://chatgpt.com/backend-api/codex/responses"
	subscriptionDefaultCompactEndpoint = "https://chatgpt.com/backend-api/codex/responses/compact"
	subscriptionDefaultOAuthPort       = 1455
	// subscriptionFallbackOAuthPort は Codex OAuth client の登録済み redirect URI と同期する。
	subscriptionFallbackOAuthPort = 1457
	subscriptionDefaultClientID   = "app_EMoamEEZ73f0CkXaXp7hrann"
	subscriptionDefaultOriginator = "xelyon"

	subscriptionOAuthCallbackPath  = "/auth/callback"
	subscriptionLoginTimeout       = 5 * time.Minute
	subscriptionDeviceLoginTimeout = 15 * time.Minute
	subscriptionHTTPTimeout        = 30 * time.Second
	subscriptionRefreshSkew        = 60 * time.Second
	subscriptionMaxHTTPBodyBytes   = 512 * 1024
	subscriptionMaxTokenFileBytes  = 64 * 1024
)

const (
	subscriptionIssuerEnv          = "XELYON_OPENAI_SUBSCRIPTION_ISSUER"
	subscriptionEndpointEnv        = "XELYON_OPENAI_SUBSCRIPTION_ENDPOINT"
	subscriptionCompactEndpointEnv = "XELYON_OPENAI_SUBSCRIPTION_COMPACT_ENDPOINT"
	subscriptionClientIDEnv        = "XELYON_OPENAI_SUBSCRIPTION_CLIENT_ID"
	subscriptionOriginatorEnv      = "XELYON_OPENAI_SUBSCRIPTION_ORIGINATOR"
	subscriptionAuthDirEnv         = "XELYON_OPENAI_SUBSCRIPTION_AUTH_DIR"
)

const subscriptionTokenFileName = "openai_subscription.json"

type subscriptionPKCE struct {
	Verifier  string
	Challenge string
}

// SubscriptionAuthConfig は openai_subscription OAuth と token store の設定です。
type SubscriptionAuthConfig struct {
	Issuer          string
	Endpoint        string
	CompactEndpoint string
	ClientID        string
	Originator      string
	AuthDir         string
	OAuthPort       int
}

// DefaultSubscriptionAuthConfig は env override を反映した subscription auth 設定を返します。
func DefaultSubscriptionAuthConfig() SubscriptionAuthConfig {
	authDir := strings.TrimSpace(os.Getenv(subscriptionAuthDirEnv))
	if authDir == "" {
		authDir = defaultSubscriptionAuthDir()
	}
	return SubscriptionAuthConfig{
		Issuer:          subscriptionEnvOrDefault(subscriptionIssuerEnv, subscriptionDefaultIssuerURL),
		Endpoint:        subscriptionEnvOrDefault(subscriptionEndpointEnv, subscriptionDefaultEndpointURL),
		CompactEndpoint: subscriptionOptionalEnvOrDefault(subscriptionCompactEndpointEnv, subscriptionDefaultCompactEndpoint),
		ClientID:        subscriptionEnvOrDefault(subscriptionClientIDEnv, subscriptionDefaultClientID),
		Originator:      subscriptionEnvOrDefault(subscriptionOriginatorEnv, subscriptionDefaultOriginator),
		AuthDir:         authDir,
		OAuthPort:       subscriptionDefaultOAuthPort,
	}
}

func subscriptionEnvOrDefault(name, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}

func subscriptionOptionalEnvOrDefault(name, fallback string) string {
	if value, ok := os.LookupEnv(name); ok {
		return strings.TrimSpace(value)
	}
	return fallback
}

func defaultSubscriptionAuthDir() string {
	if home, err := os.UserHomeDir(); err == nil && strings.TrimSpace(home) != "" {
		return filepath.Join(home, ".xelyon", "auth")
	}
	return filepath.Join(".xelyon", "auth")
}

func (c SubscriptionAuthConfig) normalized() SubscriptionAuthConfig {
	if strings.TrimSpace(c.Issuer) == "" {
		c.Issuer = subscriptionDefaultIssuerURL
	}
	if strings.TrimSpace(c.Endpoint) == "" {
		c.Endpoint = subscriptionDefaultEndpointURL
	}
	if strings.TrimSpace(c.ClientID) == "" {
		c.ClientID = subscriptionDefaultClientID
	}
	if strings.TrimSpace(c.Originator) == "" {
		c.Originator = subscriptionDefaultOriginator
	}
	if strings.TrimSpace(c.AuthDir) == "" {
		c.AuthDir = defaultSubscriptionAuthDir()
	}
	if c.OAuthPort < 0 {
		c.OAuthPort = subscriptionDefaultOAuthPort
	}
	c.Issuer = strings.TrimRight(strings.TrimSpace(c.Issuer), "/")
	c.Endpoint = strings.TrimSpace(c.Endpoint)
	c.CompactEndpoint = strings.TrimSpace(c.CompactEndpoint)
	c.ClientID = strings.TrimSpace(c.ClientID)
	c.Originator = strings.TrimSpace(c.Originator)
	c.AuthDir = filepath.Clean(strings.TrimSpace(c.AuthDir))
	return c
}

func (c SubscriptionAuthConfig) tokenStorePath() string {
	c = c.normalized()
	return filepath.Join(c.AuthDir, subscriptionTokenFileName)
}

func (c SubscriptionAuthConfig) oauthTokenURL() string {
	c = c.normalized()
	return c.Issuer + "/oauth/token"
}

func (c SubscriptionAuthConfig) oauthAuthorizeURLBase() string {
	c = c.normalized()
	return c.Issuer + "/oauth/authorize"
}

func subscriptionUserAgent() string {
	return fmt.Sprintf("xelyon/%s (%s %s)", version.GetVersion(), runtime.GOOS, runtime.GOARCH)
}

func generateSubscriptionPKCE() (subscriptionPKCE, error) {
	verifier, err := randomSubscriptionVerifier(64)
	if err != nil {
		return subscriptionPKCE{}, err
	}
	sum := sha256.Sum256([]byte(verifier))
	return subscriptionPKCE{
		Verifier:  verifier,
		Challenge: base64.RawURLEncoding.EncodeToString(sum[:]),
	}, nil
}

func randomSubscriptionVerifier(length int) (string, error) {
	const chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-._~"
	raw := make([]byte, length)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	var b strings.Builder
	b.Grow(length)
	for _, value := range raw {
		b.WriteByte(chars[int(value)%len(chars)])
	}
	return b.String(), nil
}

func randomSubscriptionBase64URL(bytesLen int) (string, error) {
	raw := make([]byte, bytesLen)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

// ExtractSubscriptionAccountID は id/access token の JWT payload から ChatGPT account ID を抽出します。
func ExtractSubscriptionAccountID(idToken, accessToken string) string {
	for _, token := range []string{idToken, accessToken} {
		claims := parseSubscriptionJWTClaims(token)
		if len(claims) == 0 {
			continue
		}
		if accountID := subscriptionAccountIDFromClaims(claims); accountID != "" {
			return accountID
		}
	}
	return ""
}

func parseSubscriptionJWTClaims(token string) map[string]any {
	parts := strings.Split(strings.TrimSpace(token), ".")
	if len(parts) != 3 {
		return nil
	}
	decoded, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		decoded, err = base64.URLEncoding.DecodeString(parts[1])
		if err != nil {
			return nil
		}
	}
	var claims map[string]any
	if err := json.Unmarshal(decoded, &claims); err != nil {
		return nil
	}
	return claims
}

func subscriptionAccountIDFromClaims(claims map[string]any) string {
	for _, key := range []string{"chatgpt_account_id", "https://api.openai.com/auth.chatgpt_account_id"} {
		if value, ok := claims[key].(string); ok && strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	if nested, ok := claims["https://api.openai.com/auth"].(map[string]any); ok {
		for _, key := range []string{"chatgpt_account_id", "https://api.openai.com/auth.chatgpt_account_id"} {
			if value, ok := nested[key].(string); ok && strings.TrimSpace(value) != "" {
				return strings.TrimSpace(value)
			}
		}
	}
	if organizations, ok := claims["organizations"].([]any); ok && len(organizations) > 0 {
		if first, ok := organizations[0].(map[string]any); ok {
			if value, ok := first["id"].(string); ok && strings.TrimSpace(value) != "" {
				return strings.TrimSpace(value)
			}
		}
	}
	return ""
}

func subscriptionJWTExpiry(tokens ...string) time.Time {
	for _, token := range tokens {
		claims := parseSubscriptionJWTClaims(token)
		if len(claims) == 0 {
			continue
		}
		switch exp := claims["exp"].(type) {
		case float64:
			if exp > 0 {
				return time.Unix(int64(exp), 0)
			}
		case int64:
			if exp > 0 {
				return time.Unix(exp, 0)
			}
		case json.Number:
			value, err := exp.Int64()
			if err == nil && value > 0 {
				return time.Unix(value, 0)
			}
		}
	}
	return time.Time{}
}

// MaskSubscriptionAccountID は status/doctor 表示用に account ID を mask します。
func MaskSubscriptionAccountID(accountID string) string {
	accountID = strings.TrimSpace(accountID)
	if accountID == "" {
		return ""
	}
	if strings.HasPrefix(accountID, "acct_") && len(accountID) > len("acct_")+4 {
		return "acct_****" + accountID[len(accountID)-4:]
	}
	if len(accountID) <= 4 {
		return "****"
	}
	return "****" + accountID[len(accountID)-4:]
}

var (
	subscriptionBearerRegexp               = regexp.MustCompile(`(?i)\bBearer\s+[A-Za-z0-9._~+\-/=]+`)
	subscriptionAuthHeaderRegexp           = regexp.MustCompile(`(?i)(Authorization\s*[:=]\s*)(?:Bearer\s+)?[A-Za-z0-9._~+\-/=]+`)
	subscriptionSecretJSONRegexp           = regexp.MustCompile(`(?i)("(?:access_token|refresh_token|id_token|device_code|code_verifier|authorization_code|auth_code|code|state|token)"\s*:\s*")[^"]*(")`)
	subscriptionSecretKVRegexp             = regexp.MustCompile(`(?i)\b(access_token|refresh_token|id_token|device_code|code_verifier|authorization_code|auth_code|code|state|token)=([^&\s]+)`)
	subscriptionJWTRegexp                  = regexp.MustCompile(`\b[A-Za-z0-9_-]{16,}\.[A-Za-z0-9_-]{16,}\.[A-Za-z0-9_-]{8,}\b`)
	subscriptionAccountIDRegexp            = regexp.MustCompile(`\bacct_[A-Za-z0-9_-]+\b`)
	subscriptionChatGPTAccountHeaderRegexp = regexp.MustCompile(`(?i)(ChatGPT-Account-Id\s*[:=]\s*)[A-Za-z0-9._~+\-/=:-]+`)
)

// RedactSubscriptionSecrets は OAuth token/code/JWT/account ID を user-visible 出力から除去します。
func RedactSubscriptionSecrets(input string) string {
	if input == "" {
		return ""
	}
	out := subscriptionAuthHeaderRegexp.ReplaceAllString(input, `${1}Bearer <redacted>`)
	out = subscriptionBearerRegexp.ReplaceAllString(out, "Bearer <redacted>")
	out = subscriptionChatGPTAccountHeaderRegexp.ReplaceAllString(out, `${1}<redacted>`)
	out = subscriptionSecretJSONRegexp.ReplaceAllString(out, `${1}<redacted>${2}`)
	out = subscriptionSecretKVRegexp.ReplaceAllString(out, `${1}=<redacted>`)
	out = subscriptionJWTRegexp.ReplaceAllString(out, "<redacted-jwt>")
	out = subscriptionAccountIDRegexp.ReplaceAllString(out, "acct_****")
	return out
}

// RedactSubscriptionEndpointForDisplay は user-visible endpoint から URL credential/query/fragment を除去します。
func RedactSubscriptionEndpointForDisplay(endpoint string) string {
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		return ""
	}
	parsed, err := url.Parse(endpoint)
	if err != nil {
		return RedactSubscriptionSecrets(endpoint)
	}
	if parsed.User != nil {
		parsed.User = url.User("redacted")
	}
	if parsed.RawQuery != "" {
		parsed.RawQuery = "redacted"
	}
	if parsed.Fragment != "" || parsed.RawFragment != "" {
		parsed.Fragment = "redacted"
		parsed.RawFragment = ""
	}
	return RedactSubscriptionSecrets(parsed.String())
}
