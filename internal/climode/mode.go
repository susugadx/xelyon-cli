package climode

import (
	"fmt"
	"strings"
)

const (
	// OutputFormatText は通常の人間向け text output を表す。
	OutputFormatText = "text"
	// OutputFormatJSON は headless / machine-readable JSON output を表す。
	OutputFormatJSON = "json"
)

// Mode は root command が起動する実行モードを表す。
type Mode string

const (
	// ModeHeadless は JSON headless 実行を表す。
	ModeHeadless Mode = "headless"
	// ModeOnce は text one-shot 実行を表す。
	ModeOnce Mode = "once"
	// ModeInteractive は通常の interactive TUI 実行を表す。
	ModeInteractive Mode = "interactive"
	// ModeInteractiveImage は画像付き interactive TUI 実行を表す。
	ModeInteractiveImage Mode = "interactive_image"
	// ModeOnceImage は画像付き one-shot 実行を表す。
	ModeOnceImage Mode = "once_image"
	// ModeResume は保存済み session resume 実行を表す。
	ModeResume Mode = "resume"
)

// Request は CLI flags / args から実行モードを決めるための入力である。
type Request struct {
	OutputFormat string
	HasQuery     bool
	HasImage     bool
	Once         bool
	Interactive  bool
	Resume       bool
	Quiet        bool
}

func (r Request) implicitTextOnce() bool {
	return r.OutputFormat == OutputFormatText && r.HasQuery && !r.HasImage && !r.Interactive && !r.Resume
}

func (r Request) implicitImageOnce() bool {
	return r.OutputFormat == OutputFormatText && r.HasImage && !r.Interactive && !r.Resume
}

func (r Request) explicitImageOnce() bool {
	return r.HasImage && r.Once
}

func (r Request) effectiveOnce() bool {
	return r.Once || r.implicitTextOnce() || r.implicitImageOnce()
}

func (r Request) validate() error {
	if r.Interactive && r.Once {
		return fmt.Errorf("--interactive cannot be used with --once")
	}

	if r.Resume && r.HasImage {
		return fmt.Errorf("--resume cannot be used with --image")
	}

	if r.Resume && r.HasQuery && r.OutputFormat == OutputFormatText {
		return fmt.Errorf("--resume cannot be used with query arguments")
	}

	if r.HasImage && r.OutputFormat == OutputFormatJSON {
		return fmt.Errorf("--image cannot be used with --headless or --output-format json")
	}

	if r.Quiet && !r.effectiveOnce() {
		return fmt.Errorf("--quiet can only be used with one-shot execution")
	}

	if r.Once {
		if r.Resume {
			return fmt.Errorf("--once cannot be used with --resume")
		}
		if r.OutputFormat == OutputFormatJSON {
			return fmt.Errorf("--once cannot be used with --headless or --output-format json")
		}
		if !r.HasQuery && !r.HasImage {
			return fmt.Errorf("query argument is required when using --once")
		}
	}

	return nil
}

// Resolve は Request から root command の実行モードを返す。
func (r Request) Resolve() (Mode, error) {
	if err := r.validate(); err != nil {
		return "", err
	}

	switch {
	case r.OutputFormat == OutputFormatJSON:
		return ModeHeadless, nil
	case r.HasImage && r.Interactive:
		return ModeInteractiveImage, nil
	case r.explicitImageOnce() || r.implicitImageOnce():
		return ModeOnceImage, nil
	case r.Once:
		return ModeOnce, nil
	case r.Resume && !r.HasQuery:
		return ModeResume, nil
	case !r.HasQuery:
		return ModeInteractive, nil
	case r.implicitTextOnce():
		return ModeOnce, nil
	default:
		return ModeInteractive, nil
	}
}

// ResolveOutputFormat は --output-format と --headless から有効な output format を返す。
func ResolveOutputFormat(flagValue string, headless bool) (string, error) {
	format := strings.ToLower(strings.TrimSpace(flagValue))
	if format == "" {
		format = OutputFormatText
	}

	switch format {
	case OutputFormatText, OutputFormatJSON:
	case "":
		format = OutputFormatText
	default:
		return "", fmt.Errorf("invalid --output-format %q (expected text or json)", flagValue)
	}

	if headless {
		return OutputFormatJSON, nil
	}

	return format, nil
}

// IsInteractive は provider setup failure を unavailable provider として扱う interactive 系 mode かを返す。
func IsInteractive(mode Mode) bool {
	switch mode {
	case ModeInteractive, ModeResume, ModeInteractiveImage:
		return true
	default:
		return false
	}
}
