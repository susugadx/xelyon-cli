package clidoctor

import (
	"bytes"
	"context"
	"fmt"
	"strconv"
	"testing"
	"time"
)

var (
	doctorDeploymentFlag                       string
	doctorCatalogModelFlag                     string
	doctorBedrockModelFlag                     string
	doctorClaudeModelFlag                      string
	doctorDeepSeekModelFlag                    string
	doctorGeminiModelFlag                      string
	doctorGroqModelFlag                        string
	doctorKimiModelFlag                        string
	doctorOllamaModelFlag                      string
	doctorOpenAIModelFlag                      string
	doctorOpenAISubscriptionModelFlag          string
	doctorOpenRouterModelFlag                  string
	doctorSmokeFlag                            bool
	doctorToolSmokeFlag                        bool
	doctorCapabilitiesFlag                     bool
	doctorRequiredCapabilityFlags              []string
	doctorAzureRetentionSmokeFlag              bool
	doctorOpenAIRetentionSmokeFlag             bool
	doctorOpenAISubscriptionRetentionSmokeFlag bool
	doctorOpenAISubscriptionCacheSmokeFlag     bool
	doctorOpenAISubscriptionCompactSmokeFlag   bool
	doctorOpenAISubscriptionThinkingSmokeFlag  bool
	doctorOpenAISubscriptionWebSearchSmokeFlag bool
	doctorBedrockImageSmokeFlag                bool
	doctorBedrockThinkingSmokeFlag             bool
	doctorClaudeImageSmokeFlag                 bool
	doctorClaudeThinkingSmokeFlag              bool
	doctorClaudeWebSearchSmokeFlag             bool
	doctorGeminiImageSmokeFlag                 bool
	doctorGeminiWebSearchSmokeFlag             bool
	doctorKimiImageSmokeFlag                   bool
	doctorKimiWebSearchSmokeFlag               bool
	doctorMCPConnectFlag                       bool
	doctorMCPServerFlag                        string
	doctorMCPToolsFlag                         bool
	doctorTimeoutFlag                          = DefaultTimeout
	doctorJSONFlag                             bool
	doctorPrintConfigFlag                      bool
	doctorPrintRequestFlag                     bool
)

type doctorTestCommand struct {
	out          *bytes.Buffer
	flags        doctorTestFlagSet
	SilenceUsage bool
}

type doctorTestFlagSet struct {
	cmd     *doctorTestCommand
	changed map[string]bool
}

func newDoctorSubcommandTest(t *testing.T, newCommand func() *doctorTestCommand) (*doctorTestCommand, *bytes.Buffer) {
	t.Helper()
	resetDoctorTestFlags()
	t.Cleanup(resetDoctorTestFlags)

	var out bytes.Buffer
	cmd := newCommand()
	cmd.out = &out
	cmd.flags.cmd = cmd
	if cmd.flags.changed == nil {
		cmd.flags.changed = make(map[string]bool)
	}
	return cmd, &out
}

func newDoctorTestCommand() *doctorTestCommand {
	cmd := &doctorTestCommand{}
	cmd.flags.cmd = cmd
	cmd.flags.changed = make(map[string]bool)
	return cmd
}

func newAzureDoctorCommand() *doctorTestCommand              { return newDoctorTestCommand() }
func newBedrockDoctorCommand() *doctorTestCommand            { return newDoctorTestCommand() }
func newClaudeDoctorCommand() *doctorTestCommand             { return newDoctorTestCommand() }
func newDeepSeekDoctorCommand() *doctorTestCommand           { return newDoctorTestCommand() }
func newGeminiDoctorCommand() *doctorTestCommand             { return newDoctorTestCommand() }
func newGroqDoctorCommand() *doctorTestCommand               { return newDoctorTestCommand() }
func newKimiDoctorCommand() *doctorTestCommand               { return newDoctorTestCommand() }
func newMCPDoctorCommand() *doctorTestCommand                { return newDoctorTestCommand() }
func newOllamaDoctorCommand() *doctorTestCommand             { return newDoctorTestCommand() }
func newOpenAIDoctorCommand() *doctorTestCommand             { return newDoctorTestCommand() }
func newOpenAISubscriptionDoctorCommand() *doctorTestCommand { return newDoctorTestCommand() }
func newOpenRouterDoctorCommand() *doctorTestCommand         { return newDoctorTestCommand() }

func (cmd *doctorTestCommand) Flags() *doctorTestFlagSet {
	return &cmd.flags
}

func (flags *doctorTestFlagSet) Set(name, value string) error {
	if flags.changed == nil {
		flags.changed = make(map[string]bool)
	}
	flags.changed[name] = true
	switch name {
	case "deployment":
		doctorDeploymentFlag = value
	case "catalog-model":
		doctorCatalogModelFlag = value
	case "model":
		setDoctorTestModelFlag(value)
	case "smoke":
		return setDoctorTestBool(value, &doctorSmokeFlag)
	case "tool-smoke":
		return setDoctorTestBool(value, &doctorToolSmokeFlag)
	case "capabilities":
		return setDoctorTestBool(value, &doctorCapabilitiesFlag)
	case "retention-smoke":
		doctorAzureRetentionSmokeFlag = true
		doctorOpenAIRetentionSmokeFlag = true
		doctorOpenAISubscriptionRetentionSmokeFlag = true
		return setDoctorTestBool(value, &doctorAzureRetentionSmokeFlag)
	case "cache-smoke":
		return setDoctorTestBool(value, &doctorOpenAISubscriptionCacheSmokeFlag)
	case "compact-smoke":
		return setDoctorTestBool(value, &doctorOpenAISubscriptionCompactSmokeFlag)
	case "thinking-smoke":
		doctorBedrockThinkingSmokeFlag = true
		doctorClaudeThinkingSmokeFlag = true
		doctorOpenAISubscriptionThinkingSmokeFlag = true
		return setDoctorTestBool(value, &doctorBedrockThinkingSmokeFlag)
	case "image-smoke":
		doctorBedrockImageSmokeFlag = true
		doctorClaudeImageSmokeFlag = true
		doctorGeminiImageSmokeFlag = true
		doctorKimiImageSmokeFlag = true
		return setDoctorTestBool(value, &doctorBedrockImageSmokeFlag)
	case "web-search-smoke":
		doctorClaudeWebSearchSmokeFlag = true
		doctorGeminiWebSearchSmokeFlag = true
		doctorKimiWebSearchSmokeFlag = true
		doctorOpenAISubscriptionWebSearchSmokeFlag = true
		return setDoctorTestBool(value, &doctorClaudeWebSearchSmokeFlag)
	case "connect":
		return setDoctorTestBool(value, &doctorMCPConnectFlag)
	case "server":
		doctorMCPServerFlag = value
	case "tools":
		return setDoctorTestBool(value, &doctorMCPToolsFlag)
	case "json":
		return setDoctorTestBool(value, &doctorJSONFlag)
	case "print-config":
		return setDoctorTestBool(value, &doctorPrintConfigFlag)
	case "print-request":
		return setDoctorTestBool(value, &doctorPrintRequestFlag)
	case "timeout":
		timeout, err := time.ParseDuration(value)
		if err != nil {
			return err
		}
		doctorTimeoutFlag = timeout
	case "require-capability":
		doctorRequiredCapabilityFlags = append(doctorRequiredCapabilityFlags, value)
	default:
		return fmt.Errorf("unknown doctor test flag %q", name)
	}
	return nil
}

func (flags *doctorTestFlagSet) Changed(name string) bool {
	return flags.changed[name]
}

func setDoctorTestBool(value string, target *bool) error {
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return err
	}
	*target = parsed
	return nil
}

func setDoctorTestModelFlag(value string) {
	doctorBedrockModelFlag = value
	doctorClaudeModelFlag = value
	doctorDeepSeekModelFlag = value
	doctorGeminiModelFlag = value
	doctorGroqModelFlag = value
	doctorKimiModelFlag = value
	doctorOllamaModelFlag = value
	doctorOpenAIModelFlag = value
	doctorOpenAISubscriptionModelFlag = value
	doctorOpenRouterModelFlag = value
}

func resetDoctorTestFlags() {
	doctorDeploymentFlag = ""
	doctorCatalogModelFlag = ""
	doctorBedrockModelFlag = ""
	doctorClaudeModelFlag = ""
	doctorDeepSeekModelFlag = ""
	doctorGeminiModelFlag = ""
	doctorGroqModelFlag = ""
	doctorKimiModelFlag = ""
	doctorOllamaModelFlag = ""
	doctorOpenAIModelFlag = ""
	doctorOpenAISubscriptionModelFlag = ""
	doctorOpenRouterModelFlag = ""
	doctorSmokeFlag = false
	doctorToolSmokeFlag = false
	doctorCapabilitiesFlag = false
	doctorRequiredCapabilityFlags = nil
	doctorAzureRetentionSmokeFlag = false
	doctorOpenAIRetentionSmokeFlag = false
	doctorOpenAISubscriptionRetentionSmokeFlag = false
	doctorOpenAISubscriptionCacheSmokeFlag = false
	doctorOpenAISubscriptionCompactSmokeFlag = false
	doctorOpenAISubscriptionThinkingSmokeFlag = false
	doctorOpenAISubscriptionWebSearchSmokeFlag = false
	doctorBedrockImageSmokeFlag = false
	doctorBedrockThinkingSmokeFlag = false
	doctorClaudeImageSmokeFlag = false
	doctorClaudeThinkingSmokeFlag = false
	doctorClaudeWebSearchSmokeFlag = false
	doctorGeminiImageSmokeFlag = false
	doctorGeminiWebSearchSmokeFlag = false
	doctorKimiImageSmokeFlag = false
	doctorKimiWebSearchSmokeFlag = false
	doctorMCPConnectFlag = false
	doctorMCPServerFlag = ""
	doctorMCPToolsFlag = false
	doctorTimeoutFlag = DefaultTimeout
	doctorJSONFlag = false
	doctorPrintConfigFlag = false
	doctorPrintRequestFlag = false
}

func setGeminiDoctorCommandTestEnv(t *testing.T, apiKey string) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("GEMINI_API_KEY", apiKey)
	t.Setenv("GEMINI_API_URL", "")
	t.Setenv("GEMINI_CONTEXT_CACHING", "")
	t.Setenv("GEMINI_FC_MODE", "")
	t.Setenv("XELYON_MODEL", "")
}

func setKimiDoctorCommandTestEnv(t *testing.T, apiKey string) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("MOONSHOT_API_KEY", apiKey)
	t.Setenv("KIMI_API_URL", "")
	t.Setenv("KIMI_FUNCTION_CALLING", "")
	t.Setenv("XELYON_MODEL", "")
}

func doctorTestCommonOptions() CommonOptions {
	return CommonOptions{
		CatalogModel:         doctorCatalogModelFlag,
		Smoke:                doctorSmokeFlag,
		ToolSmoke:            doctorToolSmokeFlag,
		Capabilities:         doctorCapabilitiesFlag,
		RequiredCapabilities: doctorRequiredCapabilityFlags,
		Timeout:              doctorTimeoutFlag,
		JSON:                 doctorJSONFlag,
		PrintRequest:         doctorPrintRequestFlag,
	}
}

func applyDoctorTestResult(cmd *doctorTestCommand, silenceUsage bool, err error) error {
	if silenceUsage {
		cmd.SilenceUsage = true
	}
	return err
}

func runAzureDoctorInvocation(cmd *doctorTestCommand, args []string) error {
	silenceUsage, err := RunAzureDoctor(context.Background(), cmd.out, AzureOptions{
		CommonOptions:  doctorTestCommonOptions(),
		Deployment:     doctorDeploymentFlag,
		RetentionSmoke: doctorAzureRetentionSmokeFlag,
		PrintConfig:    doctorPrintConfigFlag,
	})
	return applyDoctorTestResult(cmd, silenceUsage, err)
}

func runBedrockDoctorInvocation(cmd *doctorTestCommand, args []string) error {
	silenceUsage, err := RunBedrockDoctor(context.Background(), cmd.out, BedrockOptions{
		CommonOptions: doctorTestCommonOptions(),
		Model:         doctorBedrockModelFlag,
		ImageSmoke:    doctorBedrockImageSmokeFlag,
		ThinkingSmoke: doctorBedrockThinkingSmokeFlag,
	})
	return applyDoctorTestResult(cmd, silenceUsage, err)
}

func runClaudeDoctorInvocation(cmd *doctorTestCommand, args []string) error {
	silenceUsage, err := RunClaudeDoctor(context.Background(), cmd.out, ClaudeOptions{
		CommonOptions:  doctorTestCommonOptions(),
		Model:          doctorClaudeModelFlag,
		ImageSmoke:     doctorClaudeImageSmokeFlag,
		ThinkingSmoke:  doctorClaudeThinkingSmokeFlag,
		WebSearchSmoke: doctorClaudeWebSearchSmokeFlag,
	})
	return applyDoctorTestResult(cmd, silenceUsage, err)
}

func runDeepSeekDoctorInvocation(cmd *doctorTestCommand, args []string) error {
	silenceUsage, err := RunDeepSeekDoctor(context.Background(), cmd.out, DeepSeekOptions{
		CommonOptions: doctorTestCommonOptions(),
		Model:         doctorDeepSeekModelFlag,
	})
	return applyDoctorTestResult(cmd, silenceUsage, err)
}

func runGeminiDoctorInvocation(cmd *doctorTestCommand, args []string) error {
	silenceUsage, err := RunGeminiDoctor(context.Background(), cmd.out, GeminiOptions{
		CommonOptions:  doctorTestCommonOptions(),
		Model:          doctorGeminiModelFlag,
		ImageSmoke:     doctorGeminiImageSmokeFlag,
		WebSearchSmoke: doctorGeminiWebSearchSmokeFlag,
	})
	return applyDoctorTestResult(cmd, silenceUsage, err)
}

func runGroqDoctorInvocation(cmd *doctorTestCommand, args []string) error {
	silenceUsage, err := RunGroqDoctor(context.Background(), cmd.out, GroqOptions{
		CommonOptions: doctorTestCommonOptions(),
		Model:         doctorGroqModelFlag,
	})
	return applyDoctorTestResult(cmd, silenceUsage, err)
}

func runKimiDoctorInvocation(cmd *doctorTestCommand, args []string) error {
	silenceUsage, err := RunKimiDoctor(context.Background(), cmd.out, KimiOptions{
		CommonOptions:  doctorTestCommonOptions(),
		Model:          doctorKimiModelFlag,
		ModelChanged:   cmd.Flags().Changed("model"),
		ImageSmoke:     doctorKimiImageSmokeFlag,
		WebSearchSmoke: doctorKimiWebSearchSmokeFlag,
	})
	return applyDoctorTestResult(cmd, silenceUsage, err)
}

func runMCPDoctorInvocation(cmd *doctorTestCommand, args []string) error {
	silenceUsage, err := RunMCPDoctor(context.Background(), cmd.out, MCPOptions{
		JSON:         doctorJSONFlag,
		Connect:      doctorMCPConnectFlag,
		Server:       doctorMCPServerFlag,
		IncludeTools: doctorMCPToolsFlag,
	})
	return applyDoctorTestResult(cmd, silenceUsage, err)
}

func runOllamaDoctorInvocation(cmd *doctorTestCommand, args []string) error {
	silenceUsage, err := RunOllamaDoctor(context.Background(), cmd.out, OllamaOptions{
		CommonOptions: doctorTestCommonOptions(),
		Model:         doctorOllamaModelFlag,
	})
	return applyDoctorTestResult(cmd, silenceUsage, err)
}

func runOpenAIDoctorInvocation(cmd *doctorTestCommand, args []string) error {
	silenceUsage, err := RunOpenAIDoctor(context.Background(), cmd.out, OpenAIOptions{
		CommonOptions:  doctorTestCommonOptions(),
		Model:          doctorOpenAIModelFlag,
		RetentionSmoke: doctorOpenAIRetentionSmokeFlag,
	})
	return applyDoctorTestResult(cmd, silenceUsage, err)
}

func runOpenAISubscriptionDoctorInvocation(cmd *doctorTestCommand, args []string) error {
	silenceUsage, err := RunOpenAISubscriptionDoctor(context.Background(), cmd.out, OpenAISubscriptionOptions{
		CommonOptions:  doctorTestCommonOptions(),
		Model:          doctorOpenAISubscriptionModelFlag,
		RetentionSmoke: doctorOpenAISubscriptionRetentionSmokeFlag,
		CacheSmoke:     doctorOpenAISubscriptionCacheSmokeFlag,
		CompactSmoke:   doctorOpenAISubscriptionCompactSmokeFlag,
		ThinkingSmoke:  doctorOpenAISubscriptionThinkingSmokeFlag,
		WebSearchSmoke: doctorOpenAISubscriptionWebSearchSmokeFlag,
		SmokeOutput:    cmd.out,
	})
	return applyDoctorTestResult(cmd, silenceUsage, err)
}

func runOpenRouterDoctorInvocation(cmd *doctorTestCommand, args []string) error {
	silenceUsage, err := RunOpenRouterDoctor(context.Background(), cmd.out, OpenRouterOptions{
		CommonOptions: doctorTestCommonOptions(),
		Model:         doctorOpenRouterModelFlag,
	})
	return applyDoctorTestResult(cmd, silenceUsage, err)
}
