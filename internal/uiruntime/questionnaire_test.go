package uiruntime

import (
	"bytes"
	"strings"
	"testing"
)

func TestQuestionnaireAskWithIO_SingleChoice(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		options []string
		def     string
		want    string
	}{
		{
			name:    "empty input uses explicit default",
			input:   "\n",
			options: []string{"alpha", "beta", "gamma"},
			def:     "beta",
			want:    "beta",
		},
		{
			name:    "numeric choice selects option",
			input:   "2\n",
			options: []string{"alpha", "beta", "gamma"},
			want:    "beta",
		},
		{
			name:    "text match is case insensitive",
			input:   "GAMMA\n",
			options: []string{"alpha", "beta", "gamma"},
			want:    "gamma",
		},
		{
			name:    "invalid input falls back to first option",
			input:   "unknown\n",
			options: []string{"alpha", "beta", "gamma"},
			want:    "alpha",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out bytes.Buffer
			q := &Questionnaire{
				Question:     "Select target",
				QuestionType: "single_choice",
				Options:      tt.options,
				Default:      tt.def,
			}

			answer, err := q.AskWithIO(NewPromptIO(strings.NewReader(tt.input), &out, &out, nil))
			if err != nil {
				t.Fatalf("AskWithIO() error = %v", err)
			}
			if answer.Value != tt.want {
				t.Fatalf("AskWithIO() value = %q, want %q", answer.Value, tt.want)
			}

			gotOutput := stripANSI(out.String())
			if !strings.Contains(gotOutput, "Select target") {
				t.Fatalf("expected question in output, got %q", gotOutput)
			}
			if !strings.Contains(gotOutput, tt.want) {
				t.Fatalf("expected selected value in output, got %q", gotOutput)
			}
		})
	}
}

func TestQuestionnaireAskWithIO_MultiChoice(t *testing.T) {
	t.Run("deduplicates numeric selections", func(t *testing.T) {
		var out bytes.Buffer
		q := &Questionnaire{
			Question:     "Select files",
			QuestionType: "multi_choice",
			Options:      []string{"alpha", "beta", "gamma"},
		}

		answer, err := q.AskWithIO(NewPromptIO(strings.NewReader("2, 1, 2, 99\n"), &out, &out, nil))
		if err != nil {
			t.Fatalf("AskWithIO() error = %v", err)
		}
		if !answer.IsMultiple {
			t.Fatal("AskWithIO() IsMultiple = false, want true")
		}
		if got, want := strings.Join(answer.Values, ","), "beta,alpha"; got != want {
			t.Fatalf("AskWithIO() values = %q, want %q", got, want)
		}
		if !strings.Contains(stripANSI(out.String()), "beta, alpha") {
			t.Fatalf("expected selected values in output, got %q", stripANSI(out.String()))
		}
	})

	t.Run("empty input returns empty selection", func(t *testing.T) {
		q := &Questionnaire{
			Question:     "Select files",
			QuestionType: "multi_choice",
			Options:      []string{"alpha", "beta", "gamma"},
		}

		answer, err := q.AskWithIO(NewPromptIO(strings.NewReader("\n"), &bytes.Buffer{}, &bytes.Buffer{}, nil))
		if err != nil {
			t.Fatalf("AskWithIO() error = %v", err)
		}
		if !answer.IsMultiple {
			t.Fatal("AskWithIO() IsMultiple = false, want true")
		}
		if len(answer.Values) != 0 {
			t.Fatalf("AskWithIO() values = %+v, want empty", answer.Values)
		}
	})
}

func TestQuestionnaireAskWithIO_FreeText(t *testing.T) {
	t.Run("empty input uses default", func(t *testing.T) {
		q := &Questionnaire{
			Question:     "Describe change",
			QuestionType: "free_text",
			Default:      "default note",
		}

		answer, err := q.AskWithIO(NewPromptIO(strings.NewReader("\n"), &bytes.Buffer{}, &bytes.Buffer{}, nil))
		if err != nil {
			t.Fatalf("AskWithIO() error = %v", err)
		}
		if answer.Value != "default note" {
			t.Fatalf("AskWithIO() value = %q, want %q", answer.Value, "default note")
		}
	})

	t.Run("explicit input overrides default", func(t *testing.T) {
		q := &Questionnaire{
			Question:     "Describe change",
			QuestionType: "free_text",
			Default:      "default note",
		}

		answer, err := q.AskWithIO(NewPromptIO(strings.NewReader("updated plan\n"), &bytes.Buffer{}, &bytes.Buffer{}, nil))
		if err != nil {
			t.Fatalf("AskWithIO() error = %v", err)
		}
		if answer.Value != "updated plan" {
			t.Fatalf("AskWithIO() value = %q, want %q", answer.Value, "updated plan")
		}
	})
}

func TestQuestionnaireAskWithIO_UnknownType(t *testing.T) {
	q := &Questionnaire{
		Question:     "Unsupported",
		QuestionType: "unknown",
	}

	if _, err := q.AskWithIO(NewPromptIO(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{}, nil)); err == nil {
		t.Fatal("AskWithIO() error = nil, want unknown question type error")
	}
}
