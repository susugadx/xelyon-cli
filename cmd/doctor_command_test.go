package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestRootCommand_AzureDoctorHelpShowsDoctorFlags(t *testing.T) {
	out := newRootCommandExecutionTest(t)
	rootCmd.SetArgs([]string{"doctor", "azure", "--help"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("root Execute() error = %v\noutput:\n%s", err, out.String())
	}
	for _, want := range []string{
		"--deployment",
		"--print-config",
		"--retention-smoke",
		"--capabilities",
		"--require-capability",
		"--print-request",
		"Diagnose Azure OpenAI configuration",
		"AZURE_OPENAI_BASE_URL is a resource v1 base URL",
	} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("output = %q, want Azure doctor help substring %q", out.String(), want)
		}
	}
	if strings.Contains(out.String(), "XELYON CLI is an AI coding agent") {
		t.Fatalf("output = %q, should not show root long help", out.String())
	}
}

func TestRootCommand_AzureDoctorFailureDoesNotPrintRootUsage(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	_ = os.Unsetenv("AZURE_OPENAI_BASE_URL")
	_ = os.Unsetenv("AZURE_OPENAI_API_KEY")
	_ = os.Unsetenv("AZURE_OPENAI_AUTH_TOKEN")
	_ = os.Unsetenv("AZURE_OPENAI_AUTH_TOKEN_COMMAND")

	out := newRootCommandExecutionTest(t)
	rootCmd.SetArgs([]string{"doctor", "azure"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatalf("root Execute() error = nil, want diagnostics failure\noutput:\n%s", out.String())
	}
	if !strings.Contains(out.String(), "Azure OpenAI doctor") {
		t.Fatalf("output = %q, want doctor report", out.String())
	}
	if strings.Contains(out.String(), "Usage:\n  xelyon [query]") {
		t.Fatalf("output = %q, should not append root usage", out.String())
	}
}

func TestMCPDoctorCommandFlags(t *testing.T) {
	cmd, _ := newDoctorSubcommandTest(t, newMCPDoctorCommand)
	for _, name := range []string{"connect", "server", "tools", "json"} {
		if cmd.Flags().Lookup(name) == nil {
			t.Fatalf("doctor mcp missing --%s flag", name)
		}
	}
}

func TestNewDoctorCommandIncludesMCP(t *testing.T) {
	doctor := newDoctorCommand()
	if findSubcommand(doctor, "mcp") == nil {
		t.Fatal("doctor command missing mcp subcommand")
	}
}

func TestMCPDoctorDocs(t *testing.T) {
	commandsDoc := readRepoText(t, filepath.Join("docs", "commands.md"))
	for _, want := range []string{
		"### `xelyon doctor mcp`",
		"xelyon doctor mcp --connect --tools",
		"`doctor mcp` は env value と raw args を出力しません",
	} {
		if !strings.Contains(commandsDoc, want) {
			t.Fatalf("docs/commands.md missing %q", want)
		}
	}

	mcpDoc := readRepoText(t, filepath.Join("docs", "mcp.md"))
	for _, want := range []string{
		"xelyon doctor mcp",
		"MCP server process も起動しない",
		"tools/call は実行しない",
	} {
		if !strings.Contains(mcpDoc, want) {
			t.Fatalf("docs/mcp.md missing %q", want)
		}
	}
}

func findSubcommand(cmd *cobra.Command, name string) *cobra.Command {
	for _, child := range cmd.Commands() {
		if child.Name() == name {
			return child
		}
	}
	return nil
}
