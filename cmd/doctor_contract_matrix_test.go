package cmd

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

type doctorProviderContractMatrixEntry struct {
	provider       string
	newCommand     func() *cobra.Command
	requiredFlags  []string
	forbiddenFlags []string
	docsRow        string
}

func TestDoctorProviderContractMatrixCommands(t *testing.T) {
	doctor := newDoctorCommand()
	requireDoctorSubcommands(t, doctor, doctorProviderContractMatrixNames())

	for _, entry := range doctorProviderContractMatrixEntries() {
		t.Run(entry.provider, func(t *testing.T) {
			cmd, _ := newDoctorSubcommandTest(t, entry.newCommand)
			requireDoctorCommandFlags(t, cmd, entry.requiredFlags)
			requireDoctorCommandOmitsFlags(t, cmd, entry.forbiddenFlags)
		})
	}
}

func TestDoctorProviderContractMatrixDocs(t *testing.T) {
	doc := readDoctorCommandsDoc(t)
	for _, entry := range doctorProviderContractMatrixEntries() {
		if !strings.Contains(doc, entry.docsRow) {
			t.Fatalf("docs/commands.md missing doctor provider matrix row for %s:\n%s", entry.provider, entry.docsRow)
		}
	}
}

func doctorProviderContractMatrixEntries() []doctorProviderContractMatrixEntry {
	common := []string{"catalog-model", "json", "print-request", "smoke", "timeout"}
	withCommonModel := func(extra ...string) []string {
		flags := append([]string{"model"}, common...)
		return append(flags, extra...)
	}
	withFlags := func(base []string, extra ...string) []string {
		flags := append([]string{}, base...)
		return append(flags, extra...)
	}
	openAIStyleForbidden := []string{"deployment", "image-smoke", "thinking-smoke", "web-search-smoke", "print-config"}
	openAICompatForbidden := []string{"capabilities", "require-capability", "retention-smoke", "image-smoke", "thinking-smoke", "web-search-smoke", "print-config"}

	return []doctorProviderContractMatrixEntry{
		{
			provider:       "deepseek",
			newCommand:     newDeepSeekDoctorCommand,
			requiredFlags:  withCommonModel("tool-smoke"),
			forbiddenFlags: withFlags(openAICompatForbidden, "deployment"),
			docsRow:        "| `deepseek` | `--model`, `--catalog-model` | `--smoke`, `--tool-smoke` | none | `--print-request` | `DEEPSEEK_API_URL` is an exact Chat Completions endpoint or intentional proxy path | text / tool token usage and cost when returned |",
		},
		{
			provider:       "kimi",
			newCommand:     newKimiDoctorCommand,
			requiredFlags:  withCommonModel("tool-smoke", "image-smoke", "web-search-smoke"),
			forbiddenFlags: []string{"deployment", "capabilities", "require-capability", "retention-smoke", "thinking-smoke", "print-config"},
			docsRow:        "| `kimi` | `--model`, `--catalog-model` | `--smoke`, `--tool-smoke`, `--image-smoke`, `--web-search-smoke` | none | `--print-request` | `KIMI_API_URL` is an exact Chat Completions endpoint or intentional proxy path | token usage plus built-in web search call count / fee observations |",
		},
		{
			provider:       "gemini",
			newCommand:     newGeminiDoctorCommand,
			requiredFlags:  withCommonModel("tool-smoke", "image-smoke", "web-search-smoke"),
			forbiddenFlags: []string{"deployment", "capabilities", "require-capability", "retention-smoke", "thinking-smoke", "print-config"},
			docsRow:        "| `gemini` | `--model`, `--catalog-model` | `--smoke`, `--tool-smoke`, `--image-smoke`, `--web-search-smoke` | none | `--print-request` | text / tool / image use `streamGenerateContent?alt=sse`; native web search uses `generateContent` | SSE / `usageMetadata` usage and cost when returned; web search usage is optional for success |",
		},
		{
			provider:       "claude",
			newCommand:     newClaudeDoctorCommand,
			requiredFlags:  withCommonModel("tool-smoke", "image-smoke", "thinking-smoke", "web-search-smoke"),
			forbiddenFlags: []string{"deployment", "capabilities", "require-capability", "retention-smoke", "print-config"},
			docsRow:        "| `claude` | `--model`, `--catalog-model` | `--smoke`, `--tool-smoke`, `--image-smoke`, `--thinking-smoke`, `--web-search-smoke` | none | `--print-request` | `ANTHROPIC_API_URL` is an exact `/v1/messages` endpoint or intentional proxy path | Messages usage and cost when returned; web search usage is optional for success |",
		},
		{
			provider:       "groq",
			newCommand:     newGroqDoctorCommand,
			requiredFlags:  withCommonModel("tool-smoke"),
			forbiddenFlags: withFlags(openAICompatForbidden, "deployment"),
			docsRow:        "| `groq` | `--model`, `--catalog-model` | `--smoke`, `--tool-smoke` | none | `--print-request` | `GROQ_API_URL` is an exact Chat Completions endpoint or intentional proxy path | text / tool token usage and cost when returned |",
		},
		{
			provider:       "ollama",
			newCommand:     newOllamaDoctorCommand,
			requiredFlags:  withCommonModel("tool-smoke"),
			forbiddenFlags: withFlags(openAICompatForbidden, "deployment"),
			docsRow:        "| `ollama` | `--model`, `--catalog-model` | `--smoke`, `--tool-smoke` | none | `--print-request` | `OLLAMA_BASE_URL` is a base URL; concrete `/api/chat` or `/api/tags` endpoints fail | local zero-cost token usage when returned |",
		},
		{
			provider:       "openrouter",
			newCommand:     newOpenRouterDoctorCommand,
			requiredFlags:  withCommonModel("tool-smoke"),
			forbiddenFlags: withFlags(openAICompatForbidden, "deployment"),
			docsRow:        "| `openrouter` | `--model`, `--catalog-model` | `--smoke`, `--tool-smoke` | none | `--print-request` | `OPENROUTER_API_URL` is Chat Completions / proxy; Anthropic Skin `/v1/messages` is derived | selected route token usage and cost when returned |",
		},
		{
			provider:       "openai",
			newCommand:     newOpenAIDoctorCommand,
			requiredFlags:  withCommonModel("tool-smoke", "retention-smoke", "capabilities", "require-capability"),
			forbiddenFlags: openAIStyleForbidden,
			docsRow:        "| `openai` | `--model`, `--catalog-model` | `--smoke`, `--tool-smoke`, `--retention-smoke` | `--capabilities`, `--require-capability` | `--print-request` | `OPENAI_API_URL` is Chat Completions; `OPENAI_RESPONSES_URL` is Responses | response ID, token usage, cost, and retention chain metadata when returned |",
		},
		{
			provider:       "azure",
			newCommand:     newAzureDoctorCommand,
			requiredFlags:  []string{"deployment", "catalog-model", "json", "print-request", "smoke", "tool-smoke", "retention-smoke", "capabilities", "require-capability", "timeout", "print-config"},
			forbiddenFlags: []string{"model", "image-smoke", "thinking-smoke", "web-search-smoke"},
			docsRow:        "| `azure` | `--deployment`, `--catalog-model` | `--smoke`, `--tool-smoke`, `--retention-smoke` | `--capabilities`, `--require-capability`, `--print-config` | `--print-request` | `AZURE_OPENAI_BASE_URL` is a resource v1 base URL; smoke uses `<normalized_base_url>/responses` | response ID, token usage, cost, and retention chain metadata when returned |",
		},
		{
			provider:       "bedrock",
			newCommand:     newBedrockDoctorCommand,
			requiredFlags:  withCommonModel("tool-smoke", "image-smoke", "thinking-smoke"),
			forbiddenFlags: []string{"deployment", "capabilities", "require-capability", "retention-smoke", "web-search-smoke", "print-config"},
			docsRow:        "| `bedrock` | `--model`, `--catalog-model` | `--smoke`, `--tool-smoke`, `--image-smoke`, `--thinking-smoke` | none | `--print-request` | AWS region / credentials select Bedrock runtime route; request preview is credential-independent | AWS request ID, token usage, and cost when returned; partial usage makes total cost unavailable |",
		},
	}
}

func doctorProviderContractMatrixNames() []string {
	entries := doctorProviderContractMatrixEntries()
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		names = append(names, entry.provider)
	}
	sort.Strings(names)
	return names
}

func requireDoctorSubcommands(t *testing.T, cmd *cobra.Command, want []string) {
	t.Helper()
	got := make([]string, 0, len(cmd.Commands()))
	for _, subcommand := range cmd.Commands() {
		got = append(got, subcommand.Name())
	}
	sort.Strings(got)
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("doctor subcommands = %v, want %v", got, want)
	}
}

func requireDoctorCommandFlags(t *testing.T, cmd *cobra.Command, flags []string) {
	t.Helper()
	for _, flag := range flags {
		if cmd.Flags().Lookup(flag) == nil {
			t.Fatalf("doctor %s missing --%s flag", cmd.Name(), flag)
		}
	}
}

func requireDoctorCommandOmitsFlags(t *testing.T, cmd *cobra.Command, flags []string) {
	t.Helper()
	for _, flag := range flags {
		if cmd.Flags().Lookup(flag) != nil {
			t.Fatalf("doctor %s should not expose --%s flag", cmd.Name(), flag)
		}
	}
}

func readDoctorCommandsDoc(t *testing.T) string {
	t.Helper()
	payload, err := os.ReadFile(filepath.Join("..", "docs", "commands.md"))
	if err != nil {
		t.Fatalf("read docs/commands.md: %v", err)
	}
	return string(payload)
}
