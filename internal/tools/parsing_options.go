package tools

import (
	"io"
	"os"
)

type parseRunOptions struct {
	registry    *Registry
	startFinder jsonToolCallStartFinder
	plan        parseToolCallExecutionPlan
	logger      *parseDebugLogger
}

type parseRuntimeSettings struct {
	debugEnabled bool
}

type parseRunDependencies struct {
	registry    *Registry
	startFinder jsonToolCallStartFinder
	plan        parseToolCallExecutionPlan
	debugOut    io.Writer
}

func resolveParseRunOptions(registry *Registry, debugOut io.Writer) parseRunOptions {
	settings := resolveParseRuntimeSettings()
	deps := newDefaultParseRunDependencies(registry, debugOut)
	return buildParseRunOptions(settings, deps)
}

func resolveParseRuntimeSettings() parseRuntimeSettings {
	return parseRuntimeSettings{
		debugEnabled: os.Getenv("XELYON_DEBUG_PARSE") == "1",
	}
}

func newDefaultParseRunDependencies(registry *Registry, debugOut io.Writer) parseRunDependencies {
	return parseRunDependencies{
		registry:    resolveRegistry(registry),
		startFinder: newDefaultJSONToolCallStartFinder(),
		plan:        defaultParseToolCallExecutionPlan(),
		debugOut:    debugOut,
	}
}

func buildParseRunOptions(settings parseRuntimeSettings, deps parseRunDependencies) parseRunOptions {
	return parseRunOptions{
		registry:    deps.registry,
		startFinder: deps.startFinder,
		plan:        deps.plan,
		logger:      newParseDebugLogger(settings.debugEnabled, deps.debugOut),
	}
}
