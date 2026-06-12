package skills

// Source は skill の検出元を表す。
type Source string

const (
	SourceProject Source = "project"
	SourceHome    Source = "home"
	SourceXelyon  Source = "xelyon"
)

// DiscoverOptions は skill discover の入力オプション。
type DiscoverOptions struct {
	InvocationCWD string
	HomeDir       string
}

// DiscoveredSkill は SKILL.md の候補ディレクトリ情報。
type DiscoveredSkill struct {
	Directory   string
	SkillPath   string
	Source      Source
	RootPath    string
	RootOrder   int
	PathOrder   string
	DisplayPath string
}

// DiscoverResult は discover 結果。
type DiscoverResult struct {
	Roots       []string
	Skills      []DiscoveredSkill
	Diagnostics []Diagnostic
}

// ParsedSkill は SKILL.md 解析結果。
type ParsedSkill struct {
	Name        string
	Description string
	Body        string
	Directory   string
	SkillPath   string
	Source      Source
	Routing     *RoutingMetadata
	Scripts     []string
	References  []string
	Assets      []string
}

// SkillCatalog は重複解決済みの catalog。
type SkillCatalog struct {
	Skills      []ParsedSkill
	Diagnostics []Diagnostic
}

// ActivatedSkill は activate 結果。
type ActivatedSkill struct {
	Skill   ParsedSkill
	Payload ActivatedSkillPayload
	Content string
}

// ActivatedSkillPayload は activate_skill 返却用の構造化 payload。
type ActivatedSkillPayload struct {
	Name           string   `json:"name"`
	SkillDirectory string   `json:"skill_directory"`
	Scripts        []string `json:"scripts"`
	References     []string `json:"references"`
	Assets         []string `json:"assets"`
	SkillMD        string   `json:"skill_md"`
}

// DiagnosticSeverity は診断の重大度。
type DiagnosticSeverity string

const (
	SeverityInfo    DiagnosticSeverity = "info"
	SeverityWarning DiagnosticSeverity = "warning"
	SeverityError   DiagnosticSeverity = "error"
)

// Diagnostic は skill 読み込み時の診断情報。
type Diagnostic struct {
	Severity DiagnosticSeverity
	Code     string
	Message  string
	Path     string
}
