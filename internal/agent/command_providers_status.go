package agent

import "fmt"

// handleProvidersCommand は provider credential status を表示する。
func handleProvidersCommand(agent *Agent) bool {
	out := agent.output()
	providers := agent.ProviderCandidates()

	cyan.Fprintln(out, "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	cyan.Fprintln(out, "📡 Provider credential status")
	cyan.Fprintln(out, "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	_, _ = fmt.Fprintln(out)

	for _, provider := range providers {
		icon := "  "
		if provider.Current {
			icon = "✓ "
		}
		label := provider.Label
		if label == "" {
			label = provider.Key
		}
		if provider.Key != "" && provider.Key != label {
			label += " (" + provider.Key + ")"
		}
		status := providerCredentialStatusDisplay(provider.CredentialStatus)
		if provider.Current {
			green.Fprintf(out, "%s%-24s %s\n", icon, label, status)
		} else {
			_, _ = fmt.Fprintf(out, "%s%-24s %s\n", icon, label, status)
		}
	}

	_, _ = fmt.Fprintln(out)
	cyan.Fprintln(out, "Usage: /provider [provider] [model]")
	cyan.Fprintln(out, "TUI: /provider opens the provider/model picker")
	cyan.Fprintln(out, "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	return true
}

func providerCredentialStatusDisplay(status ProviderCredentialStatus) string {
	switch status {
	case ProviderCredentialConfigured:
		return "(credential configured)"
	case ProviderCredentialLoggedIn:
		return "(logged in)"
	case ProviderCredentialLoginRequired:
		return "(login required)"
	case ProviderCredentialLocal:
		return "(local)"
	case ProviderCredentialAWSAuth:
		return "(AWS auth)"
	default:
		return "(credential missing)"
	}
}
