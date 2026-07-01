package cmd

import (
	"fmt"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/app"
	"github.com/susugadx/xelyon-cli/internal/config"
)

type headlessImageInputResolution struct {
	input  app.HeadlessInput
	image  *api.ImageData
	result *app.HeadlessResult
}

func resolveHeadlessImageInput(provider api.Provider, model string, input app.HeadlessInput, imagePath string) headlessImageInputResolution {
	if imagePath == "" {
		return headlessImageInputResolution{input: input}
	}

	providerSupportsImage := provider.SupportsImages()
	input = withHeadlessImageInputMetadata(input, imagePath, providerSupportsImage)
	if !providerSupportsImage {
		return headlessImageInputResolution{input: input, result: newHeadlessUnsupportedImageInputResult(provider.Name(), model, input)}
	}

	image, err := api.LoadImage(imagePath)
	if err != nil {
		result := app.NewHeadlessUsageErrorResult(
			provider.Name(),
			model,
			fmt.Sprintf("failed to load image: %v", err),
		).WithInput(input)
		return headlessImageInputResolution{input: input, result: result}
	}

	input = input.WithImage(app.NewHeadlessInputImageFromData(image, true))
	return headlessImageInputResolution{input: input, image: image}
}

func resolveHeadlessProviderSetupRequiredImageInput(providerName string, model string, input app.HeadlessInput, imagePath string) headlessImageInputResolution {
	if imagePath == "" {
		return headlessImageInputResolution{input: input}
	}

	providerSupportsImage := api.SupportsImages(providerName)
	input = withHeadlessImageInputMetadata(input, imagePath, providerSupportsImage)
	if !providerSupportsImage {
		return headlessImageInputResolution{input: input, result: newHeadlessUnsupportedImageInputResult(providerName, model, input)}
	}
	return headlessImageInputResolution{input: input}
}

func withHeadlessImageInputMetadata(input app.HeadlessInput, imagePath string, providerSupported bool) app.HeadlessInput {
	return input.WithImage(app.NewHeadlessInputImage(imagePath, "", 0, providerSupported))
}

func withHeadlessImageInputMetadataForProviderName(input app.HeadlessInput, imagePath string, providerName string) app.HeadlessInput {
	if imagePath == "" {
		return input
	}
	return withHeadlessImageInputMetadata(input, imagePath, api.SupportsImages(providerName))
}

func withHeadlessImageInputMetadataForCurrentProvider(input app.HeadlessInput, imagePath string) app.HeadlessInput {
	if imagePath == "" {
		return input
	}
	return withHeadlessImageInputMetadataForProviderName(input, imagePath, resolveHeadlessImageMetadataProviderName())
}

func resolveHeadlessImageMetadataProviderName() string {
	return resolveHeadlessImageMetadataProviderNameForFlag(providerFlag)
}

func resolveHeadlessImageMetadataProviderNameForFlag(providerFlagValue string) string {
	cfg, err := config.LoadConfigReadOnly()
	if err != nil || cfg == nil {
		cfg = config.DefaultConfig()
	}
	cfg.ApplyEnvironmentOverrides()
	return resolveProviderName(providerFlagValue, cfg.DefaultProvider)
}

func newHeadlessUnsupportedImageInputResult(providerName string, model string, input app.HeadlessInput) *app.HeadlessResult {
	return app.NewHeadlessUnsupportedCapabilityResult(
		providerName,
		model,
		fmt.Sprintf("provider %q does not support image input", providerName),
	).WithInput(input)
}
