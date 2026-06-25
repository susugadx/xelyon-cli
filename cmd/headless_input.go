package cmd

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/susugadx/xelyon-cli/internal/app"
)

const headlessPromptInputMaxBytes = 1 << 20

type headlessPromptInput struct {
	query string
	input app.HeadlessInput
}

func resolveHeadlessPromptInput(cmd *cobra.Command, args []string) (headlessPromptInput, error) {
	if headlessPromptFileFlagChanged(cmd) {
		input := app.NewHeadlessInput(app.HeadlessInputSourcePromptFile, promptFile, 0)
		if len(args) > 0 {
			return headlessPromptInput{input: input}, fmt.Errorf("--prompt-file cannot be used with query arguments")
		}
		if promptFile == "-" {
			return readHeadlessPromptFromStdin(cmd, app.HeadlessInputSourceStdin)
		}
		return readHeadlessPromptFromFile(promptFile)
	}

	if len(args) == 1 && args[0] == "-" {
		return readHeadlessPromptFromStdin(cmd, app.HeadlessInputSourceStdin)
	}

	query := strings.Join(args, " ")
	input := app.NewHeadlessInput(app.HeadlessInputSourceArgs, "", len([]byte(query)))
	if strings.TrimSpace(query) == "" {
		return headlessPromptInput{query: query, input: input}, fmt.Errorf("query argument is required in headless mode")
	}
	return headlessPromptInput{query: query, input: input}, nil
}

func headlessPromptFileFlagChanged(cmd *cobra.Command) bool {
	if cmd == nil {
		return false
	}
	flag := cmd.Flags().Lookup("prompt-file")
	return flag != nil && flag.Changed
}

func readHeadlessPromptFromFile(path string) (headlessPromptInput, error) {
	input := app.NewHeadlessInput(app.HeadlessInputSourcePromptFile, path, 0)
	if strings.TrimSpace(path) == "" {
		return headlessPromptInput{input: input}, fmt.Errorf("--prompt-file requires a path")
	}

	info, err := os.Stat(path)
	if err != nil {
		return headlessPromptInput{input: input}, fmt.Errorf("read prompt file %q: %w", path, err)
	}
	if info.IsDir() {
		return headlessPromptInput{input: input}, fmt.Errorf("prompt file %q is a directory", path)
	}
	if info.Size() > headlessPromptInputMaxBytes {
		return headlessPromptInput{input: input}, fmt.Errorf("prompt file %q exceeds %d bytes", path, headlessPromptInputMaxBytes)
	}

	file, err := os.Open(path)
	if err != nil {
		return headlessPromptInput{input: input}, fmt.Errorf("read prompt file %q: %w", path, err)
	}
	defer file.Close()

	return readHeadlessPrompt(file, app.HeadlessInputSourcePromptFile, path)
}

func readHeadlessPromptFromStdin(cmd *cobra.Command, source app.HeadlessInputSource) (headlessPromptInput, error) {
	var reader io.Reader = os.Stdin
	if cmd != nil {
		reader = cmd.InOrStdin()
	}
	return readHeadlessPrompt(reader, source, "")
}

func readHeadlessPrompt(reader io.Reader, source app.HeadlessInputSource, path string) (headlessPromptInput, error) {
	input := app.NewHeadlessInput(source, path, 0)
	data, err := io.ReadAll(io.LimitReader(reader, headlessPromptInputMaxBytes+1))
	if err != nil {
		return headlessPromptInput{input: input}, fmt.Errorf("read prompt input: %w", err)
	}
	input = app.NewHeadlessInput(source, path, len(data))
	if len(data) > headlessPromptInputMaxBytes {
		return headlessPromptInput{input: input}, fmt.Errorf("prompt input exceeds %d bytes", headlessPromptInputMaxBytes)
	}

	query := string(data)
	if strings.TrimSpace(query) == "" {
		return headlessPromptInput{query: query, input: input}, fmt.Errorf("prompt input is empty")
	}
	return headlessPromptInput{query: query, input: input}, nil
}
