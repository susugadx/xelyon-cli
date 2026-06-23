package plancontract

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
)

// SchemaVersion は Plan Mode provider output の schema version である。
const SchemaVersion = "xelyon.plan.v2"

// Document は xelyon.plan.v2 の provider-facing DTO である。
type Document struct {
	SchemaVersion      string    `json:"schema_version"`
	Goal               string    `json:"goal"`
	AcceptanceCriteria []string  `json:"acceptance_criteria"`
	Findings           []Finding `json:"findings"`
	Constraints        []string  `json:"constraints"`
	Steps              []Step    `json:"steps"`
	OpenQuestions      []string  `json:"open_questions"`
}

// Finding は plan の調査事実と根拠を表す。
type Finding struct {
	Fact     string   `json:"fact"`
	Evidence []string `json:"evidence"`
}

// Step は実装 mode へ渡す計画ステップを表す。
type Step struct {
	ID           string   `json:"id"`
	Outcome      string   `json:"outcome"`
	Files        []string `json:"files"`
	Reason       string   `json:"reason"`
	Verification []string `json:"verification"`
}

type documentJSON struct {
	SchemaVersion      *string        `json:"schema_version"`
	Goal               *string        `json:"goal"`
	AcceptanceCriteria *[]string      `json:"acceptance_criteria"`
	Findings           *[]findingJSON `json:"findings"`
	Constraints        *[]string      `json:"constraints"`
	Steps              *[]stepJSON    `json:"steps"`
	OpenQuestions      *[]string      `json:"open_questions"`
}

type findingJSON struct {
	Fact     *string   `json:"fact"`
	Evidence *[]string `json:"evidence"`
}

type stepJSON struct {
	ID           *string   `json:"id"`
	Outcome      *string   `json:"outcome"`
	Files        *[]string `json:"files"`
	Reason       *string   `json:"reason"`
	Verification *[]string `json:"verification"`
}

// DecodeStrict は xelyon.plan.v2 JSON を unknown field 禁止で decode して検証する。
func DecodeStrict(data []byte) (Document, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var raw documentJSON
	if err := decoder.Decode(&raw); err != nil {
		return Document{}, fmt.Errorf("decode %s: %w", SchemaVersion, err)
	}
	var trailing json.RawMessage
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err != nil {
			return Document{}, fmt.Errorf("%s must contain a single JSON value: %w", SchemaVersion, err)
		}
		return Document{}, fmt.Errorf("%s must contain a single JSON value", SchemaVersion)
	}
	doc, err := documentFromJSON(raw)
	if err != nil {
		return Document{}, err
	}
	if err := Validate(doc); err != nil {
		return Document{}, err
	}
	return doc, nil
}

func documentFromJSON(raw documentJSON) (Document, error) {
	missing := []string{}
	if raw.SchemaVersion == nil {
		missing = append(missing, "schema_version")
	}
	if raw.Goal == nil {
		missing = append(missing, "goal")
	}
	if raw.AcceptanceCriteria == nil {
		missing = append(missing, "acceptance_criteria")
	}
	if raw.Findings == nil {
		missing = append(missing, "findings")
	}
	if raw.Constraints == nil {
		missing = append(missing, "constraints")
	}
	if raw.Steps == nil {
		missing = append(missing, "steps")
	}
	if raw.OpenQuestions == nil {
		missing = append(missing, "open_questions")
	}
	if len(missing) > 0 {
		return Document{}, fmt.Errorf("%s JSON missing keys: %s", SchemaVersion, strings.Join(missing, ", "))
	}
	if err := validateNestedRequiredKeys(*raw.Findings, *raw.Steps); err != nil {
		return Document{}, err
	}
	return Document{
		SchemaVersion:      *raw.SchemaVersion,
		Goal:               *raw.Goal,
		AcceptanceCriteria: append([]string(nil), (*raw.AcceptanceCriteria)...),
		Findings:           findingsFromJSON(*raw.Findings),
		Constraints:        append([]string(nil), (*raw.Constraints)...),
		Steps:              stepsFromJSON(*raw.Steps),
		OpenQuestions:      append([]string(nil), (*raw.OpenQuestions)...),
	}, nil
}

func validateNestedRequiredKeys(findings []findingJSON, steps []stepJSON) error {
	for i, finding := range findings {
		missing := []string{}
		if finding.Fact == nil {
			missing = append(missing, "fact")
		}
		if finding.Evidence == nil {
			missing = append(missing, "evidence")
		}
		if len(missing) > 0 {
			return fmt.Errorf("%s JSON findings[%d] missing keys: %s", SchemaVersion, i, strings.Join(missing, ", "))
		}
	}
	for i, step := range steps {
		missing := []string{}
		if step.ID == nil {
			missing = append(missing, "id")
		}
		if step.Outcome == nil {
			missing = append(missing, "outcome")
		}
		if step.Files == nil {
			missing = append(missing, "files")
		}
		if step.Reason == nil {
			missing = append(missing, "reason")
		}
		if step.Verification == nil {
			missing = append(missing, "verification")
		}
		if len(missing) > 0 {
			return fmt.Errorf("%s JSON steps[%d] missing keys: %s", SchemaVersion, i, strings.Join(missing, ", "))
		}
	}
	return nil
}

func findingsFromJSON(raw []findingJSON) []Finding {
	if len(raw) == 0 {
		return nil
	}
	out := make([]Finding, 0, len(raw))
	for _, finding := range raw {
		out = append(out, Finding{
			Fact:     stringValue(finding.Fact),
			Evidence: stringSliceValue(finding.Evidence),
		})
	}
	return out
}

func stepsFromJSON(raw []stepJSON) []Step {
	if len(raw) == 0 {
		return nil
	}
	out := make([]Step, 0, len(raw))
	for _, step := range raw {
		out = append(out, Step{
			ID:           stringValue(step.ID),
			Outcome:      stringValue(step.Outcome),
			Files:        stringSliceValue(step.Files),
			Reason:       stringValue(step.Reason),
			Verification: stringSliceValue(step.Verification),
		})
	}
	return out
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func stringSliceValue(values *[]string) []string {
	if values == nil {
		return nil
	}
	return append([]string(nil), (*values)...)
}

// Validate は xelyon.plan.v2 DTO の required fields と basic shape を検証する。
func Validate(doc Document) error {
	if doc.SchemaVersion != SchemaVersion {
		return fmt.Errorf("schema_version must be %q: got %q", SchemaVersion, doc.SchemaVersion)
	}
	if strings.TrimSpace(doc.Goal) == "" {
		return errors.New("goal must be non-empty")
	}
	for i, finding := range doc.Findings {
		if strings.TrimSpace(finding.Fact) == "" {
			return fmt.Errorf("findings[%d].fact must be non-empty", i)
		}
		if len(trimStringList(finding.Evidence)) == 0 {
			return fmt.Errorf("findings[%d].evidence must contain at least one item", i)
		}
	}
	seenStepIDs := make(map[string]struct{}, len(doc.Steps))
	for i, step := range doc.Steps {
		id := strings.TrimSpace(step.ID)
		if id == "" {
			return fmt.Errorf("steps[%d].id must be non-empty", i)
		}
		if id != step.ID {
			return fmt.Errorf("steps[%d].id must be canonical without leading/trailing whitespace", i)
		}
		if _, exists := seenStepIDs[id]; exists {
			return fmt.Errorf("steps[%d].id duplicates %q", i, id)
		}
		seenStepIDs[id] = struct{}{}
		if strings.TrimSpace(step.Outcome) == "" {
			return fmt.Errorf("steps[%d].outcome must be non-empty", i)
		}
	}
	return nil
}

// SchemaInstructions は Plan Mode prompt に埋め込む xelyon.plan.v2 contract 文面を返す。
func SchemaInstructions() string {
	return planV2SchemaInstructions
}

const planV2SchemaInstructions = `### Plan JSON Schema
Return exactly one JSON object for schema xelyon.plan.v2. Do not wrap it in a "plan" object.
Use this exact shape:
` + "```json" + `
{
  "schema_version": "xelyon.plan.v2",
  "goal": "User-facing implementation goal with scope",
  "acceptance_criteria": [
    "Concrete condition that proves the goal is complete"
  ],
  "findings": [
    {
      "fact": "Important investigation fact that implementation mode should know",
      "evidence": [
        "internal/agent/plan/handoff.go: ImplementationHandoff.NormalModeInput builds the implementation handoff"
      ]
    }
  ],
  "constraints": [
    "Do not carry raw investigation history into implementation mode"
  ],
  "steps": [
    {
      "id": "step-1",
      "outcome": "Concrete implementation outcome",
      "files": [
        "internal/agent/plan/handoff.go",
        "internal/agent/plan/handoff_test.go"
      ],
      "reason": "Why this step is needed or what risk/contract it closes",
      "verification": ["go test ./internal/agent/plan"]
    }
  ],
  "open_questions": []
}
` + "```" + `

Field rules:
- schema_version: must be "xelyon.plan.v2".
- goal: one concise, reviewable sentence describing the goal, scope, and important constraint when known.
- acceptance_criteria: concrete completion checks, compatibility requirements, or observable outcomes.
- findings[].fact: concise stable facts discovered from the codebase; omit guesses.
- findings[].evidence: files, functions, tests, commands, or concrete observations supporting the fact. Preserve file/function/test names when known.
- constraints: boundaries, compatibility requirements, existing design constraints, and changes to avoid.
- steps: use an empty array when no implementation is needed; otherwise keep the plan short and ordered (normally 2-6 steps). Each step must be understandable without the investigation transcript.
- steps[].id: stable string ID. Use values like "step-1"; the runtime normalizes IDs for execution.
- steps[].outcome: an implementation outcome, not an investigation action. Mention tests/docs/config in the step when that is the point of the work.
- steps[].files: implementation-relevant repo-relative files to confirm first in implementation mode. Include source, tests, docs, and config files when known.
- steps[].reason: a short review-facing reason. Say what user-visible behavior, contract, risk, or cleanup it addresses.
- steps[].verification: focused commands or checks that prove the step, or the whole plan, worked.
- open_questions: unresolved questions only. Use an empty array when no user input is required.`

func trimStringList(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			out = append(out, value)
		}
	}
	return out
}
