package commandruntime

import "testing"

func TestParse_ResolvesAliasesAndQuotes(t *testing.T) {
	got, ok := Parse(`/h "hello world"`, map[string]string{"/x": "/status"})
	if !ok {
		t.Fatal("Parse() ok = false, want true")
	}
	if got.Command != "/help" {
		t.Fatalf("Command = %q, want /help", got.Command)
	}
	if len(got.Args) != 1 || got.Args[0] != "hello world" {
		t.Fatalf("Args = %#v, want quoted argument", got.Args)
	}
}

func TestDispatch_RunsRegisteredHandler(t *testing.T) {
	var gotArgs []string
	handled := Dispatch(`/x one two`, map[string]string{"/x": "/run"}, Registry{
		"/run": func(args []string) bool {
			gotArgs = args
			return true
		},
	})
	if !handled {
		t.Fatal("Dispatch() = false, want true")
	}
	if len(gotArgs) != 2 || gotArgs[0] != "one" || gotArgs[1] != "two" {
		t.Fatalf("handler args = %#v", gotArgs)
	}
}

func TestIsNonInteractiveConfigSubcommand(t *testing.T) {
	if !IsNonInteractiveConfigSubcommand([]string{"show"}) {
		t.Fatal("show should be non-interactive")
	}
	if !IsNonInteractiveConfigSubcommand([]string{"model", "gpt-test"}) {
		t.Fatal("model <name> should be non-interactive")
	}
	if IsNonInteractiveConfigSubcommand(nil) {
		t.Fatal("bare /config should be interactive")
	}
}
