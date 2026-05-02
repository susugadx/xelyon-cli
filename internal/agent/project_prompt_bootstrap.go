package agent

import "github.com/susugadx/xelyon-cli/internal/config"

type projectPromptBootstrapOptions struct {
	input             string
	showLoadedMessage bool
}

// bootstrapProjectPromptState は project config + project map の初期化を行う。
// showLoadedMessage=true の場合は applyProjectConfig を経由し、従来の UI 表示を維持する。
func bootstrapProjectPromptState(agent *Agent, opts projectPromptBootstrapOptions) {
	if agent == nil {
		return
	}

	if pc := agent.loadProjectConfig(); pc != nil {
		if opts.showLoadedMessage {
			applyProjectConfig(agent, pc)
		} else {
			agent.SystemPrompt = injectProjectConfig(agent.SystemPrompt, pc, opts.input)
			if resolved := config.ResolveFinalChecks(agent.cfg(), pc); resolved != nil {
				current := agent.cfg()
				current.FinalChecks = *resolved
			}
		}
	}
	injectProjectMap(agent, opts.input)
}
