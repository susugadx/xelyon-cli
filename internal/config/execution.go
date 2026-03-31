package config

// ExecutionMode はツール実行の承認モード
type ExecutionMode string

const (
	// ExecutionBalanced はデフォルトモード。
	// read は自動、write は確認、verification bash は安全なもの自動。
	ExecutionBalanced ExecutionMode = "balanced"
	// ExecutionTrusted はワークスペース内の通常編集を自動にするモード。
	// 高影響操作は確認。verification / env / git 系 bash は広めに自動。
	ExecutionTrusted ExecutionMode = "trusted"
	// ExecutionFullAuto は原則自動モード。
	// always_confirm に指定されたカテゴリのみ確認。discovery bash は解禁しない。
	ExecutionFullAuto ExecutionMode = "full_auto"
)

// ValidExecutionModes は有効な ExecutionMode の一覧
var ValidExecutionModes = []string{
	string(ExecutionBalanced),
	string(ExecutionTrusted),
	string(ExecutionFullAuto),
}

// ExecutionConfig は user-facing の実行承認設定
type ExecutionConfig struct {
	Mode              string   `yaml:"mode"`                // balanced / trusted / full_auto
	AlwaysConfirm     []string `yaml:"always_confirm"`      // どの mode でも確認するカテゴリ
	SafeShellCommands []string `yaml:"safe_shell_commands"` // verification / env 用の追加安全コマンド
}

// ConfirmCategory は always_confirm で指定可能な操作カテゴリ
type ConfirmCategory string

const (
	ConfirmDeleteFile        ConfirmCategory = "delete_file"
	ConfirmMoveFile          ConfirmCategory = "move_file"
	ConfirmDestructiveWrite  ConfirmCategory = "destructive_write"
	ConfirmWorkspaceOutside  ConfirmCategory = "workspace_outside_write"
	ConfirmBashDestructive   ConfirmCategory = "bash_destructive"
	ConfirmBashRedirectWrite ConfirmCategory = "bash_redirect_write"
	ConfirmMassChange        ConfirmCategory = "mass_change"
)

// ShellCategory は bash コマンドの用途分類
type ShellCategory int

const (
	// ShellDiscovery はコード探索系コマンド（grep, find, cat 等）。
	// 全 mode で原則自動許可しない。専用ツールを使う。
	ShellDiscovery ShellCategory = iota
	// ShellVerification は検証・ビルド・テスト・環境確認系コマンド。
	// mode に応じて自動許可範囲が変わる。
	ShellVerification
	// ShellDestructive は破壊的・高影響コマンド（rm, git push 等）。
	// always_confirm で止められる。
	ShellDestructive
)

// ExecutionPolicy は ExecutionMode から展開される内部ポリシー。
// runtime で参照される実際の承認判定ルール。
type ExecutionPolicy struct {
	Mode ExecutionMode

	// ツール承認
	AutoApproveRead bool // SafetyHigh ツール（全 mode で true）

	// trusted: workspace 内の「通常編集」のみ自動。高影響操作は確認。
	// 通常編集 = write_file / str_replace / apply_patch / create_dir / copy_file（workspace 内）
	// 高影響 = delete_file / move / workspace 外 / mass_change
	AutoApproveTrustedWrite bool

	// full_auto: SafetyLow 含む原則自動（always_confirm は除く）
	AutoApproveFullAuto bool

	// Bash
	AutoApproveVerificationBash bool // verification / env / git 系
	AllowDiscoveryBash          bool // discovery bash（常に false）

	// always_confirm（mode より優先）
	AlwaysConfirmCategories map[ConfirmCategory]bool

	// safe_shell_commands は verification 系の escape hatch であり、
	// destructive / discovery 操作の override ではない。
	SafeShellCommands []string
}

// ResolveExecutionPolicy は ExecutionConfig から内部ポリシーを生成する。
func ResolveExecutionPolicy(exec ExecutionConfig) ExecutionPolicy {
	mode := ExecutionMode(exec.Mode)
	if mode == "" {
		mode = ExecutionBalanced
	}

	policy := ExecutionPolicy{
		Mode:               mode,
		AutoApproveRead:    true,  // 全 mode で read は自動
		AllowDiscoveryBash: false, // 全 mode で discovery bash 禁止
		SafeShellCommands:  exec.SafeShellCommands,
	}

	// always_confirm をマップに展開
	policy.AlwaysConfirmCategories = make(map[ConfirmCategory]bool)
	for _, cat := range exec.AlwaysConfirm {
		policy.AlwaysConfirmCategories[ConfirmCategory(cat)] = true
	}

	switch mode {
	case ExecutionTrusted:
		policy.AutoApproveTrustedWrite = true
		policy.AutoApproveVerificationBash = true
	case ExecutionFullAuto:
		policy.AutoApproveTrustedWrite = true
		policy.AutoApproveFullAuto = true
		policy.AutoApproveVerificationBash = true
	default: // balanced
		policy.AutoApproveVerificationBash = true
	}

	return policy
}

// ShouldConfirm は指定カテゴリが always_confirm に含まれるか判定する。
// full_auto でも always_confirm は尊重される。mode より優先。
func (p ExecutionPolicy) ShouldConfirm(cat ConfirmCategory) bool {
	return p.AlwaysConfirmCategories[cat]
}

// TrustedWriteTools は trusted モードで自動承認される「通常編集」ツール。
// workspace 内のファイル作成・編集のみ。delete / move は含まない。
var TrustedWriteTools = map[string]bool{
	"write_file":  true,
	"str_replace": true,
	"apply_patch": true,
	"create_dir":  true,
	"copy_file":   true,
	"git_add":     true,
}

// ShellRedirectWrite はリダイレクト書き込みを含むコマンド。
// bash_redirect_write の always_confirm 対象。
const ShellRedirectWrite ShellCategory = 3

// ClassifyShellCommand は bash コマンドを用途分類する。
func ClassifyShellCommand(command string) ShellCategory {
	if isDiscoveryCommand(command) {
		return ShellDiscovery
	}
	if isDestructiveCommand(command) {
		return ShellDestructive
	}
	if isRedirectWriteCommand(command) {
		return ShellRedirectWrite
	}
	return ShellVerification
}

// isRedirectWriteCommand はリダイレクト書き込み（>, >>）を含むか判定する。
func isRedirectWriteCommand(command string) bool {
	// クォート外の > >> を検出
	inSingle := false
	inDouble := false
	for i := 0; i < len(command); i++ {
		ch := command[i]
		if ch == '\'' && !inDouble {
			inSingle = !inSingle
			continue
		}
		if ch == '"' && !inSingle {
			inDouble = !inDouble
			continue
		}
		if inSingle || inDouble {
			continue
		}
		if ch == '>' {
			return true
		}
	}
	return false
}

// isDiscoveryCommand はコード探索系コマンドかを判定する。
// repo 内のファイル内容・構造を調べるために使う bash コマンド。
func isDiscoveryCommand(command string) bool {
	// チェーンコマンドは各パーツを判定
	parts := splitShellParts(command)
	for _, part := range parts {
		if isDiscoveryPart(part) {
			return true
		}
	}
	return false
}

// discoveryPrefixes は探索系コマンドのプレフィックス
var discoveryPrefixes = []string{
	"grep", "rg", "ag",
	"find",
	"cat", "head", "tail",
	"sed -n", "awk",
	"ls", // ls 全般を discovery 扱い（list_dir を使う）
	"tree",
	"wc -l", "wc -c",
	"xargs grep", "xargs rg", "xargs cat",
}

func isDiscoveryPart(part string) bool {
	trimmed := trimCommandPrefix(part)
	for _, prefix := range discoveryPrefixes {
		if matchShellPrefix(trimmed, prefix) {
			return true
		}
	}
	return false
}

// isDestructiveCommand は破壊的コマンドかを判定する。
func isDestructiveCommand(command string) bool {
	parts := splitShellParts(command)
	for _, part := range parts {
		trimmed := trimCommandPrefix(part)
		for _, prefix := range destructivePrefixes {
			if matchShellPrefix(trimmed, prefix) {
				return true
			}
		}
	}
	return false
}

var destructivePrefixes = []string{
	"rm", "rmdir",
	"git push", "git force-push", "git reset --hard",
	"git branch -D", "git branch -d",
	"sudo",
	"chmod", "chown",
	"mv", // ファイル移動は破壊的
	"kill", "killall", "pkill",
}

// splitShellParts はコマンドをパイプ・チェーンで分割する。
// | && || ; で分割し、各パーツを返す。
func splitShellParts(command string) []string {
	var parts []string
	var current []byte
	inSingle := false
	inDouble := false

	for i := 0; i < len(command); i++ {
		ch := command[i]
		if ch == '\'' && !inDouble {
			inSingle = !inSingle
			current = append(current, ch)
			continue
		}
		if ch == '"' && !inSingle {
			inDouble = !inDouble
			current = append(current, ch)
			continue
		}
		if inSingle || inDouble {
			current = append(current, ch)
			continue
		}
		// セパレータ
		if ch == '|' || ch == ';' {
			if part := trimBytes(current); len(part) > 0 {
				parts = append(parts, string(part))
			}
			current = current[:0]
			// && ||
			if ch == '|' && i+1 < len(command) && command[i+1] == '|' {
				i++
			}
			continue
		}
		if ch == '&' && i+1 < len(command) && command[i+1] == '&' {
			if part := trimBytes(current); len(part) > 0 {
				parts = append(parts, string(part))
			}
			current = current[:0]
			i++
			continue
		}
		current = append(current, ch)
	}
	if part := trimBytes(current); len(part) > 0 {
		parts = append(parts, string(part))
	}
	return parts
}

func trimBytes(b []byte) []byte {
	start, end := 0, len(b)
	for start < end && (b[start] == ' ' || b[start] == '\t') {
		start++
	}
	for end > start && (b[end-1] == ' ' || b[end-1] == '\t') {
		end--
	}
	return b[start:end]
}

// trimCommandPrefix は環境変数設定プレフィックス（KEY=VAL cmd）を除去する。
func trimCommandPrefix(part string) string {
	// ENV=VAL ENV2=VAL2 cmd → cmd
	for {
		idx := findFirstSpace(part)
		if idx < 0 {
			break
		}
		prefix := part[:idx]
		if !isEnvAssignment(prefix) {
			break
		}
		part = part[idx+1:]
		for len(part) > 0 && part[0] == ' ' {
			part = part[1:]
		}
	}
	return part
}

func findFirstSpace(s string) int {
	for i := 0; i < len(s); i++ {
		if s[i] == ' ' || s[i] == '\t' {
			return i
		}
	}
	return -1
}

func isEnvAssignment(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] == '=' && i > 0 {
			return true
		}
		if !isIdentChar(s[i]) && s[i] != '=' {
			return false
		}
	}
	return false
}

func isIdentChar(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '_'
}

// matchShellPrefix はコマンドが prefix で始まるか判定する（単語境界チェック付き）。
func matchShellPrefix(command, prefix string) bool {
	if command == prefix {
		return true
	}
	if len(command) > len(prefix) && command[:len(prefix)] == prefix {
		next := command[len(prefix)]
		return next == ' ' || next == '\t' || next == '/' || next == '='
	}
	return false
}
