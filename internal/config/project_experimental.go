package config

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

const providerHistoryReductionModeErrorSuffix = "expected: off, dry_run, apply, auto"

// ProviderHistoryReductionEnvVar は project-local history reduction mode を上書きする環境変数名。
const ProviderHistoryReductionEnvVar = "XELYON_PROVIDER_HISTORY_REDUCTION"

// ProjectExperimentalConfig は xelyon.yaml の experimental 設定を保持する。
type ProjectExperimentalConfig struct {
	ProviderHistoryReduction ProjectProviderHistoryReductionConfig `yaml:"provider_history_reduction,omitempty"`
}

// IsZero は experimental が省略可能かを返す。
func (c ProjectExperimentalConfig) IsZero() bool {
	return c.ProviderHistoryReduction.IsZero()
}

// ProjectProviderHistoryReductionConfig は provider-facing history reduction の実験設定。
type ProjectProviderHistoryReductionConfig struct {
	Mode ProjectProviderHistoryReductionMode `yaml:"mode,omitempty"`
}

// IsZero は provider_history_reduction が省略可能かを返す。
func (c ProjectProviderHistoryReductionConfig) IsZero() bool {
	return c.Mode == ""
}

// ProjectProviderHistoryReductionMode は xelyon.yaml/env で指定できる実験 mode。
type ProjectProviderHistoryReductionMode string

const (
	ProjectProviderHistoryReductionModeOff    ProjectProviderHistoryReductionMode = "off"
	ProjectProviderHistoryReductionModeDryRun ProjectProviderHistoryReductionMode = "dry_run"
	ProjectProviderHistoryReductionModeApply  ProjectProviderHistoryReductionMode = "apply"
	ProjectProviderHistoryReductionModeAuto   ProjectProviderHistoryReductionMode = "auto"
)

// UnmarshalYAML は provider_history_reduction.mode を検証しながら読む。
func (m *ProjectProviderHistoryReductionMode) UnmarshalYAML(value *yaml.Node) error {
	if value == nil || value.Tag == "!!null" {
		*m = ""
		return nil
	}

	var raw string
	if err := value.Decode(&raw); err != nil {
		return err
	}
	parsed, err := ParseProjectProviderHistoryReductionMode(raw)
	if err != nil {
		return err
	}
	*m = parsed
	return nil
}

// ParseProjectProviderHistoryReductionMode は project/env の provider history reduction mode を検証する。
func ParseProjectProviderHistoryReductionMode(raw string) (ProjectProviderHistoryReductionMode, error) {
	value := strings.TrimSpace(raw)
	switch ProjectProviderHistoryReductionMode(value) {
	case ProjectProviderHistoryReductionModeOff,
		ProjectProviderHistoryReductionModeDryRun,
		ProjectProviderHistoryReductionModeApply,
		ProjectProviderHistoryReductionModeAuto:
		return ProjectProviderHistoryReductionMode(value), nil
	default:
		return "", fmt.Errorf("invalid provider history reduction mode %q (%s)", value, providerHistoryReductionModeErrorSuffix)
	}
}

// NormalizeProjectProviderHistoryReductionMode は zero value を未指定、それ以外を検証済み mode に正規化する。
func NormalizeProjectProviderHistoryReductionMode(mode ProjectProviderHistoryReductionMode) (ProjectProviderHistoryReductionMode, bool, error) {
	if mode == "" {
		return ProjectProviderHistoryReductionModeOff, false, nil
	}
	parsed, err := ParseProjectProviderHistoryReductionMode(string(mode))
	if err != nil {
		return "", false, err
	}
	return parsed, true, nil
}

// ResolveProjectProviderHistoryReductionMode は env > xelyon.yaml > default off の順で mode を解決する。
func ResolveProjectProviderHistoryReductionMode(projectCfg *ProjectConfig, lookupEnv func(string) (string, bool)) (ProjectProviderHistoryReductionMode, bool, error) {
	mode := ProjectProviderHistoryReductionModeOff
	specified := false

	if projectCfg != nil {
		resolved, ok, err := NormalizeProjectProviderHistoryReductionMode(projectCfg.Experimental.ProviderHistoryReduction.Mode)
		if err != nil {
			return "", false, err
		}
		if ok {
			mode = resolved
			specified = true
		}
	}

	if lookupEnv == nil {
		lookupEnv = os.LookupEnv
	}
	if raw, ok := lookupEnv(ProviderHistoryReductionEnvVar); ok {
		resolved, err := ParseProjectProviderHistoryReductionMode(raw)
		if err != nil {
			return "", false, err
		}
		return resolved, true, nil
	}

	return mode, specified, nil
}
