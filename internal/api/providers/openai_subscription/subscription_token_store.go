package openaisubscription

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var (
	// ErrSubscriptionLoginRequired は subscription OAuth token が存在しない状態を表します。
	ErrSubscriptionLoginRequired = errors.New("openai_subscription login required")
	// ErrSubscriptionTokenExpired は local-only status で token が期限切れの状態を表します。
	ErrSubscriptionTokenExpired = errors.New("openai_subscription token expired")
	// ErrSubscriptionRefreshFailed は OAuth refresh が失敗した状態を表します。
	ErrSubscriptionRefreshFailed = errors.New("openai_subscription refresh failed")
	// ErrSubscriptionPermissionUnsafe は auth file/dir の permission が危険な状態を表します。
	ErrSubscriptionPermissionUnsafe = errors.New("openai_subscription auth permission unsafe")
	// ErrSubscriptionTokenMalformed は token store が読めない形式の状態を表します。
	ErrSubscriptionTokenMalformed = errors.New("openai_subscription token store malformed")
)

var subscriptionRefreshMu sync.Mutex

// SubscriptionAuthState は subscription auth の local status 種別です。
type SubscriptionAuthState string

const (
	SubscriptionAuthStateLoggedIn         SubscriptionAuthState = "logged_in"
	SubscriptionAuthStateLoginRequired    SubscriptionAuthState = "login_required"
	SubscriptionAuthStateTokenExpired     SubscriptionAuthState = "token_expired"
	SubscriptionAuthStateRefreshFailed    SubscriptionAuthState = "refresh_failed"
	SubscriptionAuthStatePermissionUnsafe SubscriptionAuthState = "permission_unsafe"
	SubscriptionAuthStateMalformed        SubscriptionAuthState = "token_malformed"
)

// SubscriptionTokenState は token expiry の local 表示状態です。
type SubscriptionTokenState string

const (
	SubscriptionTokenStateValid        SubscriptionTokenState = "valid"
	SubscriptionTokenStateExpiringSoon SubscriptionTokenState = "expiring_soon"
	SubscriptionTokenStateExpired      SubscriptionTokenState = "expired"
	SubscriptionTokenStateUnknown      SubscriptionTokenState = "unknown"
)

// SubscriptionAuthStatus は status/doctor が表示する redacted auth 状態です。
type SubscriptionAuthStatus struct {
	State           SubscriptionAuthState
	TokenState      SubscriptionTokenState
	LoggedIn        bool
	AccountIDMasked string
	ExpiresAt       time.Time
	AuthFilePath    string
	Permission      string
	Endpoint        string
	Originator      string
	Message         string
	Suggestion      string
}

// SubscriptionCredential は request transport が使う OAuth credential DTO です。
type SubscriptionCredential struct {
	AccessToken  string
	RefreshToken string
	AccountID    string
	ExpiresAt    time.Time
	Issuer       string
	ClientID     string
	Originator   string
}

// SubscriptionAuthError は subscription auth 境界の user-facing error です。
type SubscriptionAuthError struct {
	Kind       SubscriptionAuthState
	Message    string
	Suggestion string
	Err        error
}

func (e *SubscriptionAuthError) Error() string {
	if e == nil {
		return ""
	}
	message := RedactSubscriptionSecrets(strings.TrimSpace(e.Message))
	if message == "" && e.Err != nil {
		message = RedactSubscriptionSecrets(e.Err.Error())
	}
	if message == "" {
		message = string(e.Kind)
	}
	if suggestion := strings.TrimSpace(e.Suggestion); suggestion != "" {
		message += "\n" + suggestion
	}
	return message
}

// Unwrap は元 error を返します。
func (e *SubscriptionAuthError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

// Is は subscription auth の sentinel error 判定を提供します。
func (e *SubscriptionAuthError) Is(target error) bool {
	if e == nil {
		return false
	}
	switch target {
	case ErrSubscriptionLoginRequired:
		return e.Kind == SubscriptionAuthStateLoginRequired
	case ErrSubscriptionTokenExpired:
		return e.Kind == SubscriptionAuthStateTokenExpired
	case ErrSubscriptionRefreshFailed:
		return e.Kind == SubscriptionAuthStateRefreshFailed
	case ErrSubscriptionPermissionUnsafe:
		return e.Kind == SubscriptionAuthStatePermissionUnsafe
	case ErrSubscriptionTokenMalformed:
		return e.Kind == SubscriptionAuthStateMalformed
	default:
		return false
	}
}

type subscriptionTokenRecord struct {
	Type            string `json:"type"`
	AccessToken     string `json:"access_token"`
	RefreshToken    string `json:"refresh_token"`
	ExpiresAtUnixMS int64  `json:"expires_at_unix_ms"`
	AccountID       string `json:"account_id,omitempty"`
	CreatedAtUnixMS int64  `json:"created_at_unix_ms"`
	UpdatedAtUnixMS int64  `json:"updated_at_unix_ms"`
	Issuer          string `json:"issuer,omitempty"`
	ClientID        string `json:"client_id,omitempty"`
	Originator      string `json:"originator,omitempty"`
}

type subscriptionTokenResponse struct {
	IDToken         string `json:"id_token"`
	AccessToken     string `json:"access_token"`
	RefreshToken    string `json:"refresh_token"`
	ExpiresIn       int64  `json:"expires_in"`
	ExpiresAtUnixMS int64  `json:"expires_at_unix_ms"`
}

func subscriptionAuthError(kind SubscriptionAuthState, message string, err error) error {
	return &SubscriptionAuthError{
		Kind:       kind,
		Message:    message,
		Suggestion: "Run: " + subscriptionLoginCommand,
		Err:        err,
	}
}

// ReadSubscriptionAuthStatus は token store を local/read-only に確認します。
func ReadSubscriptionAuthStatus(config SubscriptionAuthConfig) SubscriptionAuthStatus {
	config = config.normalized()
	path := config.tokenStorePath()
	status := SubscriptionAuthStatus{
		State:        SubscriptionAuthStateLoginRequired,
		TokenState:   SubscriptionTokenStateUnknown,
		AuthFilePath: path,
		Permission:   "unknown",
		Endpoint:     config.Endpoint,
		Originator:   config.Originator,
		Suggestion:   "Run: " + subscriptionLoginCommand,
	}
	permission := checkSubscriptionAuthPathPermissions(config, true)
	status.Permission = permission.Status
	if permission.Unsafe {
		status.State = SubscriptionAuthStatePermissionUnsafe
		status.Message = strings.Join(permission.Problems, "; ")
		return status
	}
	record, err := readSubscriptionTokenRecord(config)
	if err != nil {
		switch {
		case errors.Is(err, ErrSubscriptionLoginRequired):
			status.Message = "openai_subscription is not logged in."
		case errors.Is(err, ErrSubscriptionTokenMalformed):
			status.State = SubscriptionAuthStateMalformed
			status.Message = "openai_subscription auth file is malformed"
		default:
			status.State = SubscriptionAuthStateMalformed
			status.Message = RedactSubscriptionSecrets(err.Error())
		}
		return status
	}
	credential := credentialFromSubscriptionRecord(record, config)
	status.AccountIDMasked = MaskSubscriptionAccountID(credential.AccountID)
	status.ExpiresAt = credential.ExpiresAt
	status.TokenState = subscriptionTokenState(credential.ExpiresAt, time.Now())
	switch status.TokenState {
	case SubscriptionTokenStateExpired:
		status.State = SubscriptionAuthStateTokenExpired
		status.Message = "openai_subscription token is expired"
	case SubscriptionTokenStateUnknown:
		status.State = SubscriptionAuthStateMalformed
		status.Message = "openai_subscription token expiry is unknown"
	default:
		status.State = SubscriptionAuthStateLoggedIn
		status.LoggedIn = true
		status.Message = "openai_subscription is logged in"
	}
	return status
}

// SaveSubscriptionCredential は OAuth credential を permission-safe に保存します。
func SaveSubscriptionCredential(config SubscriptionAuthConfig, credential SubscriptionCredential) error {
	config = config.normalized()
	credential = credential.withConfigDefaults(config)
	if strings.TrimSpace(credential.AccessToken) == "" {
		return subscriptionAuthError(SubscriptionAuthStateMalformed, "token response did not include access_token", nil)
	}
	if strings.TrimSpace(credential.RefreshToken) == "" {
		return subscriptionAuthError(SubscriptionAuthStateMalformed, "token response did not include refresh_token", nil)
	}
	if credential.ExpiresAt.IsZero() {
		return subscriptionAuthError(SubscriptionAuthStateMalformed, "token response did not include token expiry", nil)
	}
	now := time.Now()
	record := subscriptionTokenRecord{
		Type:            "oauth",
		AccessToken:     credential.AccessToken,
		RefreshToken:    credential.RefreshToken,
		ExpiresAtUnixMS: credential.ExpiresAt.UnixMilli(),
		AccountID:       credential.AccountID,
		CreatedAtUnixMS: now.UnixMilli(),
		UpdatedAtUnixMS: now.UnixMilli(),
		Issuer:          credential.Issuer,
		ClientID:        credential.ClientID,
		Originator:      credential.Originator,
	}
	return writeSubscriptionTokenRecord(config, record)
}

// LoadSubscriptionCredential は token store から OAuth credential を refresh なしで読み込みます。
func LoadSubscriptionCredential(config SubscriptionAuthConfig) (SubscriptionCredential, error) {
	config = config.normalized()
	if permission := checkSubscriptionAuthPathPermissions(config, true); permission.Unsafe {
		return SubscriptionCredential{}, subscriptionAuthError(SubscriptionAuthStatePermissionUnsafe, strings.Join(permission.Problems, "; "), ErrSubscriptionPermissionUnsafe)
	}
	record, err := readSubscriptionTokenRecord(config)
	if err != nil {
		return SubscriptionCredential{}, err
	}
	return credentialFromSubscriptionRecord(record, config), nil
}

// GetSubscriptionCredentialForRequest は必要なら refresh して request 用 credential を返します。
func GetSubscriptionCredentialForRequest(ctx context.Context, config SubscriptionAuthConfig, client *http.Client) (SubscriptionCredential, error) {
	config = config.normalized()
	credential, err := LoadSubscriptionCredential(config)
	if err != nil {
		return SubscriptionCredential{}, err
	}
	if !credential.needsRefresh(time.Now()) {
		return credential, nil
	}
	subscriptionRefreshMu.Lock()
	defer subscriptionRefreshMu.Unlock()

	credential, err = LoadSubscriptionCredential(config)
	if err != nil {
		return SubscriptionCredential{}, err
	}
	if !credential.needsRefresh(time.Now()) {
		return credential, nil
	}
	refreshed, err := refreshSubscriptionCredential(ctx, config, credential, client)
	if err != nil {
		return SubscriptionCredential{}, subscriptionAuthError(SubscriptionAuthStateRefreshFailed, "openai_subscription token refresh failed: "+RedactSubscriptionSecrets(err.Error()), err)
	}
	if err := SaveSubscriptionCredential(config, refreshed); err != nil {
		return SubscriptionCredential{}, err
	}
	return refreshed, nil
}

// LogoutSubscriptionAuth は XELYON の subscription auth file だけを削除します。
func LogoutSubscriptionAuth(config SubscriptionAuthConfig) (bool, error) {
	config = config.normalized()
	if permission := checkSubscriptionAuthPathPermissions(config, false); permission.Unsafe {
		return false, subscriptionAuthError(SubscriptionAuthStatePermissionUnsafe, strings.Join(permission.Problems, "; "), ErrSubscriptionPermissionUnsafe)
	}
	path := config.tokenStorePath()
	info, err := os.Lstat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return false, subscriptionAuthError(SubscriptionAuthStatePermissionUnsafe, "openai_subscription auth file is a symlink", ErrSubscriptionPermissionUnsafe)
	}
	if err := os.Remove(path); err != nil {
		return false, err
	}
	return true, nil
}

func credentialFromSubscriptionRecord(record subscriptionTokenRecord, config SubscriptionAuthConfig) SubscriptionCredential {
	return SubscriptionCredential{
		AccessToken:  strings.TrimSpace(record.AccessToken),
		RefreshToken: strings.TrimSpace(record.RefreshToken),
		AccountID:    strings.TrimSpace(record.AccountID),
		ExpiresAt:    time.UnixMilli(record.ExpiresAtUnixMS),
		Issuer:       firstNonEmptyString(record.Issuer, config.Issuer),
		ClientID:     firstNonEmptyString(record.ClientID, config.ClientID),
		Originator:   strings.TrimSpace(config.Originator),
	}
}

func (c SubscriptionCredential) withConfigDefaults(config SubscriptionAuthConfig) SubscriptionCredential {
	c.AccessToken = strings.TrimSpace(c.AccessToken)
	c.RefreshToken = strings.TrimSpace(c.RefreshToken)
	c.AccountID = strings.TrimSpace(c.AccountID)
	c.Issuer = firstNonEmptyString(c.Issuer, config.Issuer)
	c.ClientID = firstNonEmptyString(c.ClientID, config.ClientID)
	c.Originator = firstNonEmptyString(c.Originator, config.Originator)
	return c
}

func (c SubscriptionCredential) needsRefresh(now time.Time) bool {
	if c.ExpiresAt.IsZero() {
		return true
	}
	return !c.ExpiresAt.After(now.Add(subscriptionRefreshSkew))
}

func readSubscriptionTokenRecord(config SubscriptionAuthConfig) (subscriptionTokenRecord, error) {
	path := config.tokenStorePath()
	file, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return subscriptionTokenRecord{}, subscriptionAuthError(SubscriptionAuthStateLoginRequired, "openai_subscription is not logged in.", ErrSubscriptionLoginRequired)
		}
		return subscriptionTokenRecord{}, err
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, subscriptionMaxTokenFileBytes+1))
	if err != nil {
		return subscriptionTokenRecord{}, err
	}
	if len(data) > subscriptionMaxTokenFileBytes {
		return subscriptionTokenRecord{}, subscriptionAuthError(SubscriptionAuthStateMalformed, "openai_subscription auth file is too large", ErrSubscriptionTokenMalformed)
	}
	var record subscriptionTokenRecord
	if err := json.Unmarshal(data, &record); err != nil {
		return subscriptionTokenRecord{}, subscriptionAuthError(SubscriptionAuthStateMalformed, "openai_subscription auth file is malformed", ErrSubscriptionTokenMalformed)
	}
	if strings.TrimSpace(record.Type) != "oauth" ||
		strings.TrimSpace(record.AccessToken) == "" ||
		strings.TrimSpace(record.RefreshToken) == "" ||
		record.ExpiresAtUnixMS <= 0 {
		return subscriptionTokenRecord{}, subscriptionAuthError(SubscriptionAuthStateMalformed, "openai_subscription auth file is missing required OAuth fields", ErrSubscriptionTokenMalformed)
	}
	return record, nil
}

func writeSubscriptionTokenRecord(config SubscriptionAuthConfig, record subscriptionTokenRecord) error {
	path := config.tokenStorePath()
	if err := ensureSubscriptionAuthDirectory(config); err != nil {
		return err
	}
	if info, err := os.Lstat(path); err == nil && info.Mode()&os.ModeSymlink != 0 {
		return subscriptionAuthError(SubscriptionAuthStatePermissionUnsafe, "openai_subscription auth file is a symlink", ErrSubscriptionPermissionUnsafe)
	}
	data, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	tmp, err := os.CreateTemp(config.AuthDir, "."+subscriptionTokenFileName+".*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(tmpPath)
		}
	}()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmpPath, 0o600); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return err
	}
	cleanup = false
	if err := os.Chmod(path, 0o600); err != nil {
		return err
	}
	_ = syncSubscriptionDirectory(config.AuthDir)
	return nil
}

func ensureSubscriptionAuthDirectory(config SubscriptionAuthConfig) error {
	config = config.normalized()
	authDir := config.AuthDir
	if info, err := os.Lstat(authDir); err == nil && info.Mode()&os.ModeSymlink != 0 {
		return subscriptionAuthError(SubscriptionAuthStatePermissionUnsafe, "openai_subscription auth dir is a symlink", ErrSubscriptionPermissionUnsafe)
	}
	parent := filepath.Dir(authDir)
	if info, err := os.Lstat(parent); err == nil && info.Mode()&os.ModeSymlink != 0 {
		return subscriptionAuthError(SubscriptionAuthStatePermissionUnsafe, "openai_subscription auth parent dir is a symlink", ErrSubscriptionPermissionUnsafe)
	}
	if err := os.MkdirAll(authDir, 0o700); err != nil {
		return err
	}
	if err := os.Chmod(authDir, 0o700); err != nil {
		return err
	}
	if filepath.Base(authDir) == "auth" && filepath.Base(parent) == ".xelyon" {
		_ = os.Chmod(parent, 0o700)
	}
	return nil
}

type subscriptionPermissionReport struct {
	Status   string
	Unsafe   bool
	Problems []string
}

func checkSubscriptionAuthPathPermissions(config SubscriptionAuthConfig, requireFile bool) subscriptionPermissionReport {
	config = config.normalized()
	var problems []string
	authDir := config.AuthDir
	if info, err := os.Lstat(authDir); err == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			problems = append(problems, "auth dir is a symlink")
		} else if !info.IsDir() {
			problems = append(problems, "auth dir is not a directory")
		} else if info.Mode().Perm()&0o077 != 0 {
			problems = append(problems, fmt.Sprintf("auth dir permission is %04o; want 0700", info.Mode().Perm()))
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		problems = append(problems, "auth dir cannot be inspected")
	}
	path := config.tokenStorePath()
	if info, err := os.Lstat(path); err == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			problems = append(problems, "auth file is a symlink")
		} else if !info.Mode().IsRegular() {
			problems = append(problems, "auth file is not a regular file")
		} else if info.Mode().Perm()&0o077 != 0 {
			problems = append(problems, fmt.Sprintf("auth file permission is %04o; want 0600", info.Mode().Perm()))
		}
	} else if requireFile && !errors.Is(err, os.ErrNotExist) {
		problems = append(problems, "auth file cannot be inspected")
	}
	if len(problems) > 0 {
		return subscriptionPermissionReport{Status: "unsafe", Unsafe: true, Problems: problems}
	}
	return subscriptionPermissionReport{Status: "ok"}
}

func refreshSubscriptionCredential(ctx context.Context, config SubscriptionAuthConfig, credential SubscriptionCredential, client *http.Client) (SubscriptionCredential, error) {
	if strings.TrimSpace(credential.RefreshToken) == "" {
		return SubscriptionCredential{}, fmt.Errorf("refresh_token is missing")
	}
	originator, err := validateSubscriptionOriginatorForRequest(config.Originator)
	if err != nil {
		return SubscriptionCredential{}, err
	}
	if client == nil {
		client = &http.Client{Timeout: subscriptionHTTPTimeout}
	}
	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("refresh_token", credential.RefreshToken)
	form.Set("client_id", config.ClientID)
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
		return SubscriptionCredential{}, fmt.Errorf("token refresh failed: HTTP %d: %s", resp.StatusCode, RedactSubscriptionSecrets(string(body)))
	}
	var tokens subscriptionTokenResponse
	if err := json.Unmarshal(body, &tokens); err != nil {
		return SubscriptionCredential{}, err
	}
	if strings.TrimSpace(tokens.AccessToken) == "" {
		return SubscriptionCredential{}, fmt.Errorf("token refresh response did not include access_token")
	}
	now := time.Now()
	expiresAt := subscriptionTokenResponseExpiresAt(tokens, now)
	if expiresAt.IsZero() {
		return SubscriptionCredential{}, fmt.Errorf("token refresh response did not include token expiry")
	}
	accountID := ExtractSubscriptionAccountID(tokens.IDToken, tokens.AccessToken)
	if strings.TrimSpace(accountID) == "" {
		accountID = credential.AccountID
	}
	refreshToken := strings.TrimSpace(tokens.RefreshToken)
	if refreshToken == "" {
		refreshToken = credential.RefreshToken
	}
	return SubscriptionCredential{
		AccessToken:  tokens.AccessToken,
		RefreshToken: refreshToken,
		AccountID:    accountID,
		ExpiresAt:    expiresAt,
		Issuer:       config.Issuer,
		ClientID:     config.ClientID,
		Originator:   originator,
	}, nil
}

func subscriptionTokenResponseExpiresAt(tokens subscriptionTokenResponse, now time.Time) time.Time {
	if tokens.ExpiresAtUnixMS > 0 {
		return time.UnixMilli(tokens.ExpiresAtUnixMS)
	}
	if tokens.ExpiresIn > 0 {
		return now.Add(time.Duration(tokens.ExpiresIn) * time.Second)
	}
	if expiresAt := subscriptionJWTExpiry(tokens.AccessToken, tokens.IDToken); !expiresAt.IsZero() {
		return expiresAt
	}
	return time.Time{}
}

func readSubscriptionLimitedBody(body io.Reader) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(body, subscriptionMaxHTTPBodyBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) > subscriptionMaxHTTPBodyBytes {
		return nil, fmt.Errorf("response body exceeded %d bytes", subscriptionMaxHTTPBodyBytes)
	}
	return data, nil
}

func subscriptionTokenState(expiresAt time.Time, now time.Time) SubscriptionTokenState {
	if expiresAt.IsZero() {
		return SubscriptionTokenStateUnknown
	}
	if !expiresAt.After(now) {
		return SubscriptionTokenStateExpired
	}
	if !expiresAt.After(now.Add(subscriptionRefreshSkew)) {
		return SubscriptionTokenStateExpiringSoon
	}
	return SubscriptionTokenStateValid
}

func syncSubscriptionDirectory(path string) error {
	dir, err := os.Open(path)
	if err != nil {
		return err
	}
	defer dir.Close()
	return dir.Sync()
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
