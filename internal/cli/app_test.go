package cli

import (
	"bytes"
	"database/sql"
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/Ddam-j/mynamr/internal/registry"
	"github.com/Ddam-j/mynamr/internal/version"
	_ "modernc.org/sqlite"
)

func TestVersionFlag(t *testing.T) {
	t.Setenv("TZ", "UTC")

	version.Version = "v0.1.0"
	version.Commit = "abc123"
	version.Date = "2026-03-19T00:00:00Z"

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	t.Setenv("MYNAMR_CONFIG_DIR", t.TempDir())

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
	configDir := t.TempDir()
	t.Setenv("MYNAMR_CONFIG_DIR", configDir)

	app := NewApp(bytes.NewBuffer(nil), stdout, stderr)
	code := app.Run([]string{"rule", "list"})

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d, stderr=%q", code, stderr.String())
	}

	want := "catalog_code_title\nname_alias_bundle\n"
	if stdout.String() != want {
		t.Fatalf("unexpected rule list output: got %q want %q", stdout.String(), want)
	}

	configPath := filepath.Join(configDir, "config.json")
	if _, err := os.Stat(configPath); err != nil {
		t.Fatalf("expected config file to exist: %v", err)
	}

	dbPath := filepath.Join(configDir, "rules.db")
	if _, err := os.Stat(dbPath); err != nil {
		t.Fatalf("expected db file to exist: %v", err)
	}
}

func TestPromptFallback(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	t.Setenv("MYNAMR_CONFIG_DIR", t.TempDir())

	app := NewApp(bytes.NewBufferString("hello world\n"), stdout, stderr)
	code := app.Run(nil)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d, stderr=%q", code, stderr.String())
	}

	if stdout.String() != "hello world\n" {
		t.Fatalf("unexpected prompt output: got %q", stdout.String())
	}
}

func TestClipboardInputAutoDetectsAndWritesClipboard(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	t.Setenv("MYNAMR_CONFIG_DIR", t.TempDir())
	clipboard := &fakeClipboard{
		readText: "코요이 코난 최신작 & 프로필 (Koyoi Konan, 小宵こなん (こよいこなん))",
	}

	app := newAppWithClipboard(bytes.NewBuffer(nil), stdout, stderr, clipboard)
	code := app.Run([]string{"-clip", "-outclip"})

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d, stderr=%q", code, stderr.String())
	}
	want := "코요이 코난.小宵こなん.Koyoi Konan\n"
	if stdout.String() != want {
		t.Fatalf("unexpected stdout: got %q want %q", stdout.String(), want)
	}
	if clipboard.writtenText != strings.TrimSpace(want) {
		t.Fatalf("unexpected clipboard output: got %q want %q", clipboard.writtenText, strings.TrimSpace(want))
	}
}

func TestClipboardInputForcedRule(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	t.Setenv("MYNAMR_CONFIG_DIR", t.TempDir())
	clipboard := &fakeClipboard{readText: "ABP-123 Sample Title"}

	app := newAppWithClipboard(bytes.NewBuffer(nil), stdout, stderr, clipboard)
	code := app.Run([]string{"-clip", "--rule", "catalog_code_title"})

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d, stderr=%q", code, stderr.String())
	}
	if stdout.String() != "[ABP-123] Sample Title\n" {
		t.Fatalf("unexpected forced clipboard output: %q", stdout.String())
	}
}

func TestClipboardInputEmptyTextFails(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	t.Setenv("MYNAMR_CONFIG_DIR", t.TempDir())
	clipboard := &fakeClipboard{readText: "   "}

	app := newAppWithClipboard(bytes.NewBuffer(nil), stdout, stderr, clipboard)
	code := app.Run([]string{"-clip"})

	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
	if !strings.Contains(stderr.String(), "clipboard does not contain usable text") {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
	if stdout.Len() != 0 {
		t.Fatalf("expected no stdout, got %q", stdout.String())
	}
}

func TestClipboardReadErrorFails(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	t.Setenv("MYNAMR_CONFIG_DIR", t.TempDir())
	clipboard := &fakeClipboard{readErr: errors.New("clipboard unavailable")}

	app := newAppWithClipboard(bytes.NewBuffer(nil), stdout, stderr, clipboard)
	code := app.Run([]string{"-clip"})

	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
	if !strings.Contains(stderr.String(), "clipboard unavailable") {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
	if stdout.Len() != 0 {
		t.Fatalf("expected no stdout, got %q", stdout.String())
	}
}

func TestClipboardWriteErrorStillPrintsOutput(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	t.Setenv("MYNAMR_CONFIG_DIR", t.TempDir())
	clipboard := &fakeClipboard{writeErr: errors.New("clipboard busy")}

	app := newAppWithClipboard(bytes.NewBufferString("ABP-123 Sample Title\n"), stdout, stderr, clipboard)
	code := app.Run([]string{"-outclip"})

	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
	if stdout.String() != "[ABP-123] Sample Title\n" {
		t.Fatalf("unexpected stdout on clipboard write error: %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "clipboard busy") {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
}

func TestOutclipWithArgvStillPrintsOutputOnWriteError(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	t.Setenv("MYNAMR_CONFIG_DIR", t.TempDir())
	clipboard := &fakeClipboard{writeErr: errors.New("clipboard busy")}

	app := newAppWithClipboard(bytes.NewBuffer(nil), stdout, stderr, clipboard)
	code := app.Run([]string{"-outclip", "ABP-123 Sample Title"})

	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
	if stdout.String() != "[ABP-123] Sample Title\n" {
		t.Fatalf("unexpected stdout on argv clipboard write error: %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "clipboard busy") {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
}

func TestRuleShowIncludesSource(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	t.Setenv("MYNAMR_CONFIG_DIR", t.TempDir())

	app := NewApp(bytes.NewBuffer(nil), stdout, stderr)
	code := app.Run([]string{"rule", "show", "name_alias_bundle"})

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d, stderr=%q", code, stderr.String())
	}

	got := stdout.String()
	for _, want := range []string{
		"name: name_alias_bundle\n",
		"source: seeded\n",
		"spec:\n  name: name_alias_bundle\n",
		"detect:\n",
		"compose:\n",
		"recognizer: split primary text and parenthesized alias bundle: primary + (en_alias, jp_alias)\n",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("unexpected rule show output, missing %q: %q", want, got)
		}
	}
}

func TestRuleShowSpecOnlyOutputsRawSpec(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	t.Setenv("MYNAMR_CONFIG_DIR", t.TempDir())

	app := NewApp(bytes.NewBuffer(nil), stdout, stderr)
	code := app.Run([]string{"rule", "show", "name_alias_bundle", "--spec-only"})

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d, stderr=%q", code, stderr.String())
	}
	got := stdout.String()
	for _, want := range []string{"name: name_alias_bundle\n", "detect:\n", "compose:\n"} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected raw spec output to include %q: %q", want, got)
		}
	}
	if strings.Contains(got, "source: seeded") {
		t.Fatalf("expected raw spec without metadata labels: %q", got)
	}
}

func TestTopLevelHelp(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	t.Setenv("MYNAMR_CONFIG_DIR", t.TempDir())

	app := NewApp(bytes.NewBuffer(nil), stdout, stderr)
	code := app.Run([]string{"--help"})

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d, stderr=%q", code, stderr.String())
	}

	got := stderr.String()
	for _, want := range []string{
		"Usage:",
		"mynamr [options] [text ...]",
		"mynamr rule <add|list|remove|show|update> [args]",
		"force a rule by name, for example --rule name_alias_bundle",
		"config: ~/.config/mynamr/config.json",
		"default db: ~/.config/mynamr/rules.db",
		"Sync with gdedit:",
		"Register once: mynamr --sync gdedit",
		"mynamr launches gdedit --sync mynamr <name> automatically",
		"PowerShell: .\\mynamr.exe completion powershell | Out-String | Invoke-Expression",
		"PowerShell input with '&' or parentheses must be quoted as one argument",
		".\\mynamr.exe \"코요이 코난 최신작 & 프로필 (Koyoi Konan, 小宵こなん (こよいこなん))\"",
		"미야와키 사쿠라 최신곡 & 프로필 (Sakura Miyawaki, 宮脇咲良)",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("help output missing %q: %q", want, got)
		}
	}
}

func TestRuleHelp(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	t.Setenv("MYNAMR_CONFIG_DIR", t.TempDir())

	app := NewApp(bytes.NewBuffer(nil), stdout, stderr)
	code := app.Run([]string{"rule", "--help"})

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d, stderr=%q", code, stderr.String())
	}

	got := stderr.String()
	for _, want := range []string{
		"mynamr rule list",
		"mynamr rule show <name>",
		"mynamr rule update <name>",
		"--spec <text>",
		"--spec-stdin",
		"Register once with: mynamr --sync gdedit",
		"mynamr launches gdedit automatically when no direct update flags are given",
		"mynamr rule show name_alias_bundle",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("rule help output missing %q: %q", want, got)
		}
	}
}

func TestRuleUpdateWithoutFlagsPrintsGuidance(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	t.Setenv("MYNAMR_CONFIG_DIR", t.TempDir())

	app := NewApp(bytes.NewBuffer(nil), stdout, stderr)
	code := app.Run([]string{"rule", "update", "catalog_code_title"})

	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}

	got := stderr.String()
	for _, want := range []string{
		"Usage:",
		"mynamr rule update catalog_code_title",
		"At least one update flag is required.",
		"--description <text>",
		"--spec <text>",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected update guidance to include %q: %q", want, got)
		}
	}
	if !strings.Contains(got, "error: rule update requires at least one of --description") {
		t.Fatalf("expected actionable error message, got %q", got)
	}
	if stdout.Len() != 0 {
		t.Fatalf("expected no stdout, got %q", stdout.String())
	}
}

func TestSyncRegistrationStoresEditorAndRunsGdedit(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	configDir := t.TempDir()
	t.Setenv("MYNAMR_CONFIG_DIR", configDir)
	runner := &fakeRunner{}

	app := newAppWithDeps(bytes.NewBuffer(nil), stdout, stderr, &fakeClipboard{}, runner, func() (string, error) {
		return `D:\Tools\mynamr.exe`, nil
	})
	code := app.Run([]string{"--sync", "gdedit"})

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d, stderr=%q", code, stderr.String())
	}
	if len(runner.calls) != 1 {
		t.Fatalf("expected one runner call, got %d", len(runner.calls))
	}
	call := runner.calls[0]
	if call.name != "gdedit" {
		t.Fatalf("unexpected runner name: %q", call.name)
	}
	joined := strings.Join(call.args, " ")
	for _, want := range []string{"--sync-register mynamr", "rule show {name} --spec-only", "rule update {name} --spec-stdin"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("expected registration args to include %q: %q", want, joined)
		}
	}
	configData, err := os.ReadFile(filepath.Join(configDir, "config.json"))
	if err != nil {
		t.Fatalf("failed to read config: %v", err)
	}
	if !strings.Contains(string(configData), `"sync_editor": "gdedit"`) {
		t.Fatalf("expected synced editor in config, got %q", string(configData))
	}
	if !strings.Contains(stdout.String(), "synced editor registered: gdedit") {
		t.Fatalf("expected registration message, got %q", stdout.String())
	}
}

func TestSyncRegistrationFailureDoesNotPersistEditor(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	configDir := t.TempDir()
	t.Setenv("MYNAMR_CONFIG_DIR", configDir)
	runner := &fakeRunner{err: errors.New("gdedit not found")}

	app := newAppWithDeps(bytes.NewBuffer(nil), stdout, stderr, &fakeClipboard{}, runner, func() (string, error) {
		return `D:\Tools\mynamr.exe`, nil
	})
	code := app.Run([]string{"--sync", "gdedit"})

	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
	configData, err := os.ReadFile(filepath.Join(configDir, "config.json"))
	if err != nil {
		t.Fatalf("failed to read config: %v", err)
	}
	if strings.Contains(string(configData), `"sync_editor": "gdedit"`) {
		t.Fatalf("did not expect synced editor persisted on failure: %q", string(configData))
	}
}

func TestSyncRegistrationExistingConflictIsAccepted(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	configDir := t.TempDir()
	t.Setenv("MYNAMR_CONFIG_DIR", configDir)
	runner := &fakeRunner{
		err:    errors.New("exit status 1"),
		stdout: `failed to register sync: sync id "mynamr" already exists with different formats`,
	}

	app := newAppWithDeps(bytes.NewBuffer(nil), stdout, stderr, &fakeClipboard{}, runner, func() (string, error) {
		return `D:\Tools\mynamr.exe`, nil
	})
	code := app.Run([]string{"--sync", "gdedit"})

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d, stderr=%q stdout=%q", code, stderr.String(), stdout.String())
	}
	configData, err := os.ReadFile(filepath.Join(configDir, "config.json"))
	if err != nil {
		t.Fatalf("failed to read config: %v", err)
	}
	if !strings.Contains(string(configData), `"sync_editor": "gdedit"`) {
		t.Fatalf("expected synced editor in config, got %q", string(configData))
	}
	if !strings.Contains(stdout.String(), "synced editor already registered: gdedit") {
		t.Fatalf("expected existing-registration message, got %q", stdout.String())
	}
	if strings.Contains(stdout.String(), "failed to register sync") || strings.Contains(stderr.String(), "failed to register sync") {
		t.Fatalf("did not expect raw gdedit conflict output, got stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
}

func TestRuleUpdateWithoutFlagsLaunchesSyncedEditor(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	configDir := t.TempDir()
	t.Setenv("MYNAMR_CONFIG_DIR", configDir)
	if err := os.WriteFile(filepath.Join(configDir, "config.json"), []byte("{\n  \"db_path\": "+quoteForJSON(filepath.Join(configDir, "rules.db"))+",\n  \"sync_editor\": \"gdedit\"\n}\n"), 0o644); err != nil {
		t.Fatalf("failed to seed config: %v", err)
	}
	runner := &fakeRunner{}

	app := newAppWithDeps(bytes.NewBuffer(nil), stdout, stderr, &fakeClipboard{}, runner, func() (string, error) {
		return `D:\Tools\mynamr.exe`, nil
	})
	code := app.Run([]string{"rule", "update", "catalog_code_title"})

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d, stderr=%q", code, stderr.String())
	}
	if len(runner.calls) != 1 {
		t.Fatalf("expected one runner call, got %d", len(runner.calls))
	}
	call := runner.calls[0]
	if call.name != "gdedit" || strings.Join(call.args, " ") != "--sync mynamr catalog_code_title" {
		t.Fatalf("unexpected editor launch call: %#v", call)
	}
	if stdout.Len() != 0 || stderr.Len() != 0 {
		t.Fatalf("expected no direct output, got stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
}

func TestRuleUpdateWithoutFlagsPropagatesEditorLaunchFailure(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	configDir := t.TempDir()
	t.Setenv("MYNAMR_CONFIG_DIR", configDir)
	if err := os.WriteFile(filepath.Join(configDir, "config.json"), []byte("{\n  \"db_path\": "+quoteForJSON(filepath.Join(configDir, "rules.db"))+",\n  \"sync_editor\": \" gdedit \"\n}\n"), 0o644); err != nil {
		t.Fatalf("failed to seed config: %v", err)
	}
	runner := &fakeRunner{err: errors.New("launch failed")}

	app := newAppWithDeps(bytes.NewBuffer(nil), stdout, stderr, &fakeClipboard{}, runner, func() (string, error) {
		return `D:\Tools\mynamr.exe`, nil
	})
	code := app.Run([]string{"rule", "update", "catalog_code_title"})

	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
	if len(runner.calls) != 1 || runner.calls[0].name != "gdedit" {
		t.Fatalf("expected gdedit launch attempt, got %#v", runner.calls)
	}
	if !strings.Contains(stderr.String(), "launch failed") {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
}

func TestRuleUpdateSpecStdinUpdatesStoredSpec(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	t.Setenv("MYNAMR_CONFIG_DIR", t.TempDir())

	newSpec := "name: name_alias_bundle\nenabled: true\ndescription: updated via stdin\ncompose:\n  template: \"{selected.ko[0]}\"\n"
	app := NewApp(bytes.NewBufferString(newSpec), stdout, stderr)
	code := app.Run([]string{"rule", "update", "name_alias_bundle", "--spec-stdin"})

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d, stderr=%q", code, stderr.String())
	}
	if stdout.String() != "updated rule: name_alias_bundle\n" {
		t.Fatalf("unexpected update output: %q", stdout.String())
	}

	stdout = &bytes.Buffer{}
	stderr = &bytes.Buffer{}
	app = NewApp(bytes.NewBuffer(nil), stdout, stderr)
	code = app.Run([]string{"rule", "show", "name_alias_bundle", "--spec-only"})
	if code != 0 {
		t.Fatalf("expected show exit code 0, got %d, stderr=%q", code, stderr.String())
	}
	if stdout.String() != newSpec {
		t.Fatalf("unexpected stored spec: got %q want %q", stdout.String(), newSpec)
	}
}

func TestRuleUpdateRejectsSpecAndSpecStdinTogether(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	t.Setenv("MYNAMR_CONFIG_DIR", t.TempDir())

	app := NewApp(bytes.NewBufferString("name: test\nenabled: true\ncompose:\n  template: \"x\"\n"), stdout, stderr)
	code := app.Run([]string{"rule", "update", "name_alias_bundle", "--spec", "name: test", "--spec-stdin"})

	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
	if !strings.Contains(stderr.String(), "rule update cannot use --spec and --spec-stdin together") {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
}

func TestRuleUpdateSpecStdinRejectsInvalidSpec(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	t.Setenv("MYNAMR_CONFIG_DIR", t.TempDir())

	app := NewApp(bytes.NewBufferString("not: [valid yaml"), stdout, stderr)
	code := app.Run([]string{"rule", "update", "name_alias_bundle", "--spec-stdin"})

	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
	if !strings.Contains(stderr.String(), "parse rule spec") {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
}

func TestRuleUpdateSpecStdinRejectsEmptyInput(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	t.Setenv("MYNAMR_CONFIG_DIR", t.TempDir())

	app := NewApp(bytes.NewBufferString("   \n"), stdout, stderr)
	code := app.Run([]string{"rule", "update", "name_alias_bundle", "--spec-stdin"})

	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
	if !strings.Contains(stderr.String(), "rule update --spec-stdin requires non-empty input") {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
}

func TestCompletionBashCommand(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	t.Setenv("MYNAMR_CONFIG_DIR", t.TempDir())

	app := NewApp(bytes.NewBuffer(nil), stdout, stderr)
	code := app.Run([]string{"completion", "bash"})

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d, stderr=%q", code, stderr.String())
	}

	got := stdout.String()
	for _, want := range []string{
		"_mynamr_completion() {",
		"__complete",
		"complete -o default -F _mynamr_completion mynamr",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("bash completion missing %q: %q", want, got)
		}
	}
}

func TestCompletionPowerShellCommand(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	t.Setenv("MYNAMR_CONFIG_DIR", t.TempDir())

	app := NewApp(bytes.NewBuffer(nil), stdout, stderr)
	code := app.Run([]string{"completion", "powershell"})

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d, stderr=%q", code, stderr.String())
	}

	got := stdout.String()
	for _, want := range []string{
		"Register-ArgumentCompleter -Native -CommandName @('mynamr', 'mynamr.exe', '.\\mynamr.exe')",
		"$commandName = $commandAst.CommandElements[0].Extent.Text",
		"__complete",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("powershell completion missing %q: %q", want, got)
		}
	}
}

func TestInternalCompletionForRuleFlag(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	t.Setenv("MYNAMR_CONFIG_DIR", t.TempDir())

	app := NewApp(bytes.NewBuffer(nil), stdout, stderr)
	code := app.Run([]string{"__complete", "--rule", "name"})

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d, stderr=%q", code, stderr.String())
	}

	if stdout.String() != "name_alias_bundle\n" {
		t.Fatalf("unexpected rule-flag completion: %q", stdout.String())
	}
}

func TestInternalCompletionForRuleShow(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	t.Setenv("MYNAMR_CONFIG_DIR", t.TempDir())

	app := NewApp(bytes.NewBuffer(nil), stdout, stderr)
	code := app.Run([]string{"__complete", "rule", "show", "cat"})

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d, stderr=%q", code, stderr.String())
	}

	if stdout.String() != "catalog_code_title\n" {
		t.Fatalf("unexpected rule-show completion: %q", stdout.String())
	}
}

func TestInternalCompletionForRuleShowEmptyToken(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	t.Setenv("MYNAMR_CONFIG_DIR", t.TempDir())

	app := NewApp(bytes.NewBuffer(nil), stdout, stderr)
	code := app.Run([]string{"__complete", "rule", "show", ""})

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d, stderr=%q", code, stderr.String())
	}

	got := stdout.String()
	for _, want := range []string{"catalog_code_title\n", "name_alias_bundle\n"} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected empty-token completion to include %q: %q", want, got)
		}
	}
}

func TestCompletionRejectsExtraShellArgs(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	t.Setenv("MYNAMR_CONFIG_DIR", t.TempDir())

	app := NewApp(bytes.NewBuffer(nil), stdout, stderr)
	code := app.Run([]string{"completion", "bash", "extra"})

	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}

	if !strings.Contains(stderr.String(), "completion requires exactly one shell name") {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
	if stdout.Len() != 0 {
		t.Fatalf("expected no stdout, got %q", stdout.String())
	}
}

func TestInternalCompletionStopsAfterRuleShowArg(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	t.Setenv("MYNAMR_CONFIG_DIR", t.TempDir())

	app := NewApp(bytes.NewBuffer(nil), stdout, stderr)
	code := app.Run([]string{"__complete", "rule", "show", "catalog_code_title", "extra"})

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d, stderr=%q", code, stderr.String())
	}

	if stdout.Len() != 0 {
		t.Fatalf("expected no completion suggestions, got %q", stdout.String())
	}
}

func TestInternalCompletionIncludesRuleListSubcommand(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	t.Setenv("MYNAMR_CONFIG_DIR", t.TempDir())

	app := NewApp(bytes.NewBuffer(nil), stdout, stderr)
	code := app.Run([]string{"__complete", "rule", ""})

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d, stderr=%q", code, stderr.String())
	}

	got := stdout.String()
	for _, want := range []string{"add\n", "list\n", "remove\n", "show\n", "update\n"} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected rule subcommand completion to include %q: %q", want, got)
		}
	}
}

func TestRuleUpdateCompletionIncludesFlags(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	t.Setenv("MYNAMR_CONFIG_DIR", t.TempDir())

	app := NewApp(bytes.NewBuffer(nil), stdout, stderr)
	code := app.Run([]string{"__complete", "rule", "update", "name_alias_bundle", "--"})

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d, stderr=%q", code, stderr.String())
	}

	got := stdout.String()
	for _, want := range []string{"--description\n", "--enable\n", "--disable\n", "--spec\n", "--recognizer\n", "--matcher\n", "--selector\n", "--renderer\n"} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected update flag completion to include %q: %q", want, got)
		}
	}
}

func TestRuleAddCompletionIncludesFlags(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	t.Setenv("MYNAMR_CONFIG_DIR", t.TempDir())

	app := NewApp(bytes.NewBuffer(nil), stdout, stderr)
	code := app.Run([]string{"__complete", "rule", "add", "sample_rule", "--"})

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d, stderr=%q", code, stderr.String())
	}

	got := stdout.String()
	for _, want := range []string{"--description\n", "--enabled\n", "--spec\n", "--recognizer\n", "--matcher\n", "--selector\n", "--renderer\n"} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected add flag completion to include %q: %q", want, got)
		}
	}
}

func TestRuleAddAndShowStoredDefinitions(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv("MYNAMR_CONFIG_DIR", configDir)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	app := NewApp(bytes.NewBuffer(nil), stdout, stderr)
	code := app.Run([]string{
		"rule", "add", "script_rule",
		"--description", "Rule with stored definitions",
		"--recognizer", "recognize by demo pattern",
		"--matcher", "match when demo pattern succeeds",
		"--selector", "select the first capture",
		"--renderer", "render with {value}",
	})
	if code != 0 {
		t.Fatalf("expected add exit code 0, got %d, stderr=%q", code, stderr.String())
	}

	stdout = &bytes.Buffer{}
	stderr = &bytes.Buffer{}
	app = NewApp(bytes.NewBuffer(nil), stdout, stderr)
	code = app.Run([]string{"rule", "show", "script_rule"})
	if code != 0 {
		t.Fatalf("expected show exit code 0, got %d, stderr=%q", code, stderr.String())
	}
	got := stdout.String()
	for _, want := range []string{
		"source: user\n",
		"spec:\n  \n",
		"recognizer: recognize by demo pattern\n",
		"matcher: match when demo pattern succeeds\n",
		"selector: select the first capture\n",
		"renderer: render with {value}\n",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected stored definition output to include %q: %q", want, got)
		}
	}
}

func TestForcedRuleExecutesNameAliasBundle(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	t.Setenv("MYNAMR_CONFIG_DIR", t.TempDir())

	app := NewApp(bytes.NewBuffer(nil), stdout, stderr)
	code := app.Run([]string{"--rule", "name_alias_bundle", "코요이 코난 최신작 & 프로필 (Koyoi Konan, 小宵こなん (こよいこなん))"})

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d, stderr=%q", code, stderr.String())
	}

	if stdout.String() != "코요이 코난.小宵こなん.Koyoi Konan\n" {
		t.Fatalf("unexpected forced rule output: %q", stdout.String())
	}
}

func TestForcedRuleExecutesCatalogCodeTitle(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	t.Setenv("MYNAMR_CONFIG_DIR", t.TempDir())

	app := NewApp(bytes.NewBuffer(nil), stdout, stderr)
	code := app.Run([]string{"--rule", "catalog_code_title", "ABP-123 Sample Title"})

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d, stderr=%q", code, stderr.String())
	}

	if stdout.String() != "[ABP-123] Sample Title\n" {
		t.Fatalf("unexpected catalog rule output: %q", stdout.String())
	}
}

func TestCatalogCodeTitleDropWordsApplyToAfterCode(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv("MYNAMR_CONFIG_DIR", configDir)

	spec := "name: catalog_code_title\nenabled: true\ndescription: catalog title cleanup\n\ndetect:\n  all:\n    - has_leading_catalog_code\n  priority: 70\n\ndrop_words:\n  - Latest\n  - Profile\n\ngroups:\n  code:\n    sources:\n      - input.leading_catalog_code\n  title:\n    sources:\n      - input.after_catalog_code\n\nselect:\n  groups:\n    code:\n      policy: first\n    title:\n      policy: first\n\ncompose:\n  template: \"[{selected.code[0]}] {selected.title[0]}\"\n"

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	app := NewApp(bytes.NewBufferString(spec), stdout, stderr)
	code := app.Run([]string{"rule", "update", "catalog_code_title", "--spec-stdin"})
	if code != 0 {
		t.Fatalf("expected spec update exit code 0, got %d, stderr=%q", code, stderr.String())
	}

	stdout = &bytes.Buffer{}
	stderr = &bytes.Buffer{}
	app = NewApp(bytes.NewBuffer(nil), stdout, stderr)
	code = app.Run([]string{"--rule", "catalog_code_title", "ABP-123 Latest Profile Title"})
	if code != 0 {
		t.Fatalf("expected forced rule exit code 0, got %d, stderr=%q", code, stderr.String())
	}
	if stdout.String() != "[ABP-123] Title\n" {
		t.Fatalf("unexpected catalog drop-word output: %q", stdout.String())
	}
}

func TestCatalogCodeTitleLeadingSeparatorRemovedButInnerDashPreserved(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv("MYNAMR_CONFIG_DIR", configDir)

	spec := "name: catalog_code_title\nenabled: true\ndescription: preserve inner dash\n\ndetect:\n  all:\n    - has_leading_catalog_code\n  priority: 70\n\nprefix_drop_words:\n  - \" - \"\n\ngroups:\n  code:\n    sources:\n      - input.leading_catalog_code\n  title:\n    sources:\n      - input.after_catalog_code\n\nselect:\n  groups:\n    code:\n      policy: first\n    title:\n      policy: first\n\ncompose:\n  template: \"[{selected.code[0]}] {selected.title[0]}\"\n"

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	app := NewApp(bytes.NewBufferString(spec), stdout, stderr)
	code := app.Run([]string{"rule", "update", "catalog_code_title", "--spec-stdin"})
	if code != 0 {
		t.Fatalf("expected spec update exit code 0, got %d, stderr=%q", code, stderr.String())
	}

	stdout = &bytes.Buffer{}
	stderr = &bytes.Buffer{}
	app = NewApp(bytes.NewBuffer(nil), stdout, stderr)
	code = app.Run([]string{"--rule", "catalog_code_title", "SNOS-084 - \"A - B\" 소꿉친구"})
	if code != 0 {
		t.Fatalf("expected forced rule exit code 0, got %d, stderr=%q", code, stderr.String())
	}
	if stdout.String() != "[SNOS-084] \"A - B\" 소꿉친구\n" {
		t.Fatalf("unexpected preserved inner dash output: %q", stdout.String())
	}
}

func TestAutoDetectExecutesNameAliasBundle(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	t.Setenv("MYNAMR_CONFIG_DIR", t.TempDir())

	app := NewApp(bytes.NewBuffer(nil), stdout, stderr)
	code := app.Run([]string{"코요이 코난 최신작 & 프로필 (Koyoi Konan, 小宵こなん (こよいこなん))"})

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d, stderr=%q", code, stderr.String())
	}

	if stdout.String() != "코요이 코난.小宵こなん.Koyoi Konan\n" {
		t.Fatalf("unexpected auto-detect alias output: %q", stdout.String())
	}
}

func TestAutoDetectExecutesCatalogCodeTitle(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	t.Setenv("MYNAMR_CONFIG_DIR", t.TempDir())

	app := NewApp(bytes.NewBuffer(nil), stdout, stderr)
	code := app.Run([]string{"ABP-123 Sample Title"})

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d, stderr=%q", code, stderr.String())
	}

	if stdout.String() != "[ABP-123] Sample Title\n" {
		t.Fatalf("unexpected auto-detect catalog output: %q", stdout.String())
	}
}

func TestAutoDetectFallsBackToOriginalInput(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	t.Setenv("MYNAMR_CONFIG_DIR", t.TempDir())

	app := NewApp(bytes.NewBuffer(nil), stdout, stderr)
	code := app.Run([]string{"plain input without any registered pattern"})

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d, stderr=%q", code, stderr.String())
	}

	if stdout.String() != "plain input without any registered pattern\n" {
		t.Fatalf("unexpected fallback output: %q", stdout.String())
	}
}

func TestRuleUpdateCanClearStoredDefinitions(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv("MYNAMR_CONFIG_DIR", configDir)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	app := NewApp(bytes.NewBuffer(nil), stdout, stderr)
	code := app.Run([]string{
		"rule", "add", "clear_rule",
		"--description", "Rule to clear",
		"--recognizer", "recognize something",
		"--renderer", "render something",
	})
	if code != 0 {
		t.Fatalf("expected add exit code 0, got %d, stderr=%q", code, stderr.String())
	}

	stdout = &bytes.Buffer{}
	stderr = &bytes.Buffer{}
	app = NewApp(bytes.NewBuffer(nil), stdout, stderr)
	code = app.Run([]string{"rule", "update", "clear_rule", "--recognizer", "", "--renderer", ""})
	if code != 0 {
		t.Fatalf("expected update exit code 0, got %d, stderr=%q", code, stderr.String())
	}

	stdout = &bytes.Buffer{}
	stderr = &bytes.Buffer{}
	app = NewApp(bytes.NewBuffer(nil), stdout, stderr)
	code = app.Run([]string{"rule", "show", "clear_rule"})
	if code != 0 {
		t.Fatalf("expected show exit code 0, got %d, stderr=%q", code, stderr.String())
	}
	got := stdout.String()
	for _, want := range []string{"recognizer: \n", "renderer: \n"} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected cleared stored definition output to include %q: %q", want, got)
		}
	}
}

func TestInternalCompletionFiltersRuleListSubcommand(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	t.Setenv("MYNAMR_CONFIG_DIR", t.TempDir())

	app := NewApp(bytes.NewBuffer(nil), stdout, stderr)
	code := app.Run([]string{"__complete", "rule", "l"})

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d, stderr=%q", code, stderr.String())
	}

	if stdout.String() != "list\n" {
		t.Fatalf("unexpected filtered rule completion: %q", stdout.String())
	}
}

func TestRuleAddAndRemoveLifecycleUpdatesCompletion(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv("MYNAMR_CONFIG_DIR", configDir)

	app := NewApp(bytes.NewBuffer(nil), &bytes.Buffer{}, &bytes.Buffer{})

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	app = NewApp(bytes.NewBuffer(nil), stdout, stderr)
	code := app.Run([]string{"rule", "add", "sample_rule", "--description", "User-defined sample rule"})
	if code != 0 {
		t.Fatalf("expected add exit code 0, got %d, stderr=%q", code, stderr.String())
	}
	if stdout.String() != "added rule: sample_rule\n" {
		t.Fatalf("unexpected add output: %q", stdout.String())
	}

	stdout = &bytes.Buffer{}
	stderr = &bytes.Buffer{}
	app = NewApp(bytes.NewBuffer(nil), stdout, stderr)
	code = app.Run([]string{"__complete", "rule", "show", "sam"})
	if code != 0 {
		t.Fatalf("expected completion exit code 0, got %d, stderr=%q", code, stderr.String())
	}
	if stdout.String() != "sample_rule\n" {
		t.Fatalf("unexpected completion after add: %q", stdout.String())
	}

	stdout = &bytes.Buffer{}
	stderr = &bytes.Buffer{}
	app = NewApp(bytes.NewBuffer(nil), stdout, stderr)
	code = app.Run([]string{"rule", "show", "sample_rule"})
	if code != 0 {
		t.Fatalf("expected show exit code 0, got %d, stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "source: user\n") {
		t.Fatalf("expected user source in show output: %q", stdout.String())
	}

	stdout = &bytes.Buffer{}
	stderr = &bytes.Buffer{}
	app = NewApp(bytes.NewBuffer(nil), stdout, stderr)
	code = app.Run([]string{"rule", "remove", "sample_rule"})
	if code != 0 {
		t.Fatalf("expected remove exit code 0, got %d, stderr=%q", code, stderr.String())
	}
	if stdout.String() != "removed rule: sample_rule\n" {
		t.Fatalf("unexpected remove output: %q", stdout.String())
	}

	stdout = &bytes.Buffer{}
	stderr = &bytes.Buffer{}
	app = NewApp(bytes.NewBuffer(nil), stdout, stderr)
	code = app.Run([]string{"__complete", "rule", "show", "sam"})
	if code != 0 {
		t.Fatalf("expected post-remove completion exit code 0, got %d, stderr=%q", code, stderr.String())
	}
	if stdout.Len() != 0 {
		t.Fatalf("expected removed rule to disappear from completion, got %q", stdout.String())
	}
}

func TestCannotAddDuplicateSeededRuleName(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	t.Setenv("MYNAMR_CONFIG_DIR", t.TempDir())

	app := NewApp(bytes.NewBuffer(nil), stdout, stderr)
	code := app.Run([]string{"rule", "add", "name_alias_bundle", "--description", "duplicate seeded"})
	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
	if !strings.Contains(stderr.String(), "rule \"name_alias_bundle\" already exists") {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
}

func TestSeededRuleUpdatePersists(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv("MYNAMR_CONFIG_DIR", configDir)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	app := NewApp(bytes.NewBuffer(nil), stdout, stderr)
	code := app.Run([]string{"rule", "update", "name_alias_bundle", "--description", "Stored registry rule", "--disable"})
	if code != 0 {
		t.Fatalf("expected update exit code 0, got %d, stderr=%q", code, stderr.String())
	}

	stdout = &bytes.Buffer{}
	stderr = &bytes.Buffer{}
	app = NewApp(bytes.NewBuffer(nil), stdout, stderr)
	code = app.Run([]string{"rule", "show", "name_alias_bundle"})
	if code != 0 {
		t.Fatalf("expected show exit code 0, got %d, stderr=%q", code, stderr.String())
	}
	got := stdout.String()
	for _, want := range []string{"enabled: false\n", "source: seeded\n", "description: Stored registry rule\n"} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected updated seeded rule output to include %q: %q", want, got)
		}
	}
}

func TestSeededRuleRemovePersistsAcrossRestart(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv("MYNAMR_CONFIG_DIR", configDir)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	app := NewApp(bytes.NewBuffer(nil), stdout, stderr)
	code := app.Run([]string{"rule", "remove", "name_alias_bundle"})
	if code != 0 {
		t.Fatalf("expected remove exit code 0, got %d, stderr=%q", code, stderr.String())
	}

	stdout = &bytes.Buffer{}
	stderr = &bytes.Buffer{}
	app = NewApp(bytes.NewBuffer(nil), stdout, stderr)
	code = app.Run([]string{"rule", "list"})
	if code != 0 {
		t.Fatalf("expected list exit code 0, got %d, stderr=%q", code, stderr.String())
	}
	if strings.Contains(stdout.String(), "name_alias_bundle\n") {
		t.Fatalf("expected removed seeded rule to stay removed after reopen: %q", stdout.String())
	}
}

func TestRuleUpdateCompletionIncludesSeededRule(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	t.Setenv("MYNAMR_CONFIG_DIR", t.TempDir())

	app := NewApp(bytes.NewBuffer(nil), stdout, stderr)
	code := app.Run([]string{"__complete", "rule", "update", "name"})
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d, stderr=%q", code, stderr.String())
	}
	if stdout.String() != "name_alias_bundle\n" {
		t.Fatalf("unexpected update completion output: %q", stdout.String())
	}
}

func TestRemoveCompletionShowsStoredRules(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv("MYNAMR_CONFIG_DIR", configDir)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	app := NewApp(bytes.NewBuffer(nil), stdout, stderr)
	code := app.Run([]string{"rule", "add", "sample_rule", "--description", "User-defined sample rule"})
	if code != 0 {
		t.Fatalf("expected add exit code 0, got %d, stderr=%q", code, stderr.String())
	}

	stdout = &bytes.Buffer{}
	stderr = &bytes.Buffer{}
	app = NewApp(bytes.NewBuffer(nil), stdout, stderr)
	code = app.Run([]string{"__complete", "rule", "remove", ""})
	if code != 0 {
		t.Fatalf("expected completion exit code 0, got %d, stderr=%q", code, stderr.String())
	}
	got := stdout.String()
	if !strings.Contains(got, "sample_rule\n") {
		t.Fatalf("expected added rule in remove completion: %q", got)
	}
	if !strings.Contains(got, "name_alias_bundle\n") || !strings.Contains(got, "catalog_code_title\n") {
		t.Fatalf("expected seeded rules to be included in remove completion: %q", got)
	}
}

func TestRelativeDBPathIsNormalizedAndPersisted(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv("MYNAMR_CONFIG_DIR", configDir)

	configPath := filepath.Join(configDir, "config.json")
	if err := os.WriteFile(configPath, []byte("{\n  \"db_path\": \"data/custom.db\"\n}\n"), 0o644); err != nil {
		t.Fatalf("failed to seed config: %v", err)
	}

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	app := NewApp(bytes.NewBuffer(nil), stdout, stderr)
	code := app.Run([]string{"rule", "list"})

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d, stderr=%q", code, stderr.String())
	}

	wantDBPath := filepath.Join(configDir, "data", "custom.db")
	if _, err := os.Stat(wantDBPath); err != nil {
		t.Fatalf("expected normalized db file to exist: %v", err)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read rewritten config: %v", err)
	}

	wantConfig := "{\n  \"db_path\": " + quoteForJSON(filepath.Clean(wantDBPath)) + "\n}\n"
	if string(data) != wantConfig {
		t.Fatalf("unexpected config contents: got %q want %q", string(data), wantConfig)
	}
}

func TestSeedSkipsExistingRuleAndCommitsSeedVersion(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv("MYNAMR_CONFIG_DIR", configDir)
	dbPath := filepath.Join(configDir, "rules.db")

	store, err := registry.Open(dbPath)
	if err != nil {
		t.Fatalf("failed to open registry: %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("failed to close registry: %v", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("failed to open db directly: %v", err)
	}
	defer db.Close()

	if _, err := db.Exec(`DELETE FROM registry_meta WHERE key = 'seed_version'`); err != nil {
		t.Fatalf("failed to clear seed version: %v", err)
	}
	if _, err := db.Exec(`UPDATE rules SET description = 'custom seeded description' WHERE name = 'name_alias_bundle'`); err != nil {
		t.Fatalf("failed to mutate seeded rule: %v", err)
	}

	store, err = registry.Open(dbPath)
	if err != nil {
		t.Fatalf("failed to reopen registry: %v", err)
	}
	defer store.Close()

	rule, err := store.Rule("name_alias_bundle")
	if err != nil {
		t.Fatalf("failed to load rule: %v", err)
	}
	if rule.Description != "custom seeded description" {
		t.Fatalf("expected existing rule to survive reseed, got %q", rule.Description)
	}

	var seedVersion string
	err = db.QueryRow(`SELECT value FROM registry_meta WHERE key = 'seed_version'`).Scan(&seedVersion)
	if err != nil {
		t.Fatalf("failed to read seed version: %v", err)
	}
	if seedVersion != "1" {
		t.Fatalf("unexpected seed version: %q", seedVersion)
	}
}

func TestSeedBackfillsDefinitionsForExistingSeededRows(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv("MYNAMR_CONFIG_DIR", configDir)
	dbPath := filepath.Join(configDir, "rules.db")

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("failed to open db directly: %v", err)
	}
	defer db.Close()

	if _, err := db.Exec(`
		CREATE TABLE registry_meta (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL,
			updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`); err != nil {
		t.Fatalf("failed to create registry_meta: %v", err)
	}
	if _, err := db.Exec(`
		CREATE TABLE rules (
			name TEXT PRIMARY KEY,
			description TEXT NOT NULL,
			enabled BOOLEAN NOT NULL,
			source TEXT NOT NULL,
			recognizer TEXT NOT NULL DEFAULT '',
			matcher TEXT NOT NULL DEFAULT '',
			selector TEXT NOT NULL DEFAULT '',
			renderer TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`); err != nil {
		t.Fatalf("failed to create rules: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO registry_meta (key, value) VALUES ('seed_version', '1')`); err != nil {
		t.Fatalf("failed to seed registry meta: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO rules (name, description, enabled, source, recognizer, matcher, selector, renderer)
		VALUES ('catalog_code_title', 'legacy seeded row', 1, 'seeded', '', '', '', '')
	`); err != nil {
		t.Fatalf("failed to seed legacy rule row: %v", err)
	}

	store, err := registry.Open(dbPath)
	if err != nil {
		t.Fatalf("failed to reopen registry: %v", err)
	}
	defer store.Close()

	rule, err := store.Rule("catalog_code_title")
	if err != nil {
		t.Fatalf("failed to load rule: %v", err)
	}
	for field, got := range map[string]string{
		"recognizer": rule.Recognizer,
		"matcher":    rule.Matcher,
		"selector":   rule.Selector,
		"renderer":   rule.Renderer,
	} {
		if got == "" {
			t.Fatalf("expected backfilled %s, got empty string", field)
		}
	}
	if rule.Description != "legacy seeded row" {
		t.Fatalf("expected existing description to remain untouched, got %q", rule.Description)
	}
}

type fakeClipboard struct {
	readText    string
	readErr     error
	writeErr    error
	writtenText string
}

func (f *fakeClipboard) ReadText() (string, error) {
	return f.readText, f.readErr
}

func (f *fakeClipboard) WriteText(text string) error {
	f.writtenText = text
	return f.writeErr
}

type runnerCall struct {
	name string
	args []string
}

type fakeRunner struct {
	calls  []runnerCall
	err    error
	stdout string
	stderr string
}

func (f *fakeRunner) Run(name string, args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	callArgs := append([]string(nil), args...)
	f.calls = append(f.calls, runnerCall{name: name, args: callArgs})
	if f.stdout != "" {
		_, _ = io.WriteString(stdout, f.stdout)
	}
	if f.stderr != "" {
		_, _ = io.WriteString(stderr, f.stderr)
	}
	return f.err
}

func quoteForJSON(path string) string {
	return `"` + strings.ReplaceAll(path, `\`, `\\`) + `"`
}

func TestPrepareCommandWrapsCmdOnWindows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("windows-only command resolution test")
	}
	commandName, commandArgs, err := prepareCommand("gdedit", []string{"--sync", "mynamr", "catalog_code_title"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if commandName != "cmd.exe" {
		t.Fatalf("expected cmd.exe launcher, got %q", commandName)
	}
	if len(commandArgs) < 3 || commandArgs[0] != "/c" || !strings.HasSuffix(strings.ToLower(commandArgs[1]), "gdedit.cmd") {
		t.Fatalf("unexpected command args: %#v", commandArgs)
	}
}
