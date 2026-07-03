package cmd

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/susugadx/xelyon-cli/internal/providerdiag"
)

type doctorProviderContractMatrixEntry struct {
	provider              string
	newCommand            func() *cobra.Command
	requiredFlags         []string
	forbiddenFlags        []string
	docsRow               string
	capabilityContractRow string
}

func TestDoctorProviderContractMatrixCommands(t *testing.T) {
	doctor := newDoctorCommand()
	requireDoctorSubcommands(t, doctor, doctorSubcommandNames())

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

func TestDoctorProviderCapabilityContractDocs(t *testing.T) {
	doc := readDoctorContractDoc(t)
	requireContainsAll(t, "doctor capability vocabulary", doc, providerdiag.SupportedRequiredCapabilities())
	for _, entry := range doctorProviderContractMatrixEntries() {
		if !strings.Contains(doc, entry.capabilityContractRow) {
			t.Fatalf("docs/design/doctor-contract-v1.md missing capability matrix row for %s:\n%s", entry.provider, entry.capabilityContractRow)
		}
	}
}

func TestDoctorRequiredCapabilityFlagHelpUsesSharedVocabulary(t *testing.T) {
	want := providerdiag.SupportedRequiredCapabilitiesText()
	for _, entry := range doctorProviderContractMatrixEntries() {
		t.Run(entry.provider, func(t *testing.T) {
			cmd, _ := newDoctorSubcommandTest(t, entry.newCommand)
			flag := cmd.Flags().Lookup("require-capability")
			if flag == nil {
				t.Fatalf("doctor %s missing --require-capability flag", entry.provider)
			}
			if !strings.Contains(flag.Usage, want) {
				t.Fatalf("doctor %s --require-capability usage = %q, want shared vocabulary %q", entry.provider, flag.Usage, want)
			}
		})
	}
}

func TestDoctorSmokeMatrixMakeTargetContract(t *testing.T) {
	makefile := readRepoText(t, "Makefile")
	requireContainsAll(t, "doctor smoke matrix Makefile target", makefile, []string{
		"DOCTOR_SMOKE_PROVIDERS ?=",
		"doctor-smoke-matrix:",
		"Set DOCTOR_SMOKE_PROVIDERS=",
		"openai-doctor-smoke",
		"deepseek-doctor-smoke",
		"gemini-doctor-smoke",
		"claude-doctor-smoke",
		"groq-doctor-smoke",
		"openrouter-doctor-smoke",
		"kimi-doctor-smoke",
		"ollama-doctor-smoke",
		"bedrock-doctor-smoke",
		"azure-doctor-smoke",
		"matrix_status=0",
		"if ! $(MAKE) --no-print-directory \"$$target\"",
		"matrix_status=1",
		"exit $$matrix_status",
	})

	doc := readDoctorCommandsDoc(t)
	requireContainsAll(t, "doctor smoke matrix docs", doc, []string{
		`DOCTOR_SMOKE_PROVIDERS="openai groq kimi" make doctor-smoke-matrix`,
		"`DOCTOR_SMOKE_PROVIDERS` が空のときは実 API / ローカル runtime を呼ばず終了します",
	})
}

func TestDoctorSmokeFailureVocabularyContractDocs(t *testing.T) {
	doc := readDoctorContractDoc(t)
	requireContainsAll(t, "doctor smoke failure vocabulary", doc, []string{
		"### Shared Smoke Failure Vocabulary",
		"`auth`",
		"`quota`",
		"`model_unavailable`",
		"`endpoint_mismatch`",
		"`feature_unsupported`",
		"`empty_response`",
		"`generic`",
		"`internal/providerdiag` | Provider-neutral doctor policy DTOs, required capability evaluation, and smoke failure vocabulary",
	})
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
	openAICompatForbidden := []string{"retention-smoke", "image-smoke", "thinking-smoke", "web-search-smoke", "print-config"}

	return []doctorProviderContractMatrixEntry{
		{
			provider:              "deepseek",
			newCommand:            newDeepSeekDoctorCommand,
			requiredFlags:         withCommonModel("tool-smoke", "capabilities", "require-capability"),
			forbiddenFlags:        withFlags(openAICompatForbidden, "deployment"),
			docsRow:               "| `deepseek` | `--model`, `--catalog-model` | `--smoke`, `--tool-smoke` | `--capabilities`, `--require-capability` | `--print-request` | `DEEPSEEK_API_URL` is an exact Chat Completions endpoint or intentional proxy path | text / tool token usage and cost when returned |",
			capabilityContractRow: "| DeepSeek | no | no | yes | config | no | no | model+config | no | no | no | no |",
		},
		{
			provider:              "kimi",
			newCommand:            newKimiDoctorCommand,
			requiredFlags:         withCommonModel("tool-smoke", "image-smoke", "web-search-smoke", "capabilities", "require-capability"),
			forbiddenFlags:        []string{"deployment", "retention-smoke", "thinking-smoke", "print-config"},
			docsRow:               "| `kimi` | `--model`, `--catalog-model` | `--smoke`, `--tool-smoke`, `--image-smoke`, `--web-search-smoke` | `--capabilities`, `--require-capability` | `--print-request` | `KIMI_API_URL` is an exact Chat Completions endpoint or intentional proxy path | token usage plus built-in web search call count / fee observations |",
			capabilityContractRow: "| Kimi | no | no | yes | config | yes | yes | model+config | no | no | no | no |",
		},
		{
			provider:              "gemini",
			newCommand:            newGeminiDoctorCommand,
			requiredFlags:         withCommonModel("tool-smoke", "image-smoke", "web-search-smoke", "capabilities", "require-capability"),
			forbiddenFlags:        []string{"deployment", "retention-smoke", "thinking-smoke", "print-config"},
			docsRow:               "| `gemini` | `--model`, `--catalog-model` | `--smoke`, `--tool-smoke`, `--image-smoke`, `--web-search-smoke` | `--capabilities`, `--require-capability` | `--print-request` | text / tool / image use `streamGenerateContent?alt=sse`; native web search uses `generateContent` | SSE / `usageMetadata` usage and cost when returned; web search usage is optional for success |",
			capabilityContractRow: "| Gemini | no | no | no | config | model | model | model+config | no | no | no | no |",
		},
		{
			provider:              "claude",
			newCommand:            newClaudeDoctorCommand,
			requiredFlags:         withCommonModel("tool-smoke", "image-smoke", "thinking-smoke", "web-search-smoke", "capabilities", "require-capability"),
			forbiddenFlags:        []string{"deployment", "retention-smoke", "print-config"},
			docsRow:               "| `claude` | `--model`, `--catalog-model` | `--smoke`, `--tool-smoke`, `--image-smoke`, `--thinking-smoke`, `--web-search-smoke` | `--capabilities`, `--require-capability` | `--print-request` | `ANTHROPIC_API_URL` is an exact `/v1/messages` endpoint or intentional proxy path | Messages usage and cost when returned; web search usage is optional for success |",
			capabilityContractRow: "| Claude / Anthropic | no | no | no | config | model | model | config | no | no | no | no |",
		},
		{
			provider:              "groq",
			newCommand:            newGroqDoctorCommand,
			requiredFlags:         withCommonModel("tool-smoke", "capabilities", "require-capability"),
			forbiddenFlags:        withFlags(openAICompatForbidden, "deployment"),
			docsRow:               "| `groq` | `--model`, `--catalog-model` | `--smoke`, `--tool-smoke` | `--capabilities`, `--require-capability` | `--print-request` | `GROQ_API_URL` is an exact Chat Completions endpoint or intentional proxy path | text / tool token usage and cost when returned |",
			capabilityContractRow: "| Groq | no | no | yes | config | no | no | no | no | no | no | no |",
		},
		{
			provider:              "ollama",
			newCommand:            newOllamaDoctorCommand,
			requiredFlags:         withCommonModel("tool-smoke", "capabilities", "require-capability"),
			forbiddenFlags:        withFlags(openAICompatForbidden, "deployment"),
			docsRow:               "| `ollama` | `--model`, `--catalog-model` | `--smoke`, `--tool-smoke` | `--capabilities`, `--require-capability` | `--print-request` | `OLLAMA_BASE_URL` is a base URL; concrete `/api/chat` or `/api/tags` endpoints fail | local zero-cost token usage when returned |",
			capabilityContractRow: "| Ollama | no | no | no | config | no | no | no | no | no | no | discovery |",
		},
		{
			provider:              "openrouter",
			newCommand:            newOpenRouterDoctorCommand,
			requiredFlags:         withCommonModel("tool-smoke", "capabilities", "require-capability"),
			forbiddenFlags:        withFlags(openAICompatForbidden, "deployment"),
			docsRow:               "| `openrouter` | `--model`, `--catalog-model` | `--smoke`, `--tool-smoke` | `--capabilities`, `--require-capability` | `--print-request` | `OPENROUTER_API_URL` is Chat Completions / proxy; Anthropic Skin `/v1/messages` is derived | selected route token usage and cost when returned |",
			capabilityContractRow: "| OpenRouter | no | no | route | config | model | no | no | no | no | no | no |",
		},
		{
			provider:              "openai",
			newCommand:            newOpenAIDoctorCommand,
			requiredFlags:         withCommonModel("tool-smoke", "retention-smoke", "capabilities", "require-capability"),
			forbiddenFlags:        openAIStyleForbidden,
			docsRow:               "| `openai` | `--model`, `--catalog-model` | `--smoke`, `--tool-smoke`, `--retention-smoke` | `--capabilities`, `--require-capability` | `--print-request` | `OPENAI_API_URL` is Chat Completions; `OPENAI_RESPONSES_URL` is Responses | response ID, token usage, cost, and retention chain metadata when returned |",
			capabilityContractRow: "| OpenAI | route | route+catalog | route | config | model | model+route | route+config | route+config | route+config | route+config | no |",
		},
		{
			provider:              "openai-subscription",
			newCommand:            newOpenAISubscriptionDoctorCommand,
			requiredFlags:         withCommonModel("tool-smoke", "retention-smoke", "cache-smoke", "compact-smoke", "thinking-smoke", "web-search-smoke", "capabilities", "require-capability"),
			forbiddenFlags:        []string{"deployment", "image-smoke", "print-config"},
			docsRow:               "| `openai-subscription` | `--model`, `--catalog-model` | `--smoke`, `--tool-smoke`, `--retention-smoke`, `--cache-smoke`, `--compact-smoke`, `--thinking-smoke`, `--web-search-smoke` | `--capabilities`, `--require-capability` | `--print-request` | ChatGPT/Codex OAuth subscription endpoint; full-payload Responses-shaped runtime plus dedicated native web_search payload | streaming usage when returned; web search call count when observed; cost is N/A (ChatGPT subscription) |",
			capabilityContractRow: "| OpenAI Subscription | yes | yes | no | yes | no | yes | config | no | no | no | no |",
		},
		{
			provider:              "azure",
			newCommand:            newAzureDoctorCommand,
			requiredFlags:         []string{"deployment", "catalog-model", "json", "print-request", "smoke", "tool-smoke", "retention-smoke", "capabilities", "require-capability", "timeout", "print-config"},
			forbiddenFlags:        []string{"model", "image-smoke", "thinking-smoke", "web-search-smoke"},
			docsRow:               "| `azure` | `--deployment`, `--catalog-model` | `--smoke`, `--tool-smoke`, `--retention-smoke` | `--capabilities`, `--require-capability`, `--print-config` | `--print-request` | `AZURE_OPENAI_BASE_URL` is a resource v1 base URL; smoke uses `<normalized_base_url>/responses` | response ID, token usage, cost, and retention chain metadata when returned |",
			capabilityContractRow: "| Azure OpenAI | route | route+catalog | no | config | model | no | route+config | route+config | route+config | route+config | no |",
		},
		{
			provider:              "bedrock",
			newCommand:            newBedrockDoctorCommand,
			requiredFlags:         withCommonModel("tool-smoke", "image-smoke", "thinking-smoke", "capabilities", "require-capability"),
			forbiddenFlags:        []string{"deployment", "retention-smoke", "web-search-smoke", "print-config"},
			docsRow:               "| `bedrock` | `--model`, `--catalog-model` | `--smoke`, `--tool-smoke`, `--image-smoke`, `--thinking-smoke` | `--capabilities`, `--require-capability` | `--print-request` | AWS region / credentials select Bedrock runtime route; request preview is credential-independent | AWS request ID, token usage, and cost when returned; partial usage makes total cost unavailable |",
			capabilityContractRow: "| Bedrock | no | no | no | route+config | route | no | route+config | no | no | no | no |",
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

func doctorSubcommandNames() []string {
	names := append(doctorProviderContractMatrixNames(), "mcp")
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
	return readRepoText(t, filepath.Join("docs", "commands.md"))
}

func readDoctorContractDoc(t *testing.T) string {
	t.Helper()
	return readRepoText(t, filepath.Join("docs", "design", "doctor-contract-v1.md"))
}

func readRepoText(t *testing.T, path string) string {
	t.Helper()
	payload, err := os.ReadFile(filepath.Join("..", path))
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(payload)
}
