package refactor

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/review"
)

// AnalyzeWithAI uses LLM to find refactoring opportunities.
func AnalyzeWithAI(code string, filePath string, llm review.LLMClient) ([]RefactorProposal, error) {
	if llm == nil {
		return nil, fmt.Errorf("LLM client not configured")
	}

	prompt := buildRefactorAnalysisPrompt(code, filePath)
	response, err := llm.Chat(prompt)
	if err != nil {
		return nil, fmt.Errorf("LLM call failed: %w", err)
	}

	return parseRefactorResponse(response, filePath)
}

// GenerateSplitFilePlan generates a plan to split a large file.
func GenerateSplitFilePlan(code string, filePath string, llm review.LLMClient) (*review.MultiFileChange, error) {
	if llm == nil {
		return nil, fmt.Errorf("LLM client not configured")
	}

	lines := strings.Count(code, "\n") + 1
	prompt := fmt.Sprintf(`The following file is too large (%d lines).
Suggest how to split it into smaller, cohesive modules.

File: %s

%s%s%s

Generate a split plan in JSON format:
{
  "files": [
    {
      "name": "new_filename.go",
      "description": "Description of what this file contains",
      "functions": ["func1", "func2"],
      "lines_start": 1,
      "lines_end": 100
    }
  ]
}

Rules:
- Each new file should have a single responsibility
- Keep related functions together
- Consider package dependencies
- Suggest meaningful file names`, lines, filePath, "```\n", code, "\n```")

	response, err := llm.Chat(prompt)
	if err != nil {
		return nil, fmt.Errorf("LLM call failed: %w", err)
	}

	return parseSplitPlan(response, filePath, code)
}

// GenerateExtractMethodCode generates code for extracting a method.
func GenerateExtractMethodCode(funcCode string, funcName string, filePath string, llm review.LLMClient) (*review.MultiFileChange, error) {
	if llm == nil {
		return nil, fmt.Errorf("LLM client not configured")
	}

	prompt := fmt.Sprintf(`The following function is too long. Extract logical sub-functions to improve readability.

Function: %s
File: %s

%s%s%s

Generate the refactored code in JSON format:
{
  "old_code": "exact code to replace",
  "new_code": "refactored code with extracted functions"
}

Rules:
- Extract cohesive blocks into separate helper functions
- Use descriptive function names
- Keep the original function signature
- Place helper functions near the original
- Preserve all functionality`, funcName, filePath, "```\n", funcCode, "\n```")

	response, err := llm.Chat(prompt)
	if err != nil {
		return nil, fmt.Errorf("LLM call failed: %w", err)
	}

	return parseExtractMethodResponse(response, filePath)
}

// buildRefactorAnalysisPrompt creates the prompt for refactoring analysis.
func buildRefactorAnalysisPrompt(code string, filePath string) string {
	lang := detectLanguage(filePath)
	return fmt.Sprintf(`Analyze the following code for refactoring opportunities.

File: %s
Language: %s

%s%s%s

Look for:
1. Large files that should be split (>300 lines)
2. Long functions that should be broken down (>50 lines)
3. Duplicate code that could be extracted
4. Poor naming that reduces readability
5. Complex conditionals that could be simplified
6. God objects/classes with too many responsibilities

Output JSON only (no markdown):
{
  "proposals": [
    {
      "type": "split-file|extract-method|dry|rename",
      "description": "Description of the issue",
      "line_start": 10,
      "line_end": 50,
      "function_name": "optional function name",
      "confidence": 0.8
    }
  ]
}

If no issues found, return: {"proposals": []}`, filePath, lang, "```"+lang+"\n", code, "\n```")
}

// detectLanguage detects programming language from file extension.
func detectLanguage(filePath string) string {
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".go":
		return "go"
	case ".py":
		return "python"
	case ".js":
		return "javascript"
	case ".ts":
		return "typescript"
	case ".java":
		return "java"
	case ".rs":
		return "rust"
	case ".rb":
		return "ruby"
	case ".php":
		return "php"
	case ".c", ".h":
		return "c"
	case ".cpp", ".hpp":
		return "cpp"
	default:
		return ""
	}
}

// AIRefactorProposal represents the AI response structure.
type AIRefactorProposal struct {
	Type         string  `json:"type"`
	Description  string  `json:"description"`
	LineStart    int     `json:"line_start"`
	LineEnd      int     `json:"line_end"`
	FunctionName string  `json:"function_name"`
	Confidence   float64 `json:"confidence"`
}

// AIRefactorResponse represents the full AI response.
type AIRefactorResponse struct {
	Proposals []AIRefactorProposal `json:"proposals"`
}

// parseRefactorResponse parses the AI response into RefactorProposals.
func parseRefactorResponse(response string, filePath string) ([]RefactorProposal, error) {
	jsonStr := extractJSON(response)
	if jsonStr == "" {
		return nil, fmt.Errorf("no JSON found in response")
	}

	var aiResp AIRefactorResponse
	if err := json.Unmarshal([]byte(jsonStr), &aiResp); err != nil {
		return nil, fmt.Errorf("JSON parse error: %w", err)
	}

	var proposals []RefactorProposal
	for i, p := range aiResp.Proposals {
		refType := RefactorType(p.Type)
		switch refType {
		case RefactorSplitFile, RefactorExtractMethod, RefactorDRY, RefactorRename:
			// Valid type
		default:
			continue
		}

		proposals = append(proposals, RefactorProposal{
			ID:           fmt.Sprintf("ai-%s:%s:%d", p.Type, filePath, i),
			Type:         refType,
			Description:  p.Description,
			FilePath:     filePath,
			LineStart:    p.LineStart,
			LineEnd:      p.LineEnd,
			FunctionName: p.FunctionName,
			Confidence:   p.Confidence,
			Actionable:   false, // AI proposals need further processing to be actionable
		})
	}

	return proposals, nil
}

// SplitPlanFile represents a file in the split plan.
type SplitPlanFile struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Functions   []string `json:"functions"`
	LinesStart  int      `json:"lines_start"`
	LinesEnd    int      `json:"lines_end"`
}

// SplitPlan represents the AI-generated split plan.
type SplitPlan struct {
	Files []SplitPlanFile `json:"files"`
}

// parseSplitPlan parses the split plan response.
func parseSplitPlan(response string, filePath string, originalCode string) (*review.MultiFileChange, error) {
	jsonStr := extractJSON(response)
	if jsonStr == "" {
		return nil, fmt.Errorf("no JSON found in response")
	}

	var plan SplitPlan
	if err := json.Unmarshal([]byte(jsonStr), &plan); err != nil {
		return nil, fmt.Errorf("JSON parse error: %w", err)
	}

	if len(plan.Files) == 0 {
		return nil, fmt.Errorf("no split plan generated")
	}

	// Build MultiFileChange
	lines := strings.Split(originalCode, "\n")
	change := &review.MultiFileChange{
		ID:          fmt.Sprintf("split:%s", filepath.Base(filePath)),
		Description: fmt.Sprintf("Split %s into %d files", filepath.Base(filePath), len(plan.Files)),
	}

	dir := filepath.Dir(filePath)
	for _, f := range plan.Files {
		if f.LinesStart <= 0 || f.LinesEnd <= 0 || f.LinesStart > f.LinesEnd {
			continue
		}
		if f.LinesEnd > len(lines) {
			f.LinesEnd = len(lines)
		}

		newContent := strings.Join(lines[f.LinesStart-1:f.LinesEnd], "\n")
		newPath := filepath.Join(dir, f.Name)

		change.Changes = append(change.Changes, review.FileChange{
			FilePath: newPath,
			Patches: []review.Patch{
				{NewCode: newContent},
			},
		})
	}

	return change, nil
}

// ExtractMethodResult represents the AI extract method response.
type ExtractMethodResult struct {
	OldCode string `json:"old_code"`
	NewCode string `json:"new_code"`
}

// parseExtractMethodResponse parses the extract method response.
func parseExtractMethodResponse(response string, filePath string) (*review.MultiFileChange, error) {
	jsonStr := extractJSON(response)
	if jsonStr == "" {
		return nil, fmt.Errorf("no JSON found in response")
	}

	var result ExtractMethodResult
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, fmt.Errorf("JSON parse error: %w", err)
	}

	if result.OldCode == "" || result.NewCode == "" {
		return nil, fmt.Errorf("invalid extract method result")
	}

	return &review.MultiFileChange{
		ID:          fmt.Sprintf("extract:%s", filepath.Base(filePath)),
		Description: "Extract method refactoring",
		Changes: []review.FileChange{
			{
				FilePath: filePath,
				Patches: []review.Patch{
					{
						OldCode: result.OldCode,
						NewCode: result.NewCode,
					},
				},
			},
		},
	}, nil
}

// extractJSON extracts JSON from a response string.
func extractJSON(response string) string {
	start := strings.Index(response, "{")
	if start == -1 {
		return ""
	}

	depth := 0
	for i := start; i < len(response); i++ {
		switch response[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return response[start : i+1]
			}
		}
	}

	return ""
}
