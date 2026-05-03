package skills

import (
	"reflect"
	"testing"
)

func TestParseSkillScriptArgsJSON(t *testing.T) {
	t.Run("invalid syntax", func(t *testing.T) {
		_, err := parseSkillScriptArgsJSON("{bad")
		if err == nil || err.Error() != invalidArgsJSONArrayError {
			t.Fatalf("parseSkillScriptArgsJSON() error = %v, want %q", err, invalidArgsJSONArrayError)
		}
	})

	t.Run("not array", func(t *testing.T) {
		_, err := parseSkillScriptArgsJSON(`{"name":"value"}`)
		if err == nil || err.Error() != invalidArgsJSONArrayError {
			t.Fatalf("parseSkillScriptArgsJSON() error = %v, want %q", err, invalidArgsJSONArrayError)
		}
	})

	t.Run("non string element", func(t *testing.T) {
		_, err := parseSkillScriptArgsJSON(`["ok",123]`)
		want := "invalid args_json: argument at index 1 must be a string"
		if err == nil || err.Error() != want {
			t.Fatalf("parseSkillScriptArgsJSON() error = %v, want %q", err, want)
		}
	})

	t.Run("control character element", func(t *testing.T) {
		_, err := parseSkillScriptArgsJSON(`["ok","line\nbreak"]`)
		want := "invalid args_json: argument at index 1 contains unsupported control characters"
		if err == nil || err.Error() != want {
			t.Fatalf("parseSkillScriptArgsJSON() error = %v, want %q", err, want)
		}
	})

	t.Run("valid array", func(t *testing.T) {
		got, err := parseSkillScriptArgsJSON(`["--name","test user","--json"]`)
		if err != nil {
			t.Fatalf("parseSkillScriptArgsJSON() error = %v", err)
		}
		want := []string{"--name", "test user", "--json"}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("parseSkillScriptArgsJSON() = %#v, want %#v", got, want)
		}
	})
}

func TestParseLegacySkillScriptArgs(t *testing.T) {
	t.Run("reject shell metachar", func(t *testing.T) {
		_, err := parseLegacySkillScriptArgs("; rm -rf /")
		want := "unsafe legacy args; use args_json for quoted values or shell metacharacters"
		if err == nil || err.Error() != want {
			t.Fatalf("parseLegacySkillScriptArgs() error = %v, want %q", err, want)
		}
	})

	t.Run("reject quote", func(t *testing.T) {
		_, err := parseLegacySkillScriptArgs("--name 'test user'")
		want := "unsafe legacy args; use args_json for quoted values or shell metacharacters"
		if err == nil || err.Error() != want {
			t.Fatalf("parseLegacySkillScriptArgs() error = %v, want %q", err, want)
		}
	})

	t.Run("accept simple tokens", func(t *testing.T) {
		got, err := parseLegacySkillScriptArgs("--name test --json")
		if err != nil {
			t.Fatalf("parseLegacySkillScriptArgs() error = %v", err)
		}
		want := []string{"--name", "test", "--json"}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("parseLegacySkillScriptArgs() = %#v, want %#v", got, want)
		}
	})
}
