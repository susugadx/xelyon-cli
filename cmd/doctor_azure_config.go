package cmd

import (
	"fmt"
	"io"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/providerdiag"
	"gopkg.in/yaml.v3"
)

type azureDoctorConfigSnippetOptions struct {
	Deployment           string
	CatalogModel         string
	JSON                 bool
	Smoke                bool
	ToolSmoke            bool
	Capabilities         bool
	RequiredCapabilities []string
	RetentionSmoke       bool
	PrintRequest         bool
}

type azureDoctorConfigSnippet struct {
	DefaultProvider string                                    `yaml:"default_provider"`
	ProviderModels  map[string]azureDoctorProviderModelConfig `yaml:"provider_models"`
}

type azureDoctorProviderModelConfig struct {
	DefaultModel string `yaml:"default_model"`
	CatalogModel string `yaml:"catalog_model"`
}

func renderAzureDoctorConfigSnippet(w io.Writer, options azureDoctorConfigSnippetOptions) error {
	if options.JSON {
		return fmt.Errorf("--print-config cannot be combined with --json")
	}
	if options.PrintRequest {
		return fmt.Errorf("--print-config cannot be combined with --print-request")
	}
	if options.Smoke || options.ToolSmoke || options.Capabilities || providerdiag.HasRequiredCapabilityRequest(options.RequiredCapabilities) || options.RetentionSmoke {
		return fmt.Errorf("--print-config cannot be combined with --smoke, --tool-smoke, --capabilities, --require-capability, or --retention-smoke")
	}

	deployment := strings.TrimSpace(options.Deployment)
	if deployment == "" {
		return fmt.Errorf("--print-config requires --deployment <azure-deployment>")
	}
	catalogModel := strings.TrimSpace(options.CatalogModel)
	if catalogModel == "" {
		return fmt.Errorf("--print-config requires --catalog-model <underlying-model>")
	}

	snippet := azureDoctorConfigSnippet{
		DefaultProvider: "azure",
		ProviderModels: map[string]azureDoctorProviderModelConfig{
			"azure": {
				DefaultModel: deployment,
				CatalogModel: catalogModel,
			},
		},
	}
	data, err := yaml.Marshal(snippet)
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}
