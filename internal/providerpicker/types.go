package providerpicker

// ProviderCredentialStatus は provider picker に表示する認証状態を表す。
type ProviderCredentialStatus string

const (
	ProviderCredentialConfigured    ProviderCredentialStatus = "configured"
	ProviderCredentialLoggedIn      ProviderCredentialStatus = "logged in"
	ProviderCredentialMissingKey    ProviderCredentialStatus = "missing key"
	ProviderCredentialLoginRequired ProviderCredentialStatus = "login required"
	ProviderCredentialLocal         ProviderCredentialStatus = "local"
	ProviderCredentialAWSAuth       ProviderCredentialStatus = "aws auth"
)

// ProviderCandidate は provider picker の 1 行を表す。
type ProviderCandidate struct {
	Key              string
	Label            string
	Current          bool
	CredentialStatus ProviderCredentialStatus
}

// ModelCandidate は model/deployment picker の 1 行を表す。
type ModelCandidate struct {
	Name    string
	Current bool
	Default bool
	Custom  bool
}
