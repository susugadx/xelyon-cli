package agent

import (
	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/tools"
)

func (a *Agent) providerUsageAttributionCallback() tools.UsageAttributionCallback {
	if a == nil || a.Stats == nil {
		return nil
	}
	return func(provider, model string, usage api.Usage) {
		a.statsMu.Lock()
		defer a.statsMu.Unlock()
		if a.Stats == nil {
			return
		}
		a.Stats.AddUsageForProviderConfig(a.cfg(), provider, model, usage)
	}
}
