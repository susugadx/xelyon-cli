package cmd

import (
	"fmt"
	"io"
	"strings"

	"gopkg.in/yaml.v3"
)

type azureDoctorConfigSnippetOptions struct {
	Deployment   string
	CatalogModel string
	JSON         bool
	Smoke        bool
	ToolSmoke    bool
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
	if options.Smoke || options.ToolSmoke {
		return fmt.Errorf("--print-config cannot be combined with --smoke or --tool-smoke")
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
