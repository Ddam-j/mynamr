package cli

import (
	"bytes"
	"testing"

	"github.com/Ddam-j/mynamr/internal/version"
)

func TestVersionFlag(t *testing.T) {
	t.Setenv("TZ", "UTC")

	version.Version = "v0.1.0"
	version.Commit = "abc123"
	version.Date = "2026-03-19T00:00:00Z"

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	app := NewApp(bytes.NewBuffer(nil), stdout, stderr)
	code := app.Run([]string{"--version"})

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d, stderr=%q", code, stderr.String())
	}

	want := "mynamr v0.1.0 (commit: abc123, built at: 2026-03-19T00:00:00Z)\n"
	if stdout.String() != want {
		t.Fatalf("unexpected version output: got %q want %q", stdout.String(), want)
	}
}

func TestRuleListCommand(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	app := NewApp(bytes.NewBuffer(nil), stdout, stderr)
	code := app.Run([]string{"rule", "list"})

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d, stderr=%q", code, stderr.String())
	}

	want := "catalog_code_title\nname_alias_bundle\n"
	if stdout.String() != want {
		t.Fatalf("unexpected rule list output: got %q want %q", stdout.String(), want)
	}
}

func TestPromptFallback(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	app := NewApp(bytes.NewBufferString("hello world\n"), stdout, stderr)
	code := app.Run(nil)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d, stderr=%q", code, stderr.String())
	}

	if stdout.String() != "hello world\n" {
		t.Fatalf("unexpected prompt output: got %q", stdout.String())
	}
}
