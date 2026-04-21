package config

import "testing"

func TestDetectLoaderSections(t *testing.T) {
	tests := []struct {
		name       string
		yaml       string
		wantLSP    bool
		wantServer bool
	}{
		{
			name:       "lsp section with servers",
			yaml:       "lsp:\n  servers: {}\n",
			wantLSP:    true,
			wantServer: true,
		},
		{
			name:       "lsp section without servers",
			yaml:       "lsp:\n  enabled: false\n",
			wantLSP:    true,
			wantServer: false,
		},
		{
			name:       "no lsp section",
			yaml:       "default_provider: openai\n",
			wantLSP:    false,
			wantServer: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := detectLoaderSections([]byte(tt.yaml))
			if got.lspSectionExists != tt.wantLSP {
				t.Fatalf("lspSectionExists = %v, want %v", got.lspSectionExists, tt.wantLSP)
			}
			if got.lspServersExists != tt.wantServer {
				t.Fatalf("lspServersExists = %v, want %v", got.lspServersExists, tt.wantServer)
			}
		})
	}
}

func TestDefaultConfigForLoad_ClearsLSPServersOnlyWhenExplicitlyPresent(t *testing.T) {
	withServers := defaultConfigForLoad(loaderSections{lspServersExists: true})
	if withServers.LSP.Servers != nil {
		t.Fatal("LSP.Servers should be nil when lsp.servers is explicitly present in YAML")
	}

	withoutServers := defaultConfigForLoad(loaderSections{lspServersExists: false})
	if withoutServers.LSP.Servers == nil {
		t.Fatal("LSP.Servers should keep defaults when lsp.servers is absent in YAML")
	}
}
