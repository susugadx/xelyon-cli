package skills

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	// XelyonRoutingMetadataVersion は v1 で受け付ける routing sidecar schema version。
	XelyonRoutingMetadataVersion = 1
)

const xelyonRoutingMetadataRelativePath = "agents/xelyon.yaml"

// RoutingRole は skill router 上の候補役割。
type RoutingRole string

const (
	// RoutingRolePrimary は turn の主 workflow 候補。
	RoutingRolePrimary RoutingRole = "primary"
	// RoutingRoleSupporting は主 workflow を補助する候補。
	RoutingRoleSupporting RoutingRole = "supporting"
	// RoutingRoleGuardrail は安全確認や境界確認の候補。
	RoutingRoleGuardrail RoutingRole = "guardrail"
	// RoutingRoleAuthoring は skill / docs 作成支援の候補。
	RoutingRoleAuthoring RoutingRole = "authoring"
)

// RoutingActivation は full skill body の読み込み推奨方針。
type RoutingActivation string

const (
	// RoutingActivationManual は明示指定時だけ使う方針。
	RoutingActivationManual RoutingActivation = "manual"
	// RoutingActivationHint は runtime hint だけ出し、model に activate_skill 判断を任せる方針。
	RoutingActivationHint RoutingActivation = "hint"
	// RoutingActivationAuto は将来の自動読み込み予約方針。v1 runtime default では有効化しない。
	RoutingActivationAuto RoutingActivation = "auto"
	// RoutingActivationNever は通常推薦しない方針。
	RoutingActivationNever RoutingActivation = "never"
)

const (
	// RoutingConflictReadOnly は read-only turn と競合する skill の conflict group。
	RoutingConflictReadOnly = "read-only"
	// RoutingConflictImplementation は実装作業と競合する skill の conflict group。
	RoutingConflictImplementation = "implementation"
	// RoutingConflictFileEdit は file edit と競合する skill の conflict group。
	RoutingConflictFileEdit = "file-edit"
	// RoutingConflictReview は review turn と競合する skill の conflict group。
	RoutingConflictReview = "review"
	// RoutingConflictPlanningOnly は planning-only skill の conflict group。
	RoutingConflictPlanningOnly = "planning-only"
	// RoutingConflictAuthoring は authoring turn と競合する skill の conflict group。
	RoutingConflictAuthoring = "authoring"
	// RoutingConflictRuntimeExecution は runtime execution と競合する skill の conflict group。
	RoutingConflictRuntimeExecution = "runtime-execution"
	// RoutingConflictSecurityBoundary は security boundary と競合する skill の conflict group。
	RoutingConflictSecurityBoundary = "security-boundary"
	// RoutingConflictProviderRuntime は provider runtime と競合する skill の conflict group。
	RoutingConflictProviderRuntime = "provider-runtime"
	// RoutingConflictConfig は config 変更と競合する skill の conflict group。
	RoutingConflictConfig = "config"
)

// RoutingMetadata は XELYON 固有の optional routing sidecar を正規化した domain model。
type RoutingMetadata struct {
	Version    int
	Intents    []string
	Role       RoutingRole
	ReadOnly   bool
	Modes      []string
	Triggers   []string
	Conflicts  []string
	Activation RoutingActivation
}

type routingMetadataYAML struct {
	Version    int      `yaml:"version"`
	Intents    []string `yaml:"intents"`
	Role       string   `yaml:"role"`
	ReadOnly   bool     `yaml:"read_only"`
	Modes      []string `yaml:"modes"`
	Triggers   []string `yaml:"triggers"`
	Conflicts  []string `yaml:"conflicts"`
	Activation string   `yaml:"activation"`
}

var xelyonRoutingMetadataAllowedFields = map[string]struct{}{
	"version":    {},
	"intents":    {},
	"role":       {},
	"read_only":  {},
	"modes":      {},
	"triggers":   {},
	"conflicts":  {},
	"activation": {},
}

var knownRoutingIntentSet = stringSet(KnownRoutingIntents())
var knownRoutingModeSet = stringSet(KnownRoutingModes())
var knownRoutingRoleSet = stringSet(KnownRoutingRoles())
var knownRoutingConflictSet = stringSet(KnownRoutingConflicts())
var knownRoutingActivationSet = stringSet(KnownRoutingActivations())

// KnownRoutingIntents は v1 で既知の routing intent 一覧を返す。
func KnownRoutingIntents() []string {
	return []string{
		"code-review",
		"bug-investigation",
		"risk-scan",
		"implementation",
		"refactor",
		"cleanup",
		"test-coverage",
		"test-boundary",
		"config",
		"provider-runtime",
		"state-lifecycle",
		"concurrency-lifecycle",
		"security-boundary",
		"package-boundary",
		"skill-authoring",
		"docs-authoring",
		"planning",
	}
}

// KnownRoutingModes は v1 で既知の routing mode 一覧を返す。
func KnownRoutingModes() []string {
	return []string{
		"review",
		"implementation",
		"investigation",
		"planning",
		"authoring",
		"refactor",
		"cleanup",
		"test",
		"docs",
		"config",
	}
}

// KnownRoutingRoles は v1 で既知の routing role 一覧を返す。
func KnownRoutingRoles() []string {
	return []string{
		string(RoutingRolePrimary),
		string(RoutingRoleSupporting),
		string(RoutingRoleGuardrail),
		string(RoutingRoleAuthoring),
	}
}

// KnownRoutingConflicts は v1 で既知の conflict group 一覧を返す。
func KnownRoutingConflicts() []string {
	return []string{
		RoutingConflictReadOnly,
		RoutingConflictImplementation,
		RoutingConflictFileEdit,
		RoutingConflictReview,
		RoutingConflictPlanningOnly,
		RoutingConflictAuthoring,
		RoutingConflictRuntimeExecution,
		RoutingConflictSecurityBoundary,
		RoutingConflictProviderRuntime,
		RoutingConflictConfig,
	}
}

// KnownRoutingActivations は sidecar が表現できる activation policy 一覧を返す。
func KnownRoutingActivations() []string {
	return []string{
		string(RoutingActivationManual),
		string(RoutingActivationHint),
		string(RoutingActivationAuto),
		string(RoutingActivationNever),
	}
}

func loadXelyonRoutingMetadata(skillDir string) (*RoutingMetadata, []Diagnostic) {
	sidecarPath := xelyonRoutingMetadataPath(skillDir)
	data, err := os.ReadFile(sidecarPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, []Diagnostic{newDiagnostic(
			SeverityWarning,
			"invalid_xelyon_metadata",
			sidecarPath,
			fmt.Sprintf("failed to read agents/xelyon.yaml: %v", err),
		)}
	}
	return parseXelyonRoutingMetadataContent(sidecarPath, data)
}

func parseXelyonRoutingMetadataContent(path string, data []byte) (*RoutingMetadata, []Diagnostic) {
	diagnostics := inspectRoutingMetadataUnknownFields(path, data)

	raw := routingMetadataYAML{}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		diagnostics = append(diagnostics, newDiagnostic(
			SeverityWarning,
			"invalid_xelyon_metadata",
			path,
			fmt.Sprintf("invalid agents/xelyon.yaml: %v", err),
		))
		return nil, diagnostics
	}

	if raw.Version != XelyonRoutingMetadataVersion {
		message := fmt.Sprintf("agents/xelyon.yaml version must be %d", XelyonRoutingMetadataVersion)
		if raw.Version != 0 {
			message = fmt.Sprintf("unsupported agents/xelyon.yaml version %d; only version %d is supported", raw.Version, XelyonRoutingMetadataVersion)
		}
		diagnostics = append(diagnostics, newDiagnostic(SeverityWarning, "invalid_xelyon_metadata", path, message))
		return nil, diagnostics
	}

	metadata := &RoutingMetadata{
		Version:    XelyonRoutingMetadataVersion,
		Intents:    filterKnownRoutingValues(path, "unknown_intent", "intent", raw.Intents, knownRoutingIntentSet, &diagnostics),
		ReadOnly:   raw.ReadOnly,
		Modes:      filterKnownRoutingValues(path, "unknown_mode", "mode", raw.Modes, knownRoutingModeSet, &diagnostics),
		Triggers:   normalizeFreeformRoutingValues(raw.Triggers),
		Conflicts:  filterKnownRoutingValues(path, "unknown_conflict", "conflict", raw.Conflicts, knownRoutingConflictSet, &diagnostics),
		Activation: RoutingActivationHint,
	}
	if role := normalizeRoutingValue(raw.Role); role != "" {
		if _, ok := knownRoutingRoleSet[role]; ok {
			metadata.Role = RoutingRole(role)
		} else {
			diagnostics = append(diagnostics, newDiagnostic(
				SeverityWarning,
				"unknown_role",
				path,
				fmt.Sprintf("unknown role %q in agents/xelyon.yaml", role),
			))
		}
	}
	if activation := normalizeRoutingValue(raw.Activation); activation != "" {
		if _, ok := knownRoutingActivationSet[activation]; ok {
			metadata.Activation = RoutingActivation(activation)
		} else {
			diagnostics = append(diagnostics, newDiagnostic(
				SeverityWarning,
				"unknown_activation",
				path,
				fmt.Sprintf("unknown activation %q in agents/xelyon.yaml", activation),
			))
		}
	}
	return metadata, diagnostics
}

func inspectRoutingMetadataUnknownFields(path string, data []byte) []Diagnostic {
	var root yaml.Node
	if err := yaml.Unmarshal(data, &root); err != nil {
		return nil
	}
	node := routingMetadataDocumentNode(&root)
	if node == nil || node.Kind != yaml.MappingNode {
		return []Diagnostic{newDiagnostic(SeverityWarning, "invalid_xelyon_metadata", path, "agents/xelyon.yaml must be a mapping")}
	}

	var diagnostics []Diagnostic
	for i := 0; i+1 < len(node.Content); i += 2 {
		key := strings.TrimSpace(node.Content[i].Value)
		if key == "" {
			continue
		}
		if _, ok := xelyonRoutingMetadataAllowedFields[key]; ok {
			continue
		}
		diagnostics = append(diagnostics, newDiagnostic(
			SeverityWarning,
			"unknown_xelyon_metadata_field",
			path,
			fmt.Sprintf("unknown field %q in agents/xelyon.yaml", key),
		))
	}
	return diagnostics
}

func routingMetadataDocumentNode(root *yaml.Node) *yaml.Node {
	if root == nil {
		return nil
	}
	if root.Kind == yaml.DocumentNode {
		if len(root.Content) == 0 {
			return nil
		}
		return root.Content[0]
	}
	return root
}

func filterKnownRoutingValues(path, code, label string, values []string, known map[string]struct{}, diagnostics *[]Diagnostic) []string {
	normalized := normalizeFreeformRoutingValues(values)
	out := make([]string, 0, len(normalized))
	for _, value := range normalized {
		if _, ok := known[value]; ok {
			out = append(out, value)
			continue
		}
		*diagnostics = append(*diagnostics, newDiagnostic(
			SeverityWarning,
			code,
			path,
			fmt.Sprintf("unknown %s %q in agents/xelyon.yaml", label, value),
		))
	}
	return out
}

func normalizeFreeformRoutingValues(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		normalized := normalizeRoutingValue(value)
		if normalized == "" || slices.Contains(out, normalized) {
			continue
		}
		out = append(out, normalized)
	}
	return out
}

func normalizeRoutingValue(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func stringSet(values []string) map[string]struct{} {
	set := make(map[string]struct{}, len(values))
	for _, value := range values {
		set[value] = struct{}{}
	}
	return set
}

func xelyonRoutingMetadataPath(skillDir string) string {
	if strings.HasPrefix(skillDir, "xelyon://") {
		return strings.TrimRight(skillDir, "/") + "/" + xelyonRoutingMetadataRelativePath
	}
	return filepath.Join(cleanAbsPathOrFallback(skillDir), filepath.FromSlash(xelyonRoutingMetadataRelativePath))
}
