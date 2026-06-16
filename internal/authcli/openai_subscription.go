package authcli

import (
	"context"
	"fmt"
	"io"

	openaisubscription "github.com/susugadx/xelyon-cli/internal/api/providers/openai_subscription"
)

// RunOpenAISubscriptionLogin は OpenAI Subscription OAuth login を実行して status を描画する。
func RunOpenAISubscriptionLogin(ctx context.Context, w io.Writer, device bool) error {
	config := openaisubscription.DefaultSubscriptionAuthConfig()
	var (
		status openaisubscription.SubscriptionAuthStatus
		err    error
	)
	if device {
		status, err = openaisubscription.RunSubscriptionDeviceLogin(ctx, openaisubscription.SubscriptionDeviceLoginOptions{
			Config: config,
			Output: w,
		})
	} else {
		status, err = openaisubscription.RunSubscriptionBrowserLogin(ctx, openaisubscription.SubscriptionBrowserLoginOptions{
			Config: config,
			Output: w,
		})
	}
	if err != nil {
		return err
	}
	RenderOpenAISubscriptionAuthStatus(w, status)
	return nil
}

// ShowOpenAISubscriptionAuthStatus はローカル auth status を読み込んで描画する。
func ShowOpenAISubscriptionAuthStatus(w io.Writer) {
	status := openaisubscription.ReadSubscriptionAuthStatus(openaisubscription.DefaultSubscriptionAuthConfig())
	RenderOpenAISubscriptionAuthStatus(w, status)
}

// LogoutOpenAISubscription はローカル auth token を削除して結果を描画する。
func LogoutOpenAISubscription(w io.Writer) error {
	deleted, err := openaisubscription.LogoutSubscriptionAuth(openaisubscription.DefaultSubscriptionAuthConfig())
	if err != nil {
		return err
	}
	if deleted {
		fmt.Fprintln(w, "OpenAI Subscription auth token removed.")
	} else {
		fmt.Fprintln(w, "OpenAI Subscription auth token was not present.")
	}
	return nil
}

// RenderOpenAISubscriptionAuthStatus は OpenAI Subscription auth status を text 表示する。
func RenderOpenAISubscriptionAuthStatus(w io.Writer, status openaisubscription.SubscriptionAuthStatus) {
	fmt.Fprintln(w, "OpenAI Subscription auth")
	fmt.Fprintf(w, "Status: %s\n", status.State)
	if status.AccountIDMasked != "" {
		fmt.Fprintf(w, "Account: %s\n", status.AccountIDMasked)
	} else {
		fmt.Fprintln(w, "Account: unknown")
	}
	fmt.Fprintf(w, "Token: %s\n", status.TokenState)
	if !status.ExpiresAt.IsZero() {
		fmt.Fprintf(w, "Expires: %s\n", status.ExpiresAt.Format("2006-01-02 15:04:05 MST"))
	}
	fmt.Fprintf(w, "Auth file: %s\n", status.AuthFilePath)
	fmt.Fprintf(w, "Permissions: %s\n", status.Permission)
	fmt.Fprintf(w, "Endpoint: %s\n", openaisubscription.RedactSubscriptionEndpointForDisplay(status.Endpoint))
	fmt.Fprintf(w, "Originator: %s\n", status.Originator)
	if status.Message != "" {
		fmt.Fprintf(w, "Message: %s\n", openaisubscription.RedactSubscriptionSecrets(status.Message))
	}
	if status.Suggestion != "" && status.State != openaisubscription.SubscriptionAuthStateLoggedIn {
		fmt.Fprintln(w, status.Suggestion)
	}
}
