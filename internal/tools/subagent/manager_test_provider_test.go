package subagent

import (
	"context"

	"github.com/susugadx/xelyon-cli/internal/api"
)

type managerTestProvider struct {
	name       string
	configKey  string
	responseID string
	cleared    bool
}

func (p *managerTestProvider) Name() string { return p.name }

func (p *managerTestProvider) ProviderConfigKey() string { return p.configKey }

func (p *managerTestProvider) ChatWithTools(context.Context, string, []api.Message, string) (string, error) {
	return "", nil
}

func (p *managerTestProvider) SupportsImages() bool { return false }

func (p *managerTestProvider) ChatWithImage(context.Context, string, []api.Message, string, *api.ImageData, string) (string, error) {
	return "", nil
}

func (p *managerTestProvider) IsFunctionCallingEnabled() bool { return true }

func (p *managerTestProvider) ClearCache() {
	p.cleared = true
	p.responseID = ""
}

func (p *managerTestProvider) SetResponseID(id string) {
	p.responseID = id
}
