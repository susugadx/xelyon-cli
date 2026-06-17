package cmd

import (
	"github.com/spf13/cobra"
	"github.com/susugadx/xelyon-cli/internal/authcli"
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
			if err := authcli.RunOpenAISubscriptionLogin(cmd.Context(), cmd.OutOrStdout(), openAISubscriptionAuthDeviceFlag); err != nil {
				cmd.SilenceUsage = true
				return err
			}
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
			authcli.ShowOpenAISubscriptionAuthStatus(cmd.OutOrStdout())
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
			if err := authcli.LogoutOpenAISubscription(cmd.OutOrStdout()); err != nil {
				cmd.SilenceUsage = true
				return err
			}
			return nil
		},
	}
}
