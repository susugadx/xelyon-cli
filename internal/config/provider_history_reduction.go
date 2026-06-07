package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// ProviderHistoryRawOutputArtifactRootEnvVar は raw output artifact store root を上書きする環境変数名。
const ProviderHistoryRawOutputArtifactRootEnvVar = "XELYON_RAW_OUTPUT_ARTIFACT_ROOT"

const (
	stableProviderHistoryReductionModeErrorSuffix     = "expected: off, dry_run, apply"
	rawOutputArtifactsModeErrorSuffix                 = "expected: off, dry_run, apply"
	rawOutputArtifactsRetentionErrorSuffix            = "expected: session"
	defaultRawOutputArtifactMaxBytes                  = 64 * 1024 * 1024
	defaultRawOutputArtifactSessionQuotaBytes         = 1024 * 1024 * 1024
	defaultRawOutputArtifactChunkBytes                = 1024 * 1024
	defaultRawOutputArtifactActiveContextBudgetTokens = 4096
	defaultRawOutputArtifactActiveContextBudgetMax    = 8192
)

// ProviderHistoryReductionMode は stable config の provider-facing history reduction mode。
type ProviderHistoryReductionMode string

const (
	// ProviderHistoryReductionModeOff は provider-facing history reduction を無効化する。
	ProviderHistoryReductionModeOff ProviderHistoryReductionMode = "off"
	// ProviderHistoryReductionModeDryRun は provider payload を変えず削減見込みだけを記録する。
	ProviderHistoryReductionModeDryRun ProviderHistoryReductionMode = "dry_run"
	// ProviderHistoryReductionModeApply は安全条件を満たす provider-facing replacement を適用する。
	ProviderHistoryReductionModeApply ProviderHistoryReductionMode = "apply"
)

// ProviderHistoryReductionModeValues は stable config で公開する mode 値を返す。
func ProviderHistoryReductionModeValues() []string {
	return []string{
		string(ProviderHistoryReductionModeOff),
		string(ProviderHistoryReductionModeDryRun),
		string(ProviderHistoryReductionModeApply),
	}
}

// UnmarshalYAML は stable provider_history_reduction.mode を検証しながら読む。
func (m *ProviderHistoryReductionMode) UnmarshalYAML(value *yaml.Node) error {
	if value == nil || value.Tag == "!!null" {
		*m = ""
		return nil
	}

	var raw string
	if err := value.Decode(&raw); err != nil {
		return err
	}
	if strings.TrimSpace(raw) == "" {
		*m = ""
		return nil
	}
	parsed, err := ParseProviderHistoryReductionMode(raw)
	if err != nil {
		return err
	}
	*m = parsed
	return nil
}

// ParseProviderHistoryReductionMode は stable config の provider history reduction mode を検証する。
func ParseProviderHistoryReductionMode(raw string) (ProviderHistoryReductionMode, error) {
	value := strings.TrimSpace(raw)
	switch ProviderHistoryReductionMode(value) {
	case ProviderHistoryReductionModeOff,
		ProviderHistoryReductionModeDryRun,
		ProviderHistoryReductionModeApply:
		return ProviderHistoryReductionMode(value), nil
	default:
		return "", fmt.Errorf("invalid provider history reduction mode %q (%s)", value, stableProviderHistoryReductionModeErrorSuffix)
	}
}

// NormalizeProviderHistoryReductionMode は空値を stable default に正規化する。
func NormalizeProviderHistoryReductionMode(mode ProviderHistoryReductionMode) (ProviderHistoryReductionMode, bool, error) {
	if mode == "" {
		return ProviderHistoryReductionModeDryRun, false, nil
	}
	parsed, err := ParseProviderHistoryReductionMode(string(mode))
	if err != nil {
		return "", false, err
	}
	return parsed, true, nil
}

// ProviderHistoryRawOutputArtifactsMode は data-bearing raw output artifact-backed compact の mode。
type ProviderHistoryRawOutputArtifactsMode string

const (
	// ProviderHistoryRawOutputArtifactsModeOff は raw output artifact-backed 候補生成を無効化する。
	ProviderHistoryRawOutputArtifactsModeOff ProviderHistoryRawOutputArtifactsMode = "off"
	// ProviderHistoryRawOutputArtifactsModeDryRun は artifact-backed 候補を report し provider payload は変えない。
	ProviderHistoryRawOutputArtifactsModeDryRun ProviderHistoryRawOutputArtifactsMode = "dry_run"
	// ProviderHistoryRawOutputArtifactsModeApply は全 gate 成立時だけ data-bearing apply compact を許可する。
	ProviderHistoryRawOutputArtifactsModeApply ProviderHistoryRawOutputArtifactsMode = "apply"
)

// ProviderHistoryRawOutputArtifactsModeValues は raw_output_artifacts.mode の公開値を返す。
func ProviderHistoryRawOutputArtifactsModeValues() []string {
	return []string{
		string(ProviderHistoryRawOutputArtifactsModeOff),
		string(ProviderHistoryRawOutputArtifactsModeDryRun),
		string(ProviderHistoryRawOutputArtifactsModeApply),
	}
}

// UnmarshalYAML は raw_output_artifacts.mode を検証しながら読む。
func (m *ProviderHistoryRawOutputArtifactsMode) UnmarshalYAML(value *yaml.Node) error {
	if value == nil || value.Tag == "!!null" {
		*m = ""
		return nil
	}
	var raw string
	if err := value.Decode(&raw); err != nil {
		return err
	}
	if strings.TrimSpace(raw) == "" {
		*m = ""
		return nil
	}
	parsed, err := ParseProviderHistoryRawOutputArtifactsMode(raw)
	if err != nil {
		return err
	}
	*m = parsed
	return nil
}

// ParseProviderHistoryRawOutputArtifactsMode は raw_output_artifacts.mode を検証する。
func ParseProviderHistoryRawOutputArtifactsMode(raw string) (ProviderHistoryRawOutputArtifactsMode, error) {
	value := strings.TrimSpace(raw)
	switch ProviderHistoryRawOutputArtifactsMode(value) {
	case ProviderHistoryRawOutputArtifactsModeOff,
		ProviderHistoryRawOutputArtifactsModeDryRun,
		ProviderHistoryRawOutputArtifactsModeApply:
		return ProviderHistoryRawOutputArtifactsMode(value), nil
	default:
		return "", fmt.Errorf("invalid provider history raw output artifacts mode %q (%s)", value, rawOutputArtifactsModeErrorSuffix)
	}
}

// NormalizeProviderHistoryRawOutputArtifactsMode は空値を raw_output_artifacts default に正規化する。
func NormalizeProviderHistoryRawOutputArtifactsMode(mode ProviderHistoryRawOutputArtifactsMode) (ProviderHistoryRawOutputArtifactsMode, bool, error) {
	if mode == "" {
		return ProviderHistoryRawOutputArtifactsModeDryRun, false, nil
	}
	parsed, err := ParseProviderHistoryRawOutputArtifactsMode(string(mode))
	if err != nil {
		return "", false, err
	}
	return parsed, true, nil
}

// ProviderHistoryRawOutputArtifactsRetention は raw output artifact retention policy。
type ProviderHistoryRawOutputArtifactsRetention string

const (
	// ProviderHistoryRawOutputArtifactsRetentionSession は session-scoped retention。
	ProviderHistoryRawOutputArtifactsRetentionSession ProviderHistoryRawOutputArtifactsRetention = "session"
)

// UnmarshalYAML は raw_output_artifacts.retention を検証しながら読む。
func (r *ProviderHistoryRawOutputArtifactsRetention) UnmarshalYAML(value *yaml.Node) error {
	if value == nil || value.Tag == "!!null" {
		*r = ""
		return nil
	}
	var raw string
	if err := value.Decode(&raw); err != nil {
		return err
	}
	if strings.TrimSpace(raw) == "" {
		*r = ""
		return nil
	}
	parsed, err := ParseProviderHistoryRawOutputArtifactsRetention(raw)
	if err != nil {
		return err
	}
	*r = parsed
	return nil
}

// ParseProviderHistoryRawOutputArtifactsRetention は raw_output_artifacts.retention を検証する。
func ParseProviderHistoryRawOutputArtifactsRetention(raw string) (ProviderHistoryRawOutputArtifactsRetention, error) {
	value := strings.TrimSpace(raw)
	switch ProviderHistoryRawOutputArtifactsRetention(value) {
	case ProviderHistoryRawOutputArtifactsRetentionSession:
		return ProviderHistoryRawOutputArtifactsRetention(value), nil
	default:
		return "", fmt.Errorf("invalid provider history raw output artifacts retention %q (%s)", value, rawOutputArtifactsRetentionErrorSuffix)
	}
}

// ProviderHistoryRawOutputArtifactsConfig は data-bearing raw output artifact-backed compact の設定。
type ProviderHistoryRawOutputArtifactsConfig struct {
	Mode                         ProviderHistoryRawOutputArtifactsMode      `yaml:"mode,omitempty"`
	Root                         string                                     `yaml:"root,omitempty"`
	MaxArtifactBytes             int                                        `yaml:"max_artifact_bytes,omitempty"`
	SessionQuotaBytes            int                                        `yaml:"session_quota_bytes,omitempty"`
	ChunkBytes                   int                                        `yaml:"chunk_bytes,omitempty"`
	ActiveContextBudgetTokens    int                                        `yaml:"active_context_budget_tokens,omitempty"`
	ActiveContextBudgetMaxTokens int                                        `yaml:"active_context_budget_max_tokens,omitempty"`
	Retention                    ProviderHistoryRawOutputArtifactsRetention `yaml:"retention,omitempty"`
}

// DefaultProviderHistoryRawOutputArtifactsConfig は raw_output_artifacts の stable default を返す。
func DefaultProviderHistoryRawOutputArtifactsConfig() ProviderHistoryRawOutputArtifactsConfig {
	return defaultProviderHistoryRawOutputArtifactsConfig()
}

// IsZero は raw_output_artifacts が省略可能かを返す。
func (c ProviderHistoryRawOutputArtifactsConfig) IsZero() bool {
	return c.Mode == "" &&
		strings.TrimSpace(c.Root) == "" &&
		c.MaxArtifactBytes == 0 &&
		c.SessionQuotaBytes == 0 &&
		c.ChunkBytes == 0 &&
		c.ActiveContextBudgetTokens == 0 &&
		c.ActiveContextBudgetMaxTokens == 0 &&
		c.Retention == ""
}

// UnmarshalYAML は raw_output_artifacts を読み、明示された不正な numeric budget を拒否する。
func (c *ProviderHistoryRawOutputArtifactsConfig) UnmarshalYAML(value *yaml.Node) error {
	if value == nil || value.Tag == "!!null" {
		*c = ProviderHistoryRawOutputArtifactsConfig{}
		return nil
	}
	if value.Kind != yaml.MappingNode {
		return fmt.Errorf("invalid provider_history_reduction.raw_output_artifacts section")
	}

	var parsed ProviderHistoryRawOutputArtifactsConfig
	for i := 0; i+1 < len(value.Content); i += 2 {
		key := value.Content[i].Value
		child := value.Content[i+1]
		switch key {
		case "mode":
			if err := child.Decode(&parsed.Mode); err != nil {
				return err
			}
		case "root":
			if err := child.Decode(&parsed.Root); err != nil {
				return err
			}
		case "max_artifact_bytes":
			if err := decodePositiveProviderHistoryInt(child, key, &parsed.MaxArtifactBytes); err != nil {
				return err
			}
		case "session_quota_bytes":
			if err := decodePositiveProviderHistoryInt(child, key, &parsed.SessionQuotaBytes); err != nil {
				return err
			}
		case "chunk_bytes":
			if err := decodePositiveProviderHistoryInt(child, key, &parsed.ChunkBytes); err != nil {
				return err
			}
		case "active_context_budget_tokens":
			if err := decodePositiveProviderHistoryInt(child, key, &parsed.ActiveContextBudgetTokens); err != nil {
				return err
			}
		case "active_context_budget_max_tokens":
			if err := decodePositiveProviderHistoryInt(child, key, &parsed.ActiveContextBudgetMaxTokens); err != nil {
				return err
			}
		case "retention":
			if err := child.Decode(&parsed.Retention); err != nil {
				return err
			}
		}
	}
	if err := validateProviderHistoryRawOutputArtifactsConfig(parsed); err != nil {
		return err
	}
	*c = parsed
	return nil
}

func decodePositiveProviderHistoryInt(node *yaml.Node, key string, out *int) error {
	var value int
	if err := node.Decode(&value); err != nil {
		return err
	}
	if value <= 0 {
		return fmt.Errorf("invalid provider_history_reduction.raw_output_artifacts.%s %d (expected: > 0)", key, value)
	}
	*out = value
	return nil
}

func validateProviderHistoryRawOutputArtifactsConfig(cfg ProviderHistoryRawOutputArtifactsConfig) error {
	if err := validateProviderHistoryRawOutputArtifactRoot(cfg.Root); err != nil {
		return err
	}
	if cfg.SessionQuotaBytes > 0 && cfg.MaxArtifactBytes > 0 && cfg.SessionQuotaBytes < cfg.MaxArtifactBytes {
		return fmt.Errorf("invalid provider_history_reduction.raw_output_artifacts.session_quota_bytes %d (expected: >= max_artifact_bytes)", cfg.SessionQuotaBytes)
	}
	if cfg.ChunkBytes > 0 && cfg.MaxArtifactBytes > 0 && cfg.ChunkBytes > cfg.MaxArtifactBytes {
		return fmt.Errorf("invalid provider_history_reduction.raw_output_artifacts.chunk_bytes %d (expected: <= max_artifact_bytes)", cfg.ChunkBytes)
	}
	if cfg.ActiveContextBudgetMaxTokens > 0 && cfg.ActiveContextBudgetTokens > 0 && cfg.ActiveContextBudgetMaxTokens < cfg.ActiveContextBudgetTokens {
		return fmt.Errorf("invalid provider_history_reduction.raw_output_artifacts.active_context_budget_max_tokens %d (expected: >= active_context_budget_tokens)", cfg.ActiveContextBudgetMaxTokens)
	}
	if cfg.Retention != "" {
		if _, err := ParseProviderHistoryRawOutputArtifactsRetention(string(cfg.Retention)); err != nil {
			return err
		}
	}
	if cfg.Mode != "" {
		if _, err := ParseProviderHistoryRawOutputArtifactsMode(string(cfg.Mode)); err != nil {
			return err
		}
	}
	return nil
}

func validateProviderHistoryRawOutputArtifactRoot(root string) error {
	root = strings.TrimSpace(root)
	if root == "" {
		return nil
	}
	clean := filepath.Clean(root)
	if clean == "." || clean == string(filepath.Separator) || !filepath.IsAbs(clean) {
		return fmt.Errorf("invalid provider_history_reduction.raw_output_artifacts.root %q (expected: absolute path below a directory)", root)
	}
	return nil
}

// ProviderHistoryReductionConfig は provider-facing history reduction の stable 設定。
type ProviderHistoryReductionConfig struct {
	Mode               ProviderHistoryReductionMode            `yaml:"mode"`              // off / dry_run / apply。auto は stable config では公開しない。
	RehydrateContext   bool                                    `yaml:"rehydrate_context"` // 省略された古い evidence を request-local active context として戻す。
	RawOutputArtifacts ProviderHistoryRawOutputArtifactsConfig `yaml:"raw_output_artifacts"`
}

// ProjectStableProviderHistoryReductionConfig は xelyon.yaml の stable provider history reduction 設定。
type ProjectStableProviderHistoryReductionConfig struct {
	Mode                ProviderHistoryReductionMode            `yaml:"mode,omitempty"`
	RehydrateContext    ProjectProviderHistoryRehydrateContext  `yaml:"rehydrate_context,omitempty"`
	RawOutputArtifacts  ProviderHistoryRawOutputArtifactsConfig `yaml:"raw_output_artifacts,omitempty"`
	RehydrateContextSet bool                                    `yaml:"-"`
}

// IsZero は stable provider_history_reduction が省略可能かを返す。
func (c ProjectStableProviderHistoryReductionConfig) IsZero() bool {
	return c.Mode == "" && !c.RehydrateContextSet && c.RawOutputArtifacts.IsZero()
}

// UnmarshalYAML は stable project provider_history_reduction を読み、明示 false を記録する。
func (c *ProjectStableProviderHistoryReductionConfig) UnmarshalYAML(value *yaml.Node) error {
	if value == nil || value.Tag == "!!null" {
		*c = ProjectStableProviderHistoryReductionConfig{}
		return nil
	}
	if value.Kind != yaml.MappingNode {
		return fmt.Errorf("invalid provider_history_reduction section")
	}

	var parsed ProjectStableProviderHistoryReductionConfig
	for i := 0; i+1 < len(value.Content); i += 2 {
		key := value.Content[i].Value
		child := value.Content[i+1]
		switch key {
		case "mode":
			if err := child.Decode(&parsed.Mode); err != nil {
				return err
			}
		case "rehydrate_context":
			if err := child.Decode(&parsed.RehydrateContext); err != nil {
				return err
			}
			parsed.RehydrateContextSet = true
		case "raw_output_artifacts":
			if err := child.Decode(&parsed.RawOutputArtifacts); err != nil {
				return err
			}
		}
	}
	*c = parsed
	return nil
}

// MarshalYAML は明示 false の rehydrate_context を省略せずに保存する。
func (c ProjectStableProviderHistoryReductionConfig) MarshalYAML() (interface{}, error) {
	type projectStableProviderHistoryReductionYAML struct {
		Mode               ProviderHistoryReductionMode            `yaml:"mode,omitempty"`
		RehydrateContext   *ProjectProviderHistoryRehydrateContext `yaml:"rehydrate_context,omitempty"`
		RawOutputArtifacts ProviderHistoryRawOutputArtifactsConfig `yaml:"raw_output_artifacts,omitempty"`
	}
	out := projectStableProviderHistoryReductionYAML{Mode: c.Mode, RawOutputArtifacts: c.RawOutputArtifacts}
	if c.RehydrateContextSet || bool(c.RehydrateContext) {
		rehydrateContext := c.RehydrateContext
		out.RehydrateContext = &rehydrateContext
	}
	return out, nil
}

// ResolveProviderHistoryReductionMode は env > stable project > experimental project > global/default の順で mode を解決する。
func ResolveProviderHistoryReductionMode(globalCfg *Config, projectCfg *ProjectConfig, lookupEnv func(string) (string, bool)) (ProjectProviderHistoryReductionMode, bool, error) {
	mode, err := projectProviderHistoryReductionModeFromGlobalConfig(globalCfg)
	if err != nil {
		return "", false, err
	}
	specified := false

	if projectCfg != nil {
		switch {
		case !projectCfg.ProviderHistoryReduction.IsZero():
			if projectCfg.ProviderHistoryReduction.Mode != "" {
				mode = projectProviderHistoryReductionModeFromStableMode(projectCfg.ProviderHistoryReduction.Mode)
			}
			specified = true
		case !projectCfg.Experimental.ProviderHistoryReduction.IsZero():
			resolved, ok, err := NormalizeProjectProviderHistoryReductionMode(projectCfg.Experimental.ProviderHistoryReduction.Mode)
			if err != nil {
				return "", false, err
			}
			if ok {
				mode = resolved
			}
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

// ResolveProviderHistoryRehydrateContext は env > stable project > experimental project > global/default の順で rehydrate_context を解決する。
func ResolveProviderHistoryRehydrateContext(globalCfg *Config, projectCfg *ProjectConfig, lookupEnv func(string) (string, bool)) (bool, error) {
	enabled := providerHistoryRehydrateContextFromGlobalConfig(globalCfg)

	if projectCfg != nil {
		switch {
		case !projectCfg.ProviderHistoryReduction.IsZero():
			if projectCfg.ProviderHistoryReduction.RehydrateContextSet {
				enabled = bool(projectCfg.ProviderHistoryReduction.RehydrateContext)
			}
		case !projectCfg.Experimental.ProviderHistoryReduction.IsZero():
			if projectCfg.Experimental.ProviderHistoryReduction.RehydrateContextSet {
				enabled = bool(projectCfg.Experimental.ProviderHistoryReduction.RehydrateContext)
			} else {
				enabled = false
			}
		}
	}

	if lookupEnv == nil {
		lookupEnv = os.LookupEnv
	}
	if raw, ok := lookupEnv(ProviderHistoryRehydrateContextEnvVar); ok {
		if strings.TrimSpace(raw) == "" {
			return enabled, nil
		}
		return ParseProjectProviderHistoryRehydrateContext(raw)
	}

	return enabled, nil
}

// ResolveProviderHistoryRawOutputArtifactsConfig は env > stable project > global/default の順で raw_output_artifacts を解決する。
func ResolveProviderHistoryRawOutputArtifactsConfig(globalCfg *Config, projectCfg *ProjectConfig) (ProviderHistoryRawOutputArtifactsConfig, bool, error) {
	return ResolveProviderHistoryRawOutputArtifactsConfigWithEnv(globalCfg, projectCfg, nil)
}

// ResolveProviderHistoryRawOutputArtifactsConfigWithEnv は env > stable project > global/default の順で raw_output_artifacts を解決する。
func ResolveProviderHistoryRawOutputArtifactsConfigWithEnv(globalCfg *Config, projectCfg *ProjectConfig, lookupEnv func(string) (string, bool)) (ProviderHistoryRawOutputArtifactsConfig, bool, error) {
	resolved := defaultProviderHistoryRawOutputArtifactsConfig()
	if globalCfg != nil {
		resolved = normalizeProviderHistoryRawOutputArtifactsConfig(globalCfg.ProviderHistoryReduction.RawOutputArtifacts)
	}
	specified := false
	if projectCfg != nil && !projectCfg.ProviderHistoryReduction.RawOutputArtifacts.IsZero() {
		resolved = mergeProviderHistoryRawOutputArtifactsConfig(resolved, projectCfg.ProviderHistoryReduction.RawOutputArtifacts)
		specified = true
	}
	if lookupEnv == nil {
		lookupEnv = os.LookupEnv
	}
	if rawRoot, ok := lookupEnv(ProviderHistoryRawOutputArtifactRootEnvVar); ok {
		if root := strings.TrimSpace(rawRoot); root != "" {
			resolved.Root = root
			specified = true
		}
	}
	if err := validateProviderHistoryRawOutputArtifactsConfig(resolved); err != nil {
		return ProviderHistoryRawOutputArtifactsConfig{}, false, err
	}
	return resolved, specified, nil
}

func normalizeProviderHistoryRawOutputArtifactsConfig(cfg ProviderHistoryRawOutputArtifactsConfig) ProviderHistoryRawOutputArtifactsConfig {
	return mergeProviderHistoryRawOutputArtifactsConfig(defaultProviderHistoryRawOutputArtifactsConfig(), cfg)
}

// NormalizeProviderHistoryRawOutputArtifactsConfig は raw_output_artifacts を stable default で補完する。
func NormalizeProviderHistoryRawOutputArtifactsConfig(cfg ProviderHistoryRawOutputArtifactsConfig) ProviderHistoryRawOutputArtifactsConfig {
	return normalizeProviderHistoryRawOutputArtifactsConfig(cfg)
}

func mergeProviderHistoryRawOutputArtifactsConfig(base, override ProviderHistoryRawOutputArtifactsConfig) ProviderHistoryRawOutputArtifactsConfig {
	out := base
	if override.Mode != "" {
		out.Mode = override.Mode
	}
	if strings.TrimSpace(override.Root) != "" {
		out.Root = strings.TrimSpace(override.Root)
	}
	if override.MaxArtifactBytes > 0 {
		out.MaxArtifactBytes = override.MaxArtifactBytes
	}
	if override.SessionQuotaBytes > 0 {
		out.SessionQuotaBytes = override.SessionQuotaBytes
	}
	if override.ChunkBytes > 0 {
		out.ChunkBytes = override.ChunkBytes
	}
	if override.ActiveContextBudgetTokens > 0 {
		out.ActiveContextBudgetTokens = override.ActiveContextBudgetTokens
	}
	if override.ActiveContextBudgetMaxTokens > 0 {
		out.ActiveContextBudgetMaxTokens = override.ActiveContextBudgetMaxTokens
	}
	if override.Retention != "" {
		out.Retention = override.Retention
	}
	return out
}

func projectProviderHistoryReductionModeFromGlobalConfig(globalCfg *Config) (ProjectProviderHistoryReductionMode, error) {
	mode := defaultProviderHistoryReductionConfig().Mode
	if globalCfg != nil {
		mode = globalCfg.ProviderHistoryReduction.Mode
	}
	normalized, _, err := NormalizeProviderHistoryReductionMode(mode)
	if err != nil {
		return "", err
	}
	return projectProviderHistoryReductionModeFromStableMode(normalized), nil
}

func providerHistoryRehydrateContextFromGlobalConfig(globalCfg *Config) bool {
	if globalCfg == nil {
		return defaultProviderHistoryReductionConfig().RehydrateContext
	}
	return globalCfg.ProviderHistoryReduction.RehydrateContext
}

func projectProviderHistoryReductionModeFromStableMode(mode ProviderHistoryReductionMode) ProjectProviderHistoryReductionMode {
	switch mode {
	case ProviderHistoryReductionModeApply:
		return ProjectProviderHistoryReductionModeApply
	case ProviderHistoryReductionModeDryRun:
		return ProjectProviderHistoryReductionModeDryRun
	default:
		return ProjectProviderHistoryReductionModeOff
	}
}

func validateProviderHistoryReductionIssues(cfg *Config) []ValidationIssue {
	if cfg == nil {
		return nil
	}
	var issues []ValidationIssue
	mode := strings.TrimSpace(string(cfg.ProviderHistoryReduction.Mode))
	if mode != "" {
		if _, err := ParseProviderHistoryReductionMode(mode); err != nil {
			issues = append(issues, ValidationIssue{
				Field:      "provider_history_reduction.mode",
				Value:      mode,
				Message:    "無効な provider history reduction mode です (有効: off, dry_run, apply)",
				Suggestion: string(ProviderHistoryReductionModeDryRun),
				Severity:   ValidationSeverityError,
				CanAutoFix: true,
				FixedValue: ProviderHistoryReductionModeDryRun,
			})
		}
	}
	issues = append(issues, validateProviderHistoryRawOutputArtifactsIssues(cfg.ProviderHistoryReduction.RawOutputArtifacts)...)
	return issues
}

func validateProviderHistoryRawOutputArtifactsIssues(cfg ProviderHistoryRawOutputArtifactsConfig) []ValidationIssue {
	var issues []ValidationIssue
	mode := strings.TrimSpace(string(cfg.Mode))
	if mode != "" {
		if _, err := ParseProviderHistoryRawOutputArtifactsMode(mode); err != nil {
			issues = append(issues, ValidationIssue{
				Field:      "provider_history_reduction.raw_output_artifacts.mode",
				Value:      mode,
				Message:    "無効な raw output artifacts mode です (有効: off, dry_run, apply)",
				Suggestion: string(ProviderHistoryRawOutputArtifactsModeDryRun),
				Severity:   ValidationSeverityError,
				CanAutoFix: true,
				FixedValue: ProviderHistoryRawOutputArtifactsModeDryRun,
			})
		}
	}
	issues = append(issues, validateProviderHistoryPositiveIntIssue("provider_history_reduction.raw_output_artifacts.max_artifact_bytes", cfg.MaxArtifactBytes, defaultRawOutputArtifactMaxBytes)...)
	issues = append(issues, validateProviderHistoryPositiveIntIssue("provider_history_reduction.raw_output_artifacts.session_quota_bytes", cfg.SessionQuotaBytes, defaultRawOutputArtifactSessionQuotaBytes)...)
	issues = append(issues, validateProviderHistoryPositiveIntIssue("provider_history_reduction.raw_output_artifacts.chunk_bytes", cfg.ChunkBytes, defaultRawOutputArtifactChunkBytes)...)
	issues = append(issues, validateProviderHistoryPositiveIntIssue("provider_history_reduction.raw_output_artifacts.active_context_budget_tokens", cfg.ActiveContextBudgetTokens, defaultRawOutputArtifactActiveContextBudgetTokens)...)
	issues = append(issues, validateProviderHistoryPositiveIntIssue("provider_history_reduction.raw_output_artifacts.active_context_budget_max_tokens", cfg.ActiveContextBudgetMaxTokens, defaultRawOutputArtifactActiveContextBudgetMax)...)
	if cfg.SessionQuotaBytes > 0 && cfg.MaxArtifactBytes > 0 && cfg.SessionQuotaBytes < cfg.MaxArtifactBytes {
		issues = append(issues, providerHistoryRawOutputArtifactsIssue("provider_history_reduction.raw_output_artifacts.session_quota_bytes", cfg.SessionQuotaBytes, "session_quota_bytes は max_artifact_bytes 以上にしてください", defaultRawOutputArtifactSessionQuotaBytes))
	}
	if err := validateProviderHistoryRawOutputArtifactRoot(cfg.Root); err != nil {
		issues = append(issues, ValidationIssue{
			Field:    "provider_history_reduction.raw_output_artifacts.root",
			Value:    strings.TrimSpace(cfg.Root),
			Message:  "raw output artifact root は absolute path を指定してください",
			Severity: ValidationSeverityError,
		})
	}
	if cfg.ChunkBytes > 0 && cfg.MaxArtifactBytes > 0 && cfg.ChunkBytes > cfg.MaxArtifactBytes {
		issues = append(issues, providerHistoryRawOutputArtifactsIssue("provider_history_reduction.raw_output_artifacts.chunk_bytes", cfg.ChunkBytes, "chunk_bytes は max_artifact_bytes 以下にしてください", defaultRawOutputArtifactChunkBytes))
	}
	if cfg.ActiveContextBudgetMaxTokens > 0 && cfg.ActiveContextBudgetTokens > 0 && cfg.ActiveContextBudgetMaxTokens < cfg.ActiveContextBudgetTokens {
		issues = append(issues, providerHistoryRawOutputArtifactsIssue("provider_history_reduction.raw_output_artifacts.active_context_budget_max_tokens", cfg.ActiveContextBudgetMaxTokens, "active_context_budget_max_tokens は active_context_budget_tokens 以上にしてください", defaultRawOutputArtifactActiveContextBudgetMax))
	}
	retention := strings.TrimSpace(string(cfg.Retention))
	if retention != "" {
		if _, err := ParseProviderHistoryRawOutputArtifactsRetention(retention); err != nil {
			issues = append(issues, ValidationIssue{
				Field:      "provider_history_reduction.raw_output_artifacts.retention",
				Value:      retention,
				Message:    "無効な raw output artifacts retention です (有効: session)",
				Suggestion: string(ProviderHistoryRawOutputArtifactsRetentionSession),
				Severity:   ValidationSeverityError,
				CanAutoFix: true,
				FixedValue: ProviderHistoryRawOutputArtifactsRetentionSession,
			})
		}
	}
	return issues
}

func validateProviderHistoryPositiveIntIssue(field string, value, fixed int) []ValidationIssue {
	if value > 0 {
		return nil
	}
	return []ValidationIssue{providerHistoryRawOutputArtifactsIssue(field, value, "0 より大きい値を指定してください", fixed)}
}

func providerHistoryRawOutputArtifactsIssue(field string, value int, message string, fixed int) ValidationIssue {
	return ValidationIssue{
		Field:      field,
		Value:      fmt.Sprintf("%d", value),
		Message:    message,
		Suggestion: fmt.Sprintf("%d", fixed),
		Severity:   ValidationSeverityError,
		CanAutoFix: true,
		FixedValue: fixed,
	}
}

func providerHistoryReductionModeAutoFixer() configAutoFixer {
	return func(cfg *Config, value any) bool {
		mode, ok := value.(ProviderHistoryReductionMode)
		if !ok {
			return false
		}
		cfg.ProviderHistoryReduction.Mode = mode
		return true
	}
}

func providerHistoryRawOutputArtifactsModeAutoFixer() configAutoFixer {
	return func(cfg *Config, value any) bool {
		mode, ok := value.(ProviderHistoryRawOutputArtifactsMode)
		if !ok {
			return false
		}
		cfg.ProviderHistoryReduction.RawOutputArtifacts.Mode = mode
		return true
	}
}

func providerHistoryRawOutputArtifactsRetentionAutoFixer() configAutoFixer {
	return func(cfg *Config, value any) bool {
		retention, ok := value.(ProviderHistoryRawOutputArtifactsRetention)
		if !ok {
			return false
		}
		cfg.ProviderHistoryReduction.RawOutputArtifacts.Retention = retention
		return true
	}
}
