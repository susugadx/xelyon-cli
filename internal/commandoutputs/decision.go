package commandoutputs

import "strings"

// DecisionAction は command output に対する commandoutputs の判断を表す。
type DecisionAction string

const (
	// DecisionKeepRaw は provider-facing payload をこの層では削らない判断を表す。
	DecisionKeepRaw DecisionAction = "keep_raw"
	// DecisionInlineCompact は artifact なしで安全に compact / placeholder 化できる判断を表す。
	DecisionInlineCompact DecisionAction = "inline_compact"
	// DecisionArtifactBackedCandidate は apply compact に raw artifact と rehydrate が必須な候補を表す。
	DecisionArtifactBackedCandidate DecisionAction = "artifact_backed_candidate"
)

// SemanticRole は command output の provider-facing 意味役割を表す。
type SemanticRole string

const (
	// SemanticRoleValidationLog は test/build/lint などの検証ログを表す。
	SemanticRoleValidationLog SemanticRole = "validation_log"
	// SemanticRoleOperationLog は side-effect や実行結果ログを表す。
	SemanticRoleOperationLog SemanticRole = "operation_log"
	// SemanticRoleDataBearing は response/query result 自体が証拠になる output を表す。
	SemanticRoleDataBearing SemanticRole = "data_bearing"
	// SemanticRoleSensitive は secret/private 情報を含む可能性が高い output を表す。
	SemanticRoleSensitive SemanticRole = "sensitive"
	// SemanticRoleSideEffect は mutation/deploy/package install などの副作用 output を表す。
	SemanticRoleSideEffect SemanticRole = "side_effect"
	// SemanticRoleUnknown は安全な分類ができない output を表す。
	SemanticRoleUnknown SemanticRole = "unknown"
)

// DatabaseSubfamily は database command output の細分類を表す。
type DatabaseSubfamily string

const (
	// DatabaseSubfamilyQueryResult は SELECT などの query result を表す。
	DatabaseSubfamilyQueryResult DatabaseSubfamily = "database_query_result"
	// DatabaseSubfamilySchemaResult は schema / table metadata result を表す。
	DatabaseSubfamilySchemaResult DatabaseSubfamily = "database_schema_result"
	// DatabaseSubfamilyOperationLog は database write/maintenance operation log を表す。
	DatabaseSubfamilyOperationLog DatabaseSubfamily = "database_operation_log"
	// DatabaseSubfamilyMigrationLog は migration command log を表す。
	DatabaseSubfamilyMigrationLog DatabaseSubfamily = "database_migration_log"
	// DatabaseSubfamilyConnectionError は connection/auth failure log を表す。
	DatabaseSubfamilyConnectionError DatabaseSubfamily = "database_connection_error"
	// DatabaseSubfamilyUnknown は database command だが安全な細分類ができない output を表す。
	DatabaseSubfamilyUnknown DatabaseSubfamily = "database_unknown"
)

// ArtifactPolicy は apply compact 前に providerhistory が満たすべき artifact 条件を表す。
type ArtifactPolicy struct {
	Eligible         bool
	RequiredForApply bool
	Reason           string
	ExcerptPolicy    string
}

// ReplacementPlan は inline compact の表示方針を表す。
type ReplacementPlan struct {
	Kind       string
	Reason     string
	Classifier string
	Text       string
}

// DecisionEvidence は分類に使った主要 signal を表す。
type DecisionEvidence struct {
	FailureSignal string
	SuccessSignal string
}

// Decision は command output classification / compact strategy の source of truth。
//
// zero value は無効で、呼び出し側は Decide の戻り値だけを使う。
type Decision struct {
	Action          DecisionAction
	SemanticRole    SemanticRole
	Family          string
	Subfamily       string
	Classifier      string
	KeepReason      string
	ReplacementPlan ReplacementPlan
	ArtifactPolicy  ArtifactPolicy
	Preconditions   []string
	Evidence        DecisionEvidence

	replacement Replacement
}

// Replacement は inline compact 用 replacement を返す。
func (d Decision) Replacement() (Replacement, bool) {
	if d.Action != DecisionInlineCompact || d.replacement.Text() == "" {
		return Replacement{}, false
	}
	return d.replacement, true
}

// Decide は command output の semantic role / compact strategy / artifact precondition を判定する。
func Decide(req Request) Decision {
	rawCommand := strings.TrimSpace(req.command)
	content := stripANSI(req.content)
	if rawCommand == "" {
		return keepDecision(commandFamilyUnknown, SemanticRoleUnknown, "", "missing_command_argument")
	}
	if strings.TrimSpace(content) == "" {
		return keepDecision(commandFamilyUnknown, SemanticRoleUnknown, "", "empty_command_output")
	}

	family := classifyCommandFamily(rawCommand)
	switch family {
	case commandFamilySensitive:
		return decideSensitiveCommand(rawCommand, content, family)
	case commandFamilyNetwork:
		return dataBearingDecision(family, "", "network_response", networkDataBearingKeepReason, "network_first_last")
	case commandFamilyDatabase:
		return decideDatabaseCommand(rawCommand, content, family)
	case commandFamilyPackage, commandFamilyDeploy:
		return decideSideEffectCommand(rawCommand, content, family)
	default:
		return decideInlineOrKeepCommand(rawCommand, content, family)
	}
}

func decideSensitiveCommand(command, content string, family commandFamily) Decision {
	return keepDecision(family, SemanticRoleSensitive, "", sensitiveOutputKeepReason)
}

func decideSideEffectCommand(command, content string, family commandFamily) Decision {
	failure := classifyFailure(command, content, family)
	if failure != "" {
		if replacement, ok := buildFailureCompact(command, content, family, failure); ok {
			return inlineDecision(family, SemanticRoleSideEffect, "", failure, replacement, DecisionEvidence{FailureSignal: failure})
		}
		return keepDecision(family, SemanticRoleSideEffect, "", failure+"_not_large")
	}
	if !looksLikeSuccessfulOutput(content) {
		return keepDecision(family, SemanticRoleSideEffect, "", "command_output_not_success")
	}
	switch family {
	case commandFamilyPackage:
		replacement, reason, ok := buildSafePlaceholder(command, content, "omit_successful_package_command_output", "package_success", "package_install", "side-effect")
		if !ok {
			return keepDecision(family, SemanticRoleSideEffect, "", reason)
		}
		return inlineDecision(family, SemanticRoleSideEffect, "", "package_install", replacement, DecisionEvidence{SuccessSignal: "success"})
	case commandFamilyDeploy:
		replacement, reason, ok := buildSafePlaceholder(command, content, "omit_successful_deploy_command_output", "deploy_success", "deploy", "side-effect")
		if !ok {
			return keepDecision(family, SemanticRoleSideEffect, "", reason)
		}
		return inlineDecision(family, SemanticRoleSideEffect, "", "deploy", replacement, DecisionEvidence{SuccessSignal: "success"})
	default:
		return keepDecision(family, SemanticRoleSideEffect, "", "command_output_unknown_skip")
	}
}

func decideDatabaseCommand(command, content string, family commandFamily) Decision {
	subfamily := classifyDatabaseSubfamily(command, content)
	switch subfamily {
	case DatabaseSubfamilyQueryResult:
		return dataBearingDecision(family, string(subfamily), "database_query_result", databaseDataBearingKeepReason, "database_rows_first_last")
	case DatabaseSubfamilySchemaResult:
		return dataBearingDecision(family, string(subfamily), "database_schema_result", databaseDataBearingKeepReason, "database_schema_first_last")
	case DatabaseSubfamilyOperationLog:
		return decideDatabaseOperationLog(command, content, family, subfamily, SemanticRoleOperationLog)
	case DatabaseSubfamilyMigrationLog:
		return decideDatabaseOperationLog(command, content, family, subfamily, SemanticRoleSideEffect)
	case DatabaseSubfamilyConnectionError:
		return decideDatabaseOperationLog(command, content, family, subfamily, SemanticRoleOperationLog)
	default:
		return keepDecision(family, SemanticRoleUnknown, string(DatabaseSubfamilyUnknown), databaseDataBearingKeepReason)
	}
}

func decideDatabaseOperationLog(command, content string, family commandFamily, subfamily DatabaseSubfamily, role SemanticRole) Decision {
	failure := classifyFailure(command, content, family)
	if failure != "" {
		if replacement, ok := buildFailureCompact(command, content, family, failure); ok {
			return inlineDecision(family, role, string(subfamily), failure, replacement, DecisionEvidence{FailureSignal: failure})
		}
		return keepDecision(family, role, string(subfamily), failure+"_not_large")
	}
	if !looksLikeSuccessfulOutput(content) {
		return keepDecision(family, role, string(subfamily), "command_output_not_success")
	}
	replacement, reason, ok := buildSafePlaceholder(command, content, "omit_successful_database_operation_command_output", "database_operation_success", "database_operation", "database operation")
	if !ok {
		return keepDecision(family, role, string(subfamily), reason)
	}
	return inlineDecision(family, role, string(subfamily), "database_operation", replacement, DecisionEvidence{SuccessSignal: "success"})
}

func decideInlineOrKeepCommand(command, content string, family commandFamily) Decision {
	failure := classifyFailure(command, content, family)
	if failure != "" {
		if replacement, ok := buildFailureCompact(command, content, family, failure); ok {
			return inlineDecision(family, semanticRoleForFamily(family), "", failure, replacement, DecisionEvidence{FailureSignal: failure})
		}
		return keepDecision(family, semanticRoleForFamily(family), "", failure+"_not_large")
	}
	if !looksLikeSuccessfulOutput(content) {
		return keepDecision(family, semanticRoleForFamily(family), "", "command_output_not_success")
	}

	switch family {
	case commandFamilyValidation:
		replacement, reason, ok := buildValidationSuccessPlaceholder(command, content)
		if !ok {
			return keepDecision(family, SemanticRoleValidationLog, "", reason)
		}
		return inlineDecision(family, SemanticRoleValidationLog, "", "validation", replacement, DecisionEvidence{SuccessSignal: "success"})
	case commandFamilyObservation:
		return dataBearingDecision(family, "", "observation", observationEvidenceKeepReason, "observation_first_last")
	case commandFamilyFileDump:
		return dataBearingDecision(family, "", "file_dump", fileDumpEvidenceKeepReason, "file_dump_first_last")
	case commandFamilyGitDiff:
		return dataBearingDecision(family, "", "git_diff", gitDiffEvidenceKeepReason, "git_diff_first_last")
	case commandFamilyGitShow:
		return dataBearingDecision(family, "", "git_show", gitShowEvidenceKeepReason, "git_show_first_last")
	case commandFamilyGitStatus, commandFamilyGitLog, commandFamilyGitBranch, commandFamilyGitFileList:
		replacement, reason, ok := buildGitCompact(command, content, family)
		return inlineFromBuilder(family, SemanticRoleOperationLog, "", replacement, reason, ok)
	default:
		return keepDecision(family, SemanticRoleUnknown, "", "command_output_unknown_skip")
	}
}

func inlineFromBuilder(family commandFamily, role SemanticRole, subfamily string, replacement Replacement, reason string, ok bool) Decision {
	if !ok {
		return keepDecision(family, role, subfamily, reason)
	}
	return inlineDecision(family, role, subfamily, replacement.Classifier(), replacement, DecisionEvidence{SuccessSignal: "success"})
}

func dataBearingDecision(family commandFamily, subfamily, classifier, keepReason, excerptPolicy string) Decision {
	return Decision{
		Action:       DecisionArtifactBackedCandidate,
		SemanticRole: SemanticRoleDataBearing,
		Family:       string(family),
		Subfamily:    subfamily,
		Classifier:   classifier,
		KeepReason:   keepReason,
		ArtifactPolicy: ArtifactPolicy{
			Eligible:         true,
			RequiredForApply: true,
			Reason:           "raw_output_artifact_required",
			ExcerptPolicy:    excerptPolicy,
		},
		Preconditions: []string{"raw_output_ref", "artifact_verify", "active_context_rehydrate", "apply_threshold"},
	}
}

func inlineDecision(family commandFamily, role SemanticRole, subfamily, classifier string, replacement Replacement, evidence DecisionEvidence) Decision {
	return Decision{
		Action:       DecisionInlineCompact,
		SemanticRole: role,
		Family:       string(family),
		Subfamily:    subfamily,
		Classifier:   classifier,
		ReplacementPlan: ReplacementPlan{
			Kind:       replacement.Kind(),
			Reason:     replacement.Reason(),
			Classifier: replacement.Classifier(),
			Text:       replacement.Text(),
		},
		Evidence:    evidence,
		replacement: replacement,
	}
}

func keepDecision(family commandFamily, role SemanticRole, subfamily, reason string) Decision {
	return Decision{
		Action:       DecisionKeepRaw,
		SemanticRole: role,
		Family:       string(family),
		Subfamily:    subfamily,
		Classifier:   reason,
		KeepReason:   reason,
	}
}

func semanticRoleForFamily(family commandFamily) SemanticRole {
	switch family {
	case commandFamilyValidation:
		return SemanticRoleValidationLog
	case commandFamilyPackage, commandFamilyDeploy:
		return SemanticRoleSideEffect
	case commandFamilySensitive:
		return SemanticRoleSensitive
	case commandFamilyNetwork, commandFamilyDatabase:
		return SemanticRoleDataBearing
	case commandFamilyObservation, commandFamilyFileDump, commandFamilyGitStatus, commandFamilyGitDiff, commandFamilyGitShow, commandFamilyGitLog, commandFamilyGitBranch, commandFamilyGitFileList:
		return SemanticRoleOperationLog
	default:
		return SemanticRoleUnknown
	}
}

func classifyDatabaseSubfamily(command, content string) DatabaseSubfamily {
	words := commandWords(command)
	head := wordBase(wordAt(words, 0))
	second := wordAt(words, 1)
	third := wordAt(words, 2)
	lowerCommand := strings.ToLower(command)
	lowerContent := strings.ToLower(content)

	if head == "prisma" && second == "migrate" || head == "npx" && second == "prisma" && third == "migrate" ||
		strings.Contains(lowerCommand, " migrate") || strings.Contains(lowerContent, "migration failed") || strings.Contains(lowerContent, "migration") && strings.Contains(lowerContent, "applied") {
		return DatabaseSubfamilyMigrationLog
	}
	if strings.Contains(lowerContent, "could not connect") ||
		strings.Contains(lowerContent, "connection refused") ||
		strings.Contains(lowerContent, "connection failed") ||
		strings.Contains(lowerContent, "password authentication failed") ||
		strings.Contains(lowerContent, "authentication failed") ||
		strings.Contains(lowerContent, "access denied for user") {
		return DatabaseSubfamilyConnectionError
	}
	if strings.Contains(lowerCommand, ".schema") ||
		strings.Contains(lowerCommand, "pragma table_info") ||
		strings.Contains(lowerCommand, "describe ") ||
		strings.Contains(lowerCommand, "\\d") ||
		strings.Contains(lowerCommand, "show columns") ||
		strings.Contains(lowerCommand, "show create table") {
		return DatabaseSubfamilySchemaResult
	}
	if strings.Contains(lowerCommand, "select ") ||
		strings.Contains(lowerCommand, "with ") ||
		strings.Contains(lowerCommand, "show tables") ||
		strings.Contains(lowerCommand, "show databases") ||
		strings.Contains(lowerCommand, "count(") {
		return DatabaseSubfamilyQueryResult
	}
	if strings.Contains(lowerCommand, "insert ") ||
		strings.Contains(lowerCommand, "update ") ||
		strings.Contains(lowerCommand, "delete ") ||
		strings.Contains(lowerCommand, "create ") ||
		strings.Contains(lowerCommand, "drop ") ||
		strings.Contains(lowerCommand, "alter ") ||
		strings.Contains(lowerCommand, "vacuum") ||
		strings.Contains(lowerCommand, "reindex") {
		return DatabaseSubfamilyOperationLog
	}
	if databaseOutputLooksLikeRows(content) {
		return DatabaseSubfamilyQueryResult
	}
	return DatabaseSubfamilyUnknown
}

func databaseOutputLooksLikeRows(content string) bool {
	lines := outputLines(content)
	if len(lines) < 2 {
		return false
	}
	rowLike := 0
	for _, line := range lines[:minInt(len(lines), 12)] {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if strings.Contains(trimmed, "|") || strings.Contains(trimmed, "\t") {
			rowLike++
		}
	}
	return rowLike >= 2
}
