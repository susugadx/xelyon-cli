package prompt

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
)

// Authority は prompt section の権限階層を表す。
type Authority string

const (
	// AuthorityConstitution は XELYON の最上位の恒久 instruction を表す。
	AuthorityConstitution Authority = "constitution"
	// AuthorityRuntimeInstruction は runtime が現在の mode / tool surface に応じて追加する instruction を表す。
	AuthorityRuntimeInstruction Authority = "runtime_instruction"
	// AuthorityRepoInstruction は repo / project 由来の instruction を表す。
	AuthorityRepoInstruction Authority = "repo_instruction"
	// AuthorityData は Project Map や tool metadata のような data-only section を表す。
	AuthorityData Authority = "data"
)

var knownAuthorities = map[Authority]struct{}{
	AuthorityConstitution:       {},
	AuthorityRuntimeInstruction: {},
	AuthorityRepoInstruction:    {},
	AuthorityData:               {},
}

// PromptSection は stable ID と authority を持つ prompt 断片である。
// zero value は invalid であり、StaticText / DynamicText で生成する。
type PromptSection struct {
	id        string
	authority Authority
	content   string
	dynamic   bool
	metadata  map[string]string
}

// EffectivePrompt は PromptSection 群から構成される provider-facing prompt である。
// section order は constructor に渡した順序を保持する。
type EffectivePrompt struct {
	sections []PromptSection
}

// StaticText は cache static 側に置く prompt section を作る。
func StaticText(id string, authority Authority, content string) PromptSection {
	return PromptSection{
		id:        strings.TrimSpace(id),
		authority: authority,
		content:   strings.Trim(content, "\n"),
	}
}

// DynamicText は cache dynamic 側に置く prompt section を作る。
func DynamicText(id string, authority Authority, content string, metadata map[string]string) PromptSection {
	return PromptSection{
		id:        strings.TrimSpace(id),
		authority: authority,
		content:   strings.Trim(content, "\n"),
		dynamic:   true,
		metadata:  clonePromptSectionMetadata(metadata),
	}
}

// ID は section の stable ID を返す。
func (s PromptSection) ID() string {
	return s.id
}

// Authority は section の authority を返す。
func (s PromptSection) Authority() Authority {
	return s.authority
}

// Content は section の provider-facing text を返す。
func (s PromptSection) Content() string {
	return s.content
}

// Dynamic は section が cache dynamic 側に属するかを返す。
func (s PromptSection) Dynamic() bool {
	return s.dynamic
}

// Metadata は section fingerprint 用 metadata の copy を返す。
func (s PromptSection) Metadata() map[string]string {
	return clonePromptSectionMetadata(s.metadata)
}

// Validate は section の不変条件を検証する。
func (s PromptSection) Validate() error {
	if strings.TrimSpace(s.id) == "" {
		return errors.New("prompt section id must be non-empty")
	}
	if _, ok := knownAuthorities[s.authority]; !ok {
		return fmt.Errorf("prompt section %q has unknown authority %q", s.id, s.authority)
	}
	if strings.TrimSpace(s.content) == "" {
		return fmt.Errorf("prompt section %q content must be non-empty", s.id)
	}
	for key, value := range s.metadata {
		if strings.TrimSpace(key) == "" {
			return fmt.Errorf("prompt section %q metadata key must be non-empty", s.id)
		}
		if key != strings.TrimSpace(key) {
			return fmt.Errorf("prompt section %q metadata key must be canonical: %q", s.id, key)
		}
		if value != strings.TrimSpace(value) {
			return fmt.Errorf("prompt section %q metadata value for %q must be canonical", s.id, key)
		}
	}
	return nil
}

// NewEffectivePrompt は section 群を検証して EffectivePrompt を作る。
func NewEffectivePrompt(sections ...PromptSection) (EffectivePrompt, error) {
	if len(sections) == 0 {
		return EffectivePrompt{}, errors.New("effective prompt requires at least one section")
	}
	ids := make(map[string]struct{}, len(sections))
	out := make([]PromptSection, 0, len(sections))
	for i, section := range sections {
		if err := section.Validate(); err != nil {
			return EffectivePrompt{}, fmt.Errorf("sections[%d]: %w", i, err)
		}
		if _, exists := ids[section.id]; exists {
			return EffectivePrompt{}, fmt.Errorf("prompt section id %q is duplicated", section.id)
		}
		ids[section.id] = struct{}{}
		out = append(out, section.clone())
	}
	return EffectivePrompt{sections: out}, nil
}

// Sections は section 群の copy を返す。
func (p EffectivePrompt) Sections() []PromptSection {
	out := make([]PromptSection, len(p.sections))
	for i, section := range p.sections {
		out[i] = section.clone()
	}
	return out
}

// Compose は static/dynamic section を cacheBoundary で分けて provider-facing text にする。
func (p EffectivePrompt) Compose(cacheBoundary string) string {
	var staticParts []string
	var dynamicParts []string
	for _, section := range p.sections {
		content := strings.Trim(section.content, "\n")
		if strings.TrimSpace(content) == "" {
			continue
		}
		if section.dynamic {
			dynamicParts = append(dynamicParts, content)
		} else {
			staticParts = append(staticParts, content)
		}
	}

	static := strings.Join(staticParts, "\n\n")
	dynamic := strings.Join(dynamicParts, "\n\n")
	switch {
	case static == "":
		return dynamic
	case dynamic == "":
		return static
	default:
		return static + cacheBoundary + dynamic
	}
}

// Fingerprint は section ID / authority / dynamic metadata / content を含む stable fingerprint を返す。
func (p EffectivePrompt) Fingerprint() (string, error) {
	if len(p.sections) == 0 {
		return "", errors.New("effective prompt has no sections")
	}
	records := make([]promptSectionFingerprintRecord, 0, len(p.sections))
	for i, section := range p.sections {
		if err := section.Validate(); err != nil {
			return "", fmt.Errorf("sections[%d]: %w", i, err)
		}
		records = append(records, promptSectionFingerprintRecord{
			ID:        section.id,
			Authority: string(section.authority),
			Dynamic:   section.dynamic,
			Content:   section.content,
			Metadata:  sortedPromptSectionMetadata(section.metadata),
		})
	}
	data, err := json.Marshal(records)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

type promptSectionFingerprintRecord struct {
	ID        string                         `json:"id"`
	Authority string                         `json:"authority"`
	Dynamic   bool                           `json:"dynamic"`
	Content   string                         `json:"content"`
	Metadata  []promptSectionMetadataKVEntry `json:"metadata,omitempty"`
}

type promptSectionMetadataKVEntry struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

func (s PromptSection) clone() PromptSection {
	s.metadata = clonePromptSectionMetadata(s.metadata)
	return s
}

func clonePromptSectionMetadata(metadata map[string]string) map[string]string {
	if len(metadata) == 0 {
		return nil
	}
	out := make(map[string]string, len(metadata))
	for key, value := range metadata {
		out[key] = value
	}
	return out
}

func sortedPromptSectionMetadata(metadata map[string]string) []promptSectionMetadataKVEntry {
	if len(metadata) == 0 {
		return nil
	}
	keys := make([]string, 0, len(metadata))
	for key := range metadata {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]promptSectionMetadataKVEntry, 0, len(keys))
	for _, key := range keys {
		out = append(out, promptSectionMetadataKVEntry{Key: key, Value: metadata[key]})
	}
	return out
}
