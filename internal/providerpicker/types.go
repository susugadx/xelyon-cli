package providerpicker

// ProviderCredentialStatus は provider picker に表示する認証状態を表す。
type ProviderCredentialStatus string

const (
	ProviderCredentialConfigured ProviderCredentialStatus = "configured"
	ProviderCredentialMissingKey ProviderCredentialStatus = "missing key"
	ProviderCredentialLocal      ProviderCredentialStatus = "local"
	ProviderCredentialAWSAuth    ProviderCredentialStatus = "aws auth"
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
