package cmd

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"
	openaisubscription "github.com/susugadx/xelyon-cli/internal/api/providers/openai_subscription"
)

var openAISubscriptionAuthDeviceFlag bool

func newOpenAISubscriptionAuthCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "openai-subscription",
		Aliases: []string{"openai_subscription", "chatgpt", "codex-subscription"},
		Short:   "Manage OpenAI Subscription OAuth login",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
	cmd.AddCommand(newOpenAISubscriptionAuthLoginCommand())
	cmd.AddCommand(newOpenAISubscriptionAuthStatusCommand())
	cmd.AddCommand(newOpenAISubscriptionAuthLogoutCommand())
	return cmd
}

func newOpenAISubscriptionAuthLoginCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Sign in with ChatGPT/Codex OAuth",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			config := openaisubscription.DefaultSubscriptionAuthConfig()
			var (
				status openaisubscription.SubscriptionAuthStatus
				err    error
			)
			if openAISubscriptionAuthDeviceFlag {
				status, err = openaisubscription.RunSubscriptionDeviceLogin(cmd.Context(), openaisubscription.SubscriptionDeviceLoginOptions{
					Config: config,
					Output: cmd.OutOrStdout(),
				})
			} else {
				status, err = openaisubscription.RunSubscriptionBrowserLogin(cmd.Context(), openaisubscription.SubscriptionBrowserLoginOptions{
					Config: config,
					Output: cmd.OutOrStdout(),
				})
			}
			if err != nil {
				cmd.SilenceUsage = true
				return err
			}
			renderOpenAISubscriptionAuthStatus(cmd.OutOrStdout(), status)
			return nil
		},
	}
	cmd.Flags().BoolVar(&openAISubscriptionAuthDeviceFlag, "device", false, "Use device-code OAuth login")
	return cmd
}

func newOpenAISubscriptionAuthStatusCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show local OpenAI Subscription auth status",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			status := openaisubscription.ReadSubscriptionAuthStatus(openaisubscription.DefaultSubscriptionAuthConfig())
			renderOpenAISubscriptionAuthStatus(cmd.OutOrStdout(), status)
			return nil
		},
	}
}

func newOpenAISubscriptionAuthLogoutCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Remove local OpenAI Subscription auth token",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			deleted, err := openaisubscription.LogoutSubscriptionAuth(openaisubscription.DefaultSubscriptionAuthConfig())
			if err != nil {
				cmd.SilenceUsage = true
				return err
			}
			if deleted {
				fmt.Fprintln(cmd.OutOrStdout(), "OpenAI Subscription auth token removed.")
			} else {
				fmt.Fprintln(cmd.OutOrStdout(), "OpenAI Subscription auth token was not present.")
			}
			return nil
		},
	}
}

func renderOpenAISubscriptionAuthStatus(w io.Writer, status openaisubscription.SubscriptionAuthStatus) {
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
