package cliruntime

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/history"
	"github.com/susugadx/xelyon-cli/internal/llmcatalog"
	"github.com/susugadx/xelyon-cli/internal/setup"
)

// ErrResumeRuntimeStorageUnavailable は resume storage 初期化または参照ができないことを表す。
var ErrResumeRuntimeStorageUnavailable = errors.New("resume storage unavailable")

// ModelSelection は provider/model の CLI 選択入力である。
type ModelSelection struct {
	ProviderFlag string
	ModelFlag    string
}

// ProviderModelValidationContext は provider/model validation に必要な caller context である。
type ProviderModelValidationContext struct {
	ExplicitModel     bool
	ProviderConfigKey string
	ModelFlag         string
}

// ConfigOverrides は root flags 由来の config override 入力である。
type ConfigOverrides struct {
	LoopThresholdChanged bool
	LoopThreshold        int
	DiffLinesChanged     bool
	DiffLines            int
}

// ResumeTarget は resume 対象 session の指定である。
type ResumeTarget struct {
	SessionID string
	Last      bool
}

func debugLog(format string, args ...interface{}) {
	if os.Getenv("XELYON_DEBUG") == "1" {
		fmt.Fprintf(os.Stderr, "[DEBUG] "+format+"\n", args...)
	}
}

// ResolveProviderName は CLI flag、環境変数、config、default の順で provider 名を解決する。
func ResolveProviderName(flagValue, configValue string) string {
	if normalizedFlag := config.NormalizeProviderName(flagValue); normalizedFlag != "" {
		debugLog("provider from flag: %s -> %s", flagValue, normalizedFlag)
		return normalizedFlag
	}

	if envValue := config.NormalizeProviderName(os.Getenv("XELYON_PROVIDER")); envValue != "" {
		debugLog("provider from env: %s", envValue)
		return envValue
	}

	if normalizedConfig := config.NormalizeProviderName(configValue); normalizedConfig != "" {
		debugLog("provider from config: %s -> %s", configValue, normalizedConfig)
		return normalizedConfig
	}

	debugLog("using default provider: deepseek")
	return "deepseek"
}

// GetModel は CLI flag、環境変数、provider selection の順で model を解決する。
func GetModel(cfg *config.Config, selection ModelSelection) string {
	if selection.ModelFlag != "" {
		return selection.ModelFlag
	}
	if envModel := strings.TrimSpace(os.Getenv("XELYON_MODEL")); envModel != "" {
		return envModel
	}

	if cfg == nil {
		cfg = config.DefaultConfig()
	}

	providerName := ResolveProviderName(selection.ProviderFlag, cfg.DefaultProvider)
	return cfg.GetSelectedModelForProvider(providerName)
}

// CreateProvider は provider 名から api.Provider を生成する。
func CreateProvider(providerName string) (api.Provider, error) {
	name := config.NormalizeProviderName(providerName)
	return api.NewProvider(name)
}

// ResolveRequiredProvider は setup 済み provider だけを実 provider として返す。
func ResolveRequiredProvider(providerName string) (api.Provider, error) {
	providerName = config.NormalizeProviderName(providerName)
	if providerName == "" {
		providerName = "deepseek"
	}
	if !llmcatalog.IsKnownProvider(providerName) {
		return nil, fmt.Errorf("unknown provider: %s\nSupported providers: %s", providerName, strings.Join(config.GetDisplayProviders(), ", "))
	}
	if setup.ProviderSetupRequired(providerName) {
		return nil, errors.New(setup.ProviderSetupRequiredMessage(providerName))
	}
	provider, err := CreateProvider(providerName)
	if err != nil {
		return nil, providerCreationError(providerName, err)
	}
	return provider, nil
}

// ResolveInteractiveProvider は setup 不足 provider を interactive 用 unavailable provider として返す。
func ResolveInteractiveProvider(providerName string) (api.Provider, error) {
	providerName = config.NormalizeProviderName(providerName)
	if providerName == "" {
		providerName = "deepseek"
	}
	if !llmcatalog.IsKnownProvider(providerName) {
		return nil, fmt.Errorf("unknown provider: %s\nSupported providers: %s", providerName, strings.Join(config.GetDisplayProviders(), ", "))
	}
	if setup.ProviderSetupRequired(providerName) {
		return api.NewUnavailableProvider(providerName, setup.ProviderSetupRequiredMessage(providerName)), nil
	}
	provider, err := CreateProvider(providerName)
	if err != nil {
		if IsProviderSetupError(providerName, err) {
			return api.NewUnavailableProvider(providerName, setup.ProviderSetupRequiredMessage(providerName)), nil
		}
		return nil, providerCreationError(providerName, err)
	}
	return provider, nil
}

func providerCreationError(providerName string, err error) error {
	if err == nil {
		return nil
	}
	if strings.Contains(err.Error(), "unknown provider") {
		return fmt.Errorf("%w\nSupported providers: %s", err, strings.Join(config.GetDisplayProviders(), ", "))
	}
	if IsProviderSetupError(providerName, err) {
		return errors.New(setup.ProviderSetupRequiredMessage(providerName))
	}
	return err
}

// IsProviderSetupError は provider setup が不足している error かを判定する。
func IsProviderSetupError(providerName string, err error) bool {
	if err == nil {
		return false
	}
	providerName = config.NormalizeProviderName(providerName)
	if llmcatalog.IsKnownProvider(providerName) && setup.ProviderSetupRequired(providerName) {
		return true
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "not set") ||
		strings.Contains(message, "login required") ||
		strings.Contains(message, "not logged in")
}

// ValidateSelectedProviderModel は provider と model の組み合わせを検証する。
func ValidateSelectedProviderModel(cfg *config.Config, provider api.Provider, model string, validation ProviderModelValidationContext) error {
	if provider == nil {
		return nil
	}
	runtimeProvider := config.CanonicalProviderName(provider.Name())
	if runtimeProvider == "gemini" {
		providerKey := strings.TrimSpace(validation.ProviderConfigKey)
		if providerKey == "" {
			providerKey = runtimeProvider
		}
		return config.ValidateGeminiFunctionCallingSelection(cfg, providerKey, model)
	}
	if runtimeProvider != "azure" {
		return nil
	}

	err := config.ValidateAzureDeploymentSelection(cfg, validation.ProviderConfigKey, model, hasExplicitSelectedModel(validation))
	return selectedProviderModelValidationError(err)
}

func selectedProviderModelValidationError(err error) error {
	if err == nil {
		return nil
	}
	message := strings.TrimSuffix(err.Error(), ".")
	return fmt.Errorf("%s or pass --model <deployment>", message)
}

func hasExplicitSelectedModel(validation ProviderModelValidationContext) bool {
	return validation.ExplicitModel || HasExplicitCLIModelSelection(validation.ModelFlag)
}

// HasExplicitCLIModelSelection は CLI flag または XELYON_MODEL で model が明示されたかを返す。
func HasExplicitCLIModelSelection(modelFlag string) bool {
	return strings.TrimSpace(modelFlag) != "" || strings.TrimSpace(os.Getenv("XELYON_MODEL")) != ""
}

// LoadConfigSelection は config.yaml、環境変数、CLI flag override を合成した runtime config を返す。
func LoadConfigSelection(errWriter io.Writer, overrides ConfigOverrides) *config.Config {
	return loadConfigSelection(errWriter, overrides, config.LoadConfig)
}

// LoadConfigSelectionReadOnly は config bootstrap を行わずに runtime config を返す。
func LoadConfigSelectionReadOnly(errWriter io.Writer, overrides ConfigOverrides) *config.Config {
	return loadConfigSelection(errWriter, overrides, config.LoadConfigReadOnly)
}

func loadConfigSelection(errWriter io.Writer, overrides ConfigOverrides, load func() (*config.Config, error)) *config.Config {
	cfg, err := load()
	if err != nil {
		if errWriter != nil {
			fmt.Fprintf(errWriter, "Warning: Failed to load config: %v\n", err)
		}
		cfg = config.DefaultConfig()
	}

	cfg.ApplyEnvironmentOverrides()

	var loopPtr, diffPtr *int
	if overrides.LoopThresholdChanged {
		loopPtr = &overrides.LoopThreshold
	}
	if overrides.DiffLinesChanged {
		diffPtr = &overrides.DiffLines
	}
	cfg.ApplyFlagOverrides(loopPtr, diffPtr)
	return cfg
}

// LoadResumeSession は ResumeTarget から保存済み session を読み込む。
func LoadResumeSession(target ResumeTarget) (*history.Session, error) {
	storage, err := history.NewStorage()
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrResumeRuntimeStorageUnavailable, err)
	}

	sessionID := strings.TrimSpace(target.SessionID)
	if target.Last {
		sessionID, err = storage.GetLastResumeSession(history.ResumeListOptions{})
		if err != nil {
			if !errors.Is(err, history.ErrNoResumeSessions) {
				return nil, fmt.Errorf("%w: %v", ErrResumeRuntimeStorageUnavailable, err)
			}
			return nil, err
		}
	}
	if sessionID == "" {
		return nil, fmt.Errorf("resume session ID is required")
	}

	session, err := storage.Load(sessionID)
	if err != nil {
		return nil, fmt.Errorf("load resume session: %w", err)
	}
	return session, nil
}
