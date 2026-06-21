package uiconfig

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/configedit"
)

type providerModelsSnapshot struct {
	configs   map[string]config.ProviderModelConfig
	providers []string
}

func buildProviderModelsSnapshot(cfg *config.Config) providerModelsSnapshot {
	providerModels := cfg.ProviderModelsForEdit()
	providers := make([]string, 0, len(providerModels))
	for p := range providerModels {
		providers = append(providers, p)
	}
	sort.Strings(providers)
	return providerModelsSnapshot{
		configs:   providerModels,
		providers: providers,
	}
}

func (e *StructMapEditor) runProviderModels(cfg *config.Config, promptIO PromptIO) (bool, error) {
	out := promptIO.Out

	for {
		snapshot := buildProviderModelsSnapshot(cfg)
		e.renderProviderModelsMenu(out, snapshot)

		input := readConfigEditorChoice(&promptIO)
		done, saved := e.handleProviderModelsChoice(cfg, &promptIO, out, snapshot, input)
		if done {
			return saved, nil
		}
	}
}

func (e *StructMapEditor) renderProviderModelsMenu(out io.Writer, snapshot providerModelsSnapshot) {
	_, _ = fmt.Fprintf(out, "\n%s── provider_models ──────────────────────%s\n\n", colorCyan, colorReset)
	_, _ = fmt.Fprintln(out, "  Configured providers:")

	if len(snapshot.providers) == 0 {
		_, _ = fmt.Fprintf(out, "    %s(empty)%s\n", colorDim, colorReset)
	} else {
		for i, provider := range snapshot.providers {
			model := snapshot.configs[provider].DefaultModel
			_, _ = fmt.Fprintf(out, "    %d. %s: %s\n", i+1, provider, truncateString(model, 25))
		}
	}

	_, _ = fmt.Fprintln(out)
	_, _ = fmt.Fprintln(out, "  [a] Add provider")
	_, _ = fmt.Fprintln(out, "  [1-9] Edit provider")
	_, _ = fmt.Fprintln(out, "  [d] Delete provider")
	_, _ = fmt.Fprintln(out, "  [s] Save and back")
	_, _ = fmt.Fprintln(out, "  [c] Cancel")
	_, _ = fmt.Fprintf(out, "\n%sChoice:%s ", colorCyan, colorReset)
}

func (e *StructMapEditor) handleProviderModelsChoice(cfg *config.Config, promptIO *PromptIO, out io.Writer, snapshot providerModelsSnapshot, input string) (done bool, saved bool) {
	switch input {
	case "a", "add":
		e.addProviderModel(cfg, promptIO, out)
	case "d", "delete":
		e.deleteProviderModel(cfg, promptIO, out, snapshot)
	case "s", "save":
		return true, true
	case "c", "cancel":
		return true, false
	default:
		e.editProviderModel(cfg, promptIO, out, snapshot, input)
	}
	return false, false
}

func (e *StructMapEditor) addProviderModel(cfg *config.Config, promptIO *PromptIO, out io.Writer) {
	_, _ = fmt.Fprint(out, "Enter provider name: ")
	providerInput := readLineWithIO(promptIO)

	existing := cfg.ProviderModelsForEdit()
	if existing == nil {
		existing = make(map[string]config.ProviderModelConfig)
	}

	name, status := configedit.ResolveProviderAddTarget(providerInput, existing)
	switch status {
	case configedit.ProviderAddTargetEmpty:
		return
	case configedit.ProviderAddTargetDuplicate:
		_, _ = fmt.Fprintf(out, "%sProvider already configured: %s%s\n", colorDim, name, colorReset)
		return
	}

	_, _ = fmt.Fprint(out, "Enter default model: ")
	model := strings.TrimSpace(readLineWithIO(promptIO))

	cfg.SetProviderModelsForEdit(configedit.WithAddedProviderModel(existing, name, model))
	_, _ = fmt.Fprintf(out, "%s✓ Added: %s%s\n", colorGreen, name, colorReset)
}

func (e *StructMapEditor) deleteProviderModel(cfg *config.Config, promptIO *PromptIO, out io.Writer, snapshot providerModelsSnapshot) {
	if len(snapshot.providers) == 0 {
		return
	}

	_, _ = fmt.Fprintf(out, "Enter number to delete (1-%d): ", len(snapshot.providers))
	numStr := readLineWithIO(promptIO)
	name, ok := configedit.SelectProviderByInput(numStr, snapshot.providers)
	if !ok {
		return
	}

	cfg.DeleteProviderModelConfig(name)
	_, _ = fmt.Fprintf(out, "%s✓ Deleted: %s%s\n", colorGreen, name, colorReset)
}

func (e *StructMapEditor) editProviderModel(cfg *config.Config, promptIO *PromptIO, out io.Writer, snapshot providerModelsSnapshot, input string) {
	name, ok := configedit.SelectProviderByInput(input, snapshot.providers)
	if !ok {
		return
	}

	current := snapshot.configs[name].DefaultModel
	_, _ = fmt.Fprintf(out, "\nEditing %s (current: %s)\n", name, current)
	_, _ = fmt.Fprint(out, "Enter new default model: ")
	model := strings.TrimSpace(readLineWithIO(promptIO))
	if model == "" {
		return
	}

	cfg.PatchProviderModelConfig(name, func(pm *config.ProviderModelConfig) {
		pm.DefaultModel = model
	})
	_, _ = fmt.Fprintf(out, "%s✓ Updated: %s%s\n", colorGreen, name, colorReset)
}
