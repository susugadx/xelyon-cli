package usageledger

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	skillcatalog "github.com/susugadx/xelyon-cli/internal/skills"
)

const (
	defaultStateDir            = ".xelyon"
	usageLedgerRelativeDir     = "skills/router/usage"
	noRepoLedgerFile           = "no-repo.jsonl"
	defaultUsageRetentionDays  = 30
	defaultRepoKeyHashByteSize = 8
)

// Options は usage ledger store の初期化設定。
type Options struct {
	StateHome     string
	ProjectRoot   string
	Enabled       bool
	RetentionDays int
	Now           func() time.Time
}

// Store は repo-scoped skill routing usage JSONL を所有する。
type Store struct {
	stateHome     string
	projectRoot   string
	repoKey       string
	enabled       bool
	retentionDays int
	now           func() time.Time
}

// SkillSummary は routing recommendation の保存用 summary。
type SkillSummary struct {
	Name       string `json:"name"`
	Category   string `json:"category"`
	Score      int    `json:"score"`
	Confidence string `json:"confidence"`
	Activation string `json:"activation"`
}

// PolicySnapshot は ledger record に保存する router policy summary。
type PolicySnapshot struct {
	Enabled         bool   `json:"enabled"`
	Activation      string `json:"activation"`
	PrimaryLimit    int    `json:"primary_limit"`
	SupportingLimit int    `json:"supporting_limit"`
	ConflictLimit   int    `json:"conflict_limit"`
	MaybeLimit      int    `json:"maybe_limit"`
}

// Record は JSONL に保存する routing outcome。
type Record struct {
	Timestamp   time.Time      `json:"timestamp"`
	RepoKey     string         `json:"repo_key"`
	Type        string         `json:"type"`
	Recommended []SkillSummary `json:"recommended,omitempty"`
	Activated   []string       `json:"activated,omitempty"`
	Policy      PolicySnapshot `json:"policy,omitempty"`
}

// SkillUsage は skill 単位の recommendation / activation 集計。
type SkillUsage struct {
	Name              string
	RecommendedCount  int
	ActivatedCount    int
	HighestScore      int
	LastRecommendedAt time.Time
	LastActivatedAt   time.Time
}

// Summary は repo ledger の集計結果。
type Summary struct {
	RepoKey string
	Path    string
	Records int
	Skills  []SkillUsage
}

// NewStore は isolated usage ledger store を初期化する。
func NewStore(opts Options) *Store {
	stateHome := strings.TrimSpace(opts.StateHome)
	if stateHome == "" {
		stateHome = defaultStateHome()
	}
	retentionDays := opts.RetentionDays
	if retentionDays <= 0 {
		retentionDays = defaultUsageRetentionDays
	}
	return &Store{
		stateHome:     filepath.Clean(stateHome),
		projectRoot:   cleanProjectRoot(opts.ProjectRoot),
		repoKey:       RepoKey(opts.ProjectRoot),
		enabled:       opts.Enabled,
		retentionDays: retentionDays,
		now:           resolveNow(opts.Now),
	}
}

// RepoKey は normalized project root から ledger file name 用 hash を作る。
func RepoKey(projectRoot string) string {
	cleaned := cleanProjectRoot(projectRoot)
	if cleaned == "" {
		return strings.TrimSuffix(noRepoLedgerFile, ".jsonl")
	}
	sum := sha256.Sum256([]byte(cleaned))
	return hex.EncodeToString(sum[:defaultRepoKeyHashByteSize])
}

// Append は usage record を JSONL に追記する。disabled store では何もしない。
func (s *Store) Append(record Record) error {
	if s == nil || !s.enabled {
		return nil
	}
	record.Timestamp = record.Timestamp.UTC()
	if record.Timestamp.IsZero() {
		record.Timestamp = s.now().UTC()
	}
	record.RepoKey = s.repoKey
	if strings.TrimSpace(record.Type) == "" {
		return errors.New("usage ledger record type is required")
	}
	if err := s.prune(); err != nil {
		return err
	}
	path := s.path()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create usage ledger dir: %w", err)
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return fmt.Errorf("open usage ledger: %w", err)
	}
	defer f.Close()
	encoded, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("encode usage ledger record: %w", err)
	}
	if _, err := f.Write(append(encoded, '\n')); err != nil {
		return fmt.Errorf("write usage ledger: %w", err)
	}
	return nil
}

// Summary は current repo ledger を retention prune 後に集計する。
func (s *Store) Summary() (Summary, error) {
	if s == nil {
		return Summary{}, nil
	}
	if err := s.prune(); err != nil {
		return Summary{}, err
	}
	records, err := s.readRecords()
	if err != nil {
		return Summary{}, err
	}
	summary := Summary{
		RepoKey: s.repoKey,
		Path:    s.path(),
		Records: len(records),
	}
	usageByName := map[string]*SkillUsage{}
	for _, record := range records {
		for _, recommended := range record.Recommended {
			if !countsAsActivationRecommendation(recommended) {
				continue
			}
			name := strings.TrimSpace(recommended.Name)
			if name == "" {
				continue
			}
			usage := ensureSkillUsage(usageByName, name)
			usage.RecommendedCount++
			if recommended.Score > usage.HighestScore {
				usage.HighestScore = recommended.Score
			}
			if record.Timestamp.After(usage.LastRecommendedAt) {
				usage.LastRecommendedAt = record.Timestamp
			}
		}
		for _, activated := range record.Activated {
			name := strings.TrimSpace(activated)
			if name == "" {
				continue
			}
			usage := ensureSkillUsage(usageByName, name)
			usage.ActivatedCount++
			if record.Timestamp.After(usage.LastActivatedAt) {
				usage.LastActivatedAt = record.Timestamp
			}
		}
	}
	for _, usage := range usageByName {
		summary.Skills = append(summary.Skills, *usage)
	}
	sort.Slice(summary.Skills, func(i, j int) bool {
		left := summary.Skills[i]
		right := summary.Skills[j]
		if left.RecommendedCount != right.RecommendedCount {
			return left.RecommendedCount > right.RecommendedCount
		}
		if left.ActivatedCount != right.ActivatedCount {
			return left.ActivatedCount > right.ActivatedCount
		}
		return left.Name < right.Name
	})
	return summary, nil
}

// Clear は current repo ledger を削除する。
func (s *Store) Clear() error {
	if s == nil {
		return nil
	}
	err := os.Remove(s.path())
	if err == nil || os.IsNotExist(err) {
		return nil
	}
	return fmt.Errorf("clear usage ledger: %w", err)
}

// ClearAll は skill router usage ledger をすべて削除する。
func (s *Store) ClearAll() error {
	if s == nil {
		return nil
	}
	dir := s.dir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read usage ledger dir: %w", err)
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".jsonl") {
			continue
		}
		if err := os.Remove(filepath.Join(dir, entry.Name())); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("clear usage ledger %s: %w", entry.Name(), err)
		}
	}
	return nil
}

// Diagnostics は usage summary から routing quality diagnostics を作る。
func Diagnostics(summary Summary) []skillcatalog.Diagnostic {
	var diagnostics []skillcatalog.Diagnostic
	for _, usage := range summary.Skills {
		if usage.RecommendedCount >= 3 && usage.ActivatedCount == 0 {
			diagnostics = append(diagnostics, skillcatalog.Diagnostic{
				Severity: skillcatalog.SeverityInfo,
				Code:     "usage_recommended_never_activated",
				Message:  fmt.Sprintf("%s was recommended %d times but never activated", sanitizedUsageSkillName(usage.Name), usage.RecommendedCount),
			})
		}
		if usage.ActivatedCount > usage.RecommendedCount {
			diagnostics = append(diagnostics, skillcatalog.Diagnostic{
				Severity: skillcatalog.SeverityInfo,
				Code:     "usage_activated_without_recommendation",
				Message:  fmt.Sprintf("%s was activated %d times but recommended %d times", sanitizedUsageSkillName(usage.Name), usage.ActivatedCount, usage.RecommendedCount),
			})
		}
	}
	return diagnostics
}

// FormatSummary は /skills usage 向けの human-readable summary を返す。
func FormatSummary(summary Summary) string {
	var b strings.Builder
	b.WriteString("Skill Routing Usage\n\n")
	fmt.Fprintf(&b, "Repo: %s\n", summary.RepoKey)
	fmt.Fprintf(&b, "Records: %d\n", summary.Records)
	if len(summary.Skills) == 0 {
		b.WriteString("\nNo usage records.\n")
		return b.String()
	}
	b.WriteString("\nSkills:\n")
	for _, usage := range summary.Skills {
		fmt.Fprintf(&b, "- %s: recommended %d, activated %d", sanitizedUsageSkillName(usage.Name), usage.RecommendedCount, usage.ActivatedCount)
		if usage.HighestScore > 0 {
			fmt.Fprintf(&b, ", highest score %d", usage.HighestScore)
		}
		b.WriteString("\n")
	}
	return b.String()
}

func sanitizedUsageSkillName(name string) string {
	name = skillcatalog.SanitizePromptLineValue(name)
	if name == "" {
		return "(invalid-skill-name)"
	}
	return name
}

func countsAsActivationRecommendation(summary SkillSummary) bool {
	switch strings.TrimSpace(summary.Category) {
	case "", "primary", "supporting":
		return true
	default:
		return false
	}
}

func (s *Store) prune() error {
	if s == nil {
		return nil
	}
	records, err := s.readRecords()
	if err != nil {
		return err
	}
	if len(records) == 0 {
		return nil
	}
	cutoff := s.now().UTC().AddDate(0, 0, -s.retentionDays)
	kept := make([]Record, 0, len(records))
	for _, record := range records {
		if record.Timestamp.IsZero() || !record.Timestamp.Before(cutoff) {
			kept = append(kept, record)
		}
	}
	if len(kept) == len(records) {
		return nil
	}
	return s.writeRecords(kept)
}

func (s *Store) readRecords() ([]Record, error) {
	path := s.path()
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("open usage ledger: %w", err)
	}
	defer f.Close()

	var records []Record
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var record Record
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			continue
		}
		records = append(records, record)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read usage ledger: %w", err)
	}
	return records, nil
}

func (s *Store) writeRecords(records []Record) error {
	path := s.path()
	if len(records) == 0 {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("remove empty usage ledger: %w", err)
		}
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create usage ledger dir: %w", err)
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("rewrite usage ledger: %w", err)
	}
	defer f.Close()
	encoder := json.NewEncoder(f)
	for _, record := range records {
		if err := encoder.Encode(record); err != nil {
			return fmt.Errorf("encode usage ledger record: %w", err)
		}
	}
	return nil
}

func (s *Store) path() string {
	if s == nil {
		return ""
	}
	if s.repoKey == strings.TrimSuffix(noRepoLedgerFile, ".jsonl") {
		return filepath.Join(s.dir(), noRepoLedgerFile)
	}
	return filepath.Join(s.dir(), s.repoKey+".jsonl")
}

func (s *Store) dir() string {
	if s == nil {
		return ""
	}
	return filepath.Join(s.stateHome, filepath.FromSlash(usageLedgerRelativeDir))
}

func ensureSkillUsage(values map[string]*SkillUsage, name string) *SkillUsage {
	usage, ok := values[name]
	if ok {
		return usage
	}
	usage = &SkillUsage{Name: name}
	values[name] = usage
	return usage
}

func cleanProjectRoot(projectRoot string) string {
	projectRoot = strings.TrimSpace(projectRoot)
	if projectRoot == "" {
		return ""
	}
	if abs, err := filepath.Abs(projectRoot); err == nil {
		return filepath.Clean(abs)
	}
	return filepath.Clean(projectRoot)
}

func defaultStateHome() string {
	home, err := os.UserHomeDir()
	if err != nil || strings.TrimSpace(home) == "" {
		return filepath.Join(".", defaultStateDir)
	}
	return filepath.Join(home, defaultStateDir)
}

func resolveNow(now func() time.Time) func() time.Time {
	if now != nil {
		return now
	}
	return time.Now
}
