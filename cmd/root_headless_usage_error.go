package cmd

import (
	"strings"

	"github.com/spf13/cobra"
	"github.com/susugadx/xelyon-cli/internal/app"
)

type rootUsageErrorIntent struct {
	writeHeadlessJSON bool
	exitPolicy        app.HeadlessExitPolicy
	input             app.HeadlessInput
}

type rawRootUsageArgs struct {
	provider      string
	promptFile    string
	promptFileSet bool
	imagePath     string
	imageSet      bool
	exitPolicy    app.HeadlessExitPolicy
	exitPolicySet bool
	inputArgs     []string
}

func handleInvalidExitPolicyUsageError(cmd *cobra.Command, args []string, err error) error {
	if shouldWriteHeadlessUsageJSONFromFlags(outputFormat, headless) {
		return writeHeadlessUsageErrorResult(cmd, args, err, app.HeadlessExitPolicyLegacy)
	}
	return err
}

func handleRootFlagParseError(cmd *cobra.Command, err error) error {
	intent := rootFlagParseUsageErrorIntent(cmd)
	if intent.writeHeadlessJSON {
		return writeHeadlessUsageErrorResultWithInput(cmd, intent.input, err, intent.exitPolicy)
	}
	return commandErrorForExitPolicy(err, intent.exitPolicy, 2)
}

func rootFlagParseUsageErrorIntent(cmd *cobra.Command) rootUsageErrorIntent {
	if rootExecutionArgs != nil {
		return rootUsageErrorIntentFromCommandArgs(cmd, rootExecutionArgs)
	}

	policy, err := app.ParseHeadlessExitPolicy(exitCodePolicy)
	if err != nil {
		policy = app.HeadlessExitPolicyLegacy
	}
	if commandIsSubcommand(cmd) {
		return rootUsageErrorIntent{exitPolicy: policy}
	}
	return rootUsageErrorIntent{
		writeHeadlessJSON: shouldWriteHeadlessUsageJSONFromFlags(outputFormat, headless),
		exitPolicy:        policy,
		input:             newHeadlessPreRunInputMetadata(cmd, nil),
	}
}

func rootUsageErrorIntentFromCommandArgs(cmd *cobra.Command, args []string) rootUsageErrorIntent {
	parsed := parseRawRootUsageArgs(rootCommandForUsageIntent(cmd), args)
	policy := app.HeadlessExitPolicyLegacy
	if parsed.exitPolicySet {
		policy = parsed.exitPolicy
	}

	return rootUsageErrorIntent{
		writeHeadlessJSON: !commandIsSubcommand(cmd) && shouldWriteHeadlessUsageJSONFromFlags(outputFormat, headless),
		exitPolicy:        policy,
		input:             newHeadlessPreRunInputMetadataFromRaw(parsed),
	}
}

func shouldWriteHeadlessUsageJSONFromFlags(flagOutputFormat string, flagHeadless bool) bool {
	if flagHeadless {
		return true
	}
	return outputFormatRequestsHeadlessJSON(flagOutputFormat)
}

func outputFormatRequestsHeadlessJSON(flagOutputFormat string) bool {
	resolvedOutputFormat, err := resolveOutputFormat(flagOutputFormat, false)
	return err == nil && resolvedOutputFormat == outputFormatJSON
}

func parseRawRootUsageArgs(root *cobra.Command, args []string) rawRootUsageArgs {
	var parsed rawRootUsageArgs
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			parsed.inputArgs = append(parsed.inputArgs, args[i+1:]...)
			break
		}
		if isRootPositionalArg(arg) {
			parsed.inputArgs = append(parsed.inputArgs, arg)
			continue
		}
		if parseRawShortFlags(root, args, &i, &parsed) {
			continue
		}

		name, value, hasValue := splitRawLongFlag(arg)
		switch name {
		case "provider":
			if !hasValue {
				value, hasValue = rawFlagValueFromNext(args, &i)
			}
			if hasValue {
				parsed.provider = value
			}
		case "prompt-file":
			if !hasValue {
				value, hasValue = rawFlagValueFromNext(args, &i)
			}
			if hasValue {
				parsed.promptFile = value
				parsed.promptFileSet = true
			}
		case "image":
			if !hasValue {
				value, hasValue = rawFlagValueFromNext(args, &i)
			}
			if hasValue {
				parsed.imagePath = value
				parsed.imageSet = true
			}
		case "exit-code-policy":
			if !hasValue {
				value, hasValue = rawFlagValueFromNext(args, &i)
			}
			if hasValue {
				policy, err := app.ParseHeadlessExitPolicy(value)
				if err != nil {
					parsed.exitPolicySet = false
					continue
				}
				parsed.exitPolicy = policy
				parsed.exitPolicySet = true
			}
		default:
			if rawRootFlagConsumesValue(root, name, hasValue) {
				_, _ = rawFlagValueFromNext(args, &i)
			}
		}
	}
	return parsed
}

func newHeadlessPreRunInputMetadataFromRaw(parsed rawRootUsageArgs) app.HeadlessInput {
	var input app.HeadlessInput
	switch {
	case parsed.promptFileSet:
		if parsed.promptFile == "-" {
			input = app.NewHeadlessInput(app.HeadlessInputSourceStdin, "", 0)
		} else {
			input = app.NewHeadlessInput(app.HeadlessInputSourcePromptFile, parsed.promptFile, 0)
		}
	case len(parsed.inputArgs) == 1 && parsed.inputArgs[0] == "-":
		input = app.NewHeadlessInput(app.HeadlessInputSourceStdin, "", 0)
	default:
		query := strings.Join(parsed.inputArgs, " ")
		input = app.NewHeadlessInput(app.HeadlessInputSourceArgs, "", len([]byte(query)))
	}
	if parsed.imageSet {
		return withHeadlessImageInputMetadataForProviderName(input, parsed.imagePath, resolveHeadlessImageMetadataProviderNameForFlag(parsed.provider))
	}
	return input
}

func isRootPositionalArg(arg string) bool {
	return arg == "-" || !strings.HasPrefix(arg, "-")
}

func splitRawLongFlag(arg string) (name string, value string, hasValue bool) {
	if !strings.HasPrefix(arg, "--") {
		return "", "", false
	}
	body := strings.TrimPrefix(arg, "--")
	name, value, hasValue = strings.Cut(body, "=")
	return name, value, hasValue
}

func parseRawShortFlags(root *cobra.Command, args []string, index *int, parsed *rawRootUsageArgs) bool {
	arg := args[*index]
	if !strings.HasPrefix(arg, "-") || strings.HasPrefix(arg, "--") || len(arg) < 2 {
		return false
	}
	body := strings.TrimPrefix(arg, "-")
	for pos := 0; pos < len(body); pos++ {
		name := body[pos : pos+1]
		if !rootShortFlagKnown(root, name) {
			return true
		}
		if !rootShortFlagConsumesValue(root, name) {
			continue
		}

		value, hasValue := rawShortFlagValue(body[pos+1:])
		if !hasValue {
			value, hasValue = rawFlagValueFromNext(args, index)
		}
		applyRawShortFlagValue(parsed, name, value, hasValue)
		return true
	}
	return true
}

func rawShortFlagValue(remainder string) (string, bool) {
	if remainder == "" {
		return "", false
	}
	return strings.TrimPrefix(remainder, "="), true
}

func applyRawShortFlagValue(parsed *rawRootUsageArgs, name string, value string, hasValue bool) {
	if !hasValue {
		return
	}
	switch name {
	case "p":
		parsed.provider = value
	case "i":
		parsed.imagePath = value
		parsed.imageSet = true
	}
}

func rawFlagValueFromNext(args []string, index *int) (string, bool) {
	next := *index + 1
	if next >= len(args) {
		return "", false
	}
	*index = next
	return args[next], true
}

func rawRootFlagConsumesValue(root *cobra.Command, longName string, hasValue bool) bool {
	if hasValue {
		return false
	}
	return rootLongFlagConsumesValue(root, longName)
}

func commandIsSubcommand(cmd *cobra.Command) bool {
	if cmd == nil {
		return false
	}
	return cmd.Root() != cmd
}

func rootCommandForUsageIntent(cmd *cobra.Command) *cobra.Command {
	if cmd == nil {
		return nil
	}
	return cmd.Root()
}

func rootLongFlagConsumesValue(root *cobra.Command, name string) bool {
	if root == nil || name == "" {
		return false
	}
	flag := root.Flags().Lookup(name)
	return flag != nil && flag.NoOptDefVal == ""
}

func rootShortFlagKnown(root *cobra.Command, name string) bool {
	if name == "" {
		return false
	}
	if root == nil {
		return rootMetadataShortFlagKnown(name)
	}
	return root.Flags().ShorthandLookup(name) != nil
}

func rootShortFlagConsumesValue(root *cobra.Command, name string) bool {
	if name == "" {
		return false
	}
	if root == nil {
		return rootMetadataShortFlagConsumesValue(name)
	}
	flag := root.Flags().ShorthandLookup(name)
	return flag != nil && flag.NoOptDefVal == ""
}

func rootMetadataShortFlagKnown(name string) bool {
	switch name {
	case "i", "m", "p", "q", "y":
		return true
	default:
		return false
	}
}

func rootMetadataShortFlagConsumesValue(name string) bool {
	switch name {
	case "i", "m", "p":
		return true
	default:
		return false
	}
}
