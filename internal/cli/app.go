package cli

import (
	"bufio"
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/Ddam-j/mynamr/internal/clip"
	"github.com/Ddam-j/mynamr/internal/config"
	"github.com/Ddam-j/mynamr/internal/registry"
	"github.com/Ddam-j/mynamr/internal/rules"
	"github.com/Ddam-j/mynamr/internal/version"
)

type App struct {
	in     io.Reader
	stdout io.Writer
	stderr io.Writer
	clip   clip.Interface
	runner commandRunner
	exe    func() (string, error)
}

func NewApp(in io.Reader, stdout, stderr io.Writer) *App {
	return &App{in: in, stdout: stdout, stderr: stderr, clip: clip.New(), runner: systemRunner{}, exe: os.Executable}
}

func newAppWithClipboard(in io.Reader, stdout, stderr io.Writer, clipboard clip.Interface) *App {
	return &App{in: in, stdout: stdout, stderr: stderr, clip: clipboard, runner: systemRunner{}, exe: os.Executable}
}

func newAppWithDeps(in io.Reader, stdout, stderr io.Writer, clipboard clip.Interface, runner commandRunner, exe func() (string, error)) *App {
	return &App{in: in, stdout: stdout, stderr: stderr, clip: clipboard, runner: runner, exe: exe}
}

func (a *App) Run(args []string) int {
	code, err := a.run(args)
	if err != nil {
		_, _ = fmt.Fprintln(a.stderr, "error:", err)
	}
	return code
}

func (a *App) run(args []string) (int, error) {
	fs := flag.NewFlagSet("mynamr", flag.ContinueOnError)
	fs.SetOutput(a.stderr)
	fs.Usage = func() {
		a.printUsage(fs)
	}

	useClipboard := fs.Bool("clip", false, "read input from the clipboard")
	writeClipboard := fs.Bool("outclip", false, "write output to the clipboard")
	showJSON := fs.Bool("json", false, "print structured JSON output")
	showStatus := fs.Bool("status", false, "print status information")
	forcedRule := fs.String("rule", "", "force a rule by name, for example --rule name_alias_bundle")
	syncEditor := fs.String("sync", "", "register and use an editor sync target, for example gdedit")
	listRules := fs.Bool("list-rules", false, "list registered rules")
	showVersion := fs.Bool("version", false, "print build version")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0, nil
		}
		return 2, err
	}

	if *showVersion {
		_, err := fmt.Fprintln(a.stdout, version.Detailed())
		return 0, err
	}

	remaining := fs.Args()
	if len(remaining) > 0 && remaining[0] == "completion" {
		return a.runCompletionCommand(remaining[1:])
	}

	resolvedConfig, err := config.Ensure()
	if err != nil {
		return 1, err
	}

	store, err := registry.Open(resolvedConfig.DBPath)
	if err != nil {
		return 1, err
	}
	defer store.Close()

	if strings.TrimSpace(*syncEditor) != "" {
		return a.runSyncRegistration(strings.TrimSpace(*syncEditor))
	}

	if len(remaining) > 0 && remaining[0] == "__complete" {
		return a.runInternalCompletion(store, remaining[1:])
	}
	if len(remaining) > 0 && remaining[0] == "rule" {
		return a.runRuleCommand(store, resolvedConfig, remaining[1:])
	}

	switch {
	case *listRules:
		return a.printRuleList(store)
	case *showJSON:
		return 1, errors.New("--json is scaffolded but JSON status output is not implemented yet")
	case *showStatus:
		return a.printStatus(*forcedRule, resolvedConfig)
	}

	input, err := a.resolveInput(remaining, *useClipboard)
	if err != nil {
		return 1, err
	}

	output := input
	if *forcedRule != "" {
		rule, err := store.Rule(*forcedRule)
		if err != nil {
			return 1, err
		}
		if !rule.Enabled {
			return 1, fmt.Errorf("rule %q is disabled", rule.Name)
		}
		transformed, err := rules.Execute(rule, input)
		if err != nil {
			return 1, err
		}
		output = transformed
	} else {
		enabledRules, err := store.ListEnabledRules()
		if err != nil {
			return 1, err
		}
		if transformed, matched, err := rules.SelectAndExecute(enabledRules, input); err != nil {
			return 1, err
		} else if matched {
			output = transformed
		}
	}

	if *writeClipboard {
		if err := a.clip.WriteText(output); err != nil {
			_, _ = fmt.Fprintln(a.stdout, output)
			return 1, err
		}
	}

	_, err = fmt.Fprintln(a.stdout, output)
	return 0, err
}

func (a *App) printRuleList(store *registry.Store) (int, error) {
	rules, err := store.ListRules()
	if err != nil {
		return 1, err
	}

	for _, rule := range rules {
		if _, err := fmt.Fprintln(a.stdout, rule.Name); err != nil {
			return 1, err
		}
	}
	return 0, nil
}

func (a *App) printUsage(fs *flag.FlagSet) {
	_, _ = fmt.Fprintln(a.stderr, "Usage:")
	_, _ = fmt.Fprintln(a.stderr, "  mynamr [options] [text ...]")
	_, _ = fmt.Fprintln(a.stderr, "  mynamr rule <add|list|remove|show|update> [args]")
	_, _ = fmt.Fprintln(a.stderr)
	_, _ = fmt.Fprintln(a.stderr, "Commands:")
	_, _ = fmt.Fprintln(a.stderr, "  completion <shell> Generate a shell completion script")
	_, _ = fmt.Fprintln(a.stderr, "  rule add <name>    Add a user rule to the registry database")
	_, _ = fmt.Fprintln(a.stderr, "  rule list          List rules from the registry database")
	_, _ = fmt.Fprintln(a.stderr, "  rule remove <name> Remove a rule from the registry database")
	_, _ = fmt.Fprintln(a.stderr, "  rule show <name>   Show a rule from the registry database")
	_, _ = fmt.Fprintln(a.stderr, "  rule update <name> Update a stored rule in the registry database")
	_, _ = fmt.Fprintln(a.stderr)
	_, _ = fmt.Fprintln(a.stderr, "Options:")
	fs.PrintDefaults()
	_, _ = fmt.Fprintln(a.stderr)
	_, _ = fmt.Fprintln(a.stderr, "Registry:")
	_, _ = fmt.Fprintln(a.stderr, "  config: ~/.config/mynamr/config.json")
	_, _ = fmt.Fprintln(a.stderr, "  default db: ~/.config/mynamr/rules.db")
	_, _ = fmt.Fprintln(a.stderr, "  config key: db_path")
	_, _ = fmt.Fprintln(a.stderr)
	_, _ = fmt.Fprintln(a.stderr, "Sync with gdedit:")
	_, _ = fmt.Fprintln(a.stderr, "  1. Register once: mynamr --sync gdedit")
	_, _ = fmt.Fprintln(a.stderr, "  2. Edit later:    mynamr rule update <name>")
	_, _ = fmt.Fprintln(a.stderr, "  3. mynamr launches gdedit --sync mynamr <name> automatically")
	_, _ = fmt.Fprintln(a.stderr, "  4. Under the hood gdedit reads --spec-only and saves with --spec-stdin")
	_, _ = fmt.Fprintln(a.stderr, "  5. PowerShell users should run .\\mynamr.exe, not bare mynamr, unless PATH is configured")
	_, _ = fmt.Fprintln(a.stderr)
	_, _ = fmt.Fprintln(a.stderr, "Completion setup:")
	_, _ = fmt.Fprintln(a.stderr, "  Bash:       source <(mynamr completion bash)")
	_, _ = fmt.Fprintln(a.stderr, "  PowerShell: .\\mynamr.exe completion powershell | Out-String | Invoke-Expression")
	_, _ = fmt.Fprintln(a.stderr, "  PowerShell input with '&' or parentheses must be quoted as one argument")
	_, _ = fmt.Fprintln(a.stderr)
	_, _ = fmt.Fprintln(a.stderr, "Examples:")
	_, _ = fmt.Fprintln(a.stderr, "  mynamr hello world")
	_, _ = fmt.Fprintln(a.stderr, "  .\\mynamr.exe \"하늘 정원 아카이브 최신작 & 프로필 (Sky Garden Archive, 青空庭園)\"")
	_, _ = fmt.Fprintln(a.stderr, "  mynamr --rule name_alias_bundle \"하늘 정원 아카이브 최신작 & 프로필 (Sky Garden Archive, 青空庭園)\"")
	_, _ = fmt.Fprintln(a.stderr, "  mynamr rule add sample_rule --description \"User-defined sample rule\"")
	_, _ = fmt.Fprintln(a.stderr, "  mynamr rule show name_alias_bundle")
	_, _ = fmt.Fprintln(a.stderr, "  .\\mynamr.exe --sync gdedit")
	_, _ = fmt.Fprintln(a.stderr, "  .\\mynamr.exe rule update catalog_code_title")
	_, _ = fmt.Fprintln(a.stderr, "  mynamr --rule name_alias_bundle \"하늘 정원 아카이브 최신작 & 프로필 (Sky Garden Archive, 青空庭園)\"")
	_, _ = fmt.Fprintln(a.stderr, "  mynamr rule update name_alias_bundle --description \"Stored registry rule\"")
	_, _ = fmt.Fprintln(a.stderr, "  mynamr rule list")
	_, _ = fmt.Fprintln(a.stderr, "  .\\mynamr.exe completion powershell | Out-String | Invoke-Expression")
	_, _ = fmt.Fprintln(a.stderr, "  source <(mynamr completion bash)")
}

func (a *App) printRuleUsage() error {
	_, err := fmt.Fprintln(a.stderr, "Usage:")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(a.stderr, "  mynamr rule list")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(a.stderr, "  mynamr rule add <name> --description <text>")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(a.stderr, "                 [--enabled=<true|false>] [--spec <text>] [--recognizer <text>] [--matcher <text>] [--selector <text>] [--renderer <text>]")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(a.stderr, "  mynamr rule remove <name>")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(a.stderr, "  mynamr rule show <name>")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(a.stderr, "  mynamr rule update <name> [--description <text>] [--enable|--disable]")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(a.stderr, "                    [--spec <text>|--spec-stdin] [--recognizer <text>] [--matcher <text>] [--selector <text>] [--renderer <text>]")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(a.stderr)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(a.stderr, "Sync editing with gdedit:")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(a.stderr, "  - Register once with: mynamr --sync gdedit")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(a.stderr, "  - Then run: mynamr rule update <name>")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(a.stderr, "  - mynamr launches gdedit automatically when no direct update flags are given")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(a.stderr, "  - Under the hood gdedit uses: rule show <name> --spec-only / rule update <name> --spec-stdin")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(a.stderr)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(a.stderr, "Examples:")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(a.stderr, "  mynamr rule list")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(a.stderr, "  mynamr rule add sample_rule --description \"User-defined sample rule\"")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(a.stderr, "  mynamr rule add script_rule --description \"Stored rule\" --recognizer \"detect token\" --renderer \"{value}\"")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(a.stderr, "  mynamr rule show name_alias_bundle")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(a.stderr, "  mynamr rule remove sample_rule")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(a.stderr, "  mynamr rule update name_alias_bundle --description \"Stored registry rule\"")
	if err != nil {
		return err
	}
	return err
}

func (a *App) printStatus(forcedRule string, resolvedConfig config.Resolved) (int, error) {
	status := "scaffold"
	if forcedRule != "" {
		status = status + ", forced rule=" + forcedRule
	}
	_, err := fmt.Fprintf(a.stdout, "mynamr status: %s\nconfig: %s\ndb: %s\n", status, resolvedConfig.ConfigPath, resolvedConfig.DBPath)
	return 0, err
}

func (a *App) resolveInput(args []string, useClipboard bool) (string, error) {
	if useClipboard {
		text, err := a.clip.ReadText()
		if err != nil {
			return "", err
		}
		text = strings.TrimSpace(text)
		if text == "" {
			return "", errors.New("clipboard does not contain usable text")
		}
		return text, nil
	}
	if len(args) > 0 {
		return strings.Join(args, " "), nil
	}

	reader := bufio.NewReader(a.in)
	line, err := reader.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}

	line = strings.TrimSpace(line)
	if line == "" {
		return "", errors.New("no input provided")
	}
	return line, nil
}

func (a *App) runRuleCommand(store *registry.Store, resolvedConfig config.Resolved, args []string) (int, error) {
	if len(args) == 0 {
		if err := a.printRuleUsage(); err != nil {
			return 1, err
		}
		return 0, nil
	}

	if args[0] == "--help" || args[0] == "-h" || args[0] == "help" {
		if err := a.printRuleUsage(); err != nil {
			return 1, err
		}
		return 0, nil
	}

	switch args[0] {
	case "add":
		return a.runRuleAddCommand(store, args[1:])
	case "list":
		return a.printRuleList(store)
	case "remove":
		return a.runRuleRemoveCommand(store, args[1:])
	case "show":
		return a.runRuleShowCommand(store, args[1:])
	case "update":
		return a.runRuleUpdateCommand(store, resolvedConfig, args[1:])
	default:
		return 1, fmt.Errorf("unknown rule subcommand %q", args[0])
	}
}

func (a *App) runRuleShowCommand(store *registry.Store, args []string) (int, error) {
	if len(args) == 0 {
		return 1, errors.New("rule show requires a rule name")
	}

	name := ""
	parseArgs := args
	if !strings.HasPrefix(args[0], "-") {
		name = args[0]
		parseArgs = args[1:]
	}

	fs := flag.NewFlagSet("rule show", flag.ContinueOnError)
	fs.SetOutput(a.stderr)
	specOnly := fs.Bool("spec-only", false, "print only the raw rule spec")
	if err := fs.Parse(parseArgs); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0, nil
		}
		return 2, err
	}

	remaining := fs.Args()
	if name == "" {
		if len(remaining) != 1 {
			return 1, errors.New("rule show requires a rule name")
		}
		name = remaining[0]
		remaining = remaining[1:]
	}
	if len(remaining) != 0 {
		return 1, errors.New("rule show requires exactly one rule name")
	}

	rule, err := store.Rule(name)
	if err != nil {
		return 1, err
	}

	if *specOnly {
		_, err = fmt.Fprint(a.stdout, rule.Spec)
		return 0, err
	}

	_, err = fmt.Fprintf(
		a.stdout,
		"name: %s\nenabled: %t\nsource: %s\ndescription: %s\nspec:\n%s\nrecognizer: %s\nmatcher: %s\nselector: %s\nrenderer: %s\n",
		rule.Name,
		rule.Enabled,
		rule.Source,
		rule.Description,
		indentBlock(rule.Spec, "  "),
		rule.Recognizer,
		rule.Matcher,
		rule.Selector,
		rule.Renderer,
	)
	return 0, err
}

func (a *App) runRuleAddCommand(store *registry.Store, args []string) (int, error) {
	if len(args) == 0 {
		return 1, errors.New("rule add requires exactly one rule name")
	}

	name := ""
	parseArgs := args
	if !strings.HasPrefix(args[0], "-") {
		name = args[0]
		parseArgs = args[1:]
	}

	fs := flag.NewFlagSet("rule add", flag.ContinueOnError)
	fs.SetOutput(a.stderr)
	description := fs.String("description", "", "description for the user rule")
	enabled := fs.Bool("enabled", true, "whether the user rule starts enabled")
	spec := fs.String("spec", "", "stored executable rule spec")
	recognizer := fs.String("recognizer", "", "stored recognizer definition")
	matcher := fs.String("matcher", "", "stored matcher definition")
	selector := fs.String("selector", "", "stored selector definition")
	renderer := fs.String("renderer", "", "stored renderer definition")
	if err := fs.Parse(parseArgs); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0, nil
		}
		return 2, err
	}

	remaining := fs.Args()
	if name == "" {
		if len(remaining) != 1 {
			return 1, errors.New("rule add requires exactly one rule name")
		}
		name = remaining[0]
		remaining = remaining[1:]
	}
	if len(remaining) != 0 {
		return 1, errors.New("rule add requires exactly one rule name")
	}
	if strings.TrimSpace(*description) == "" {
		return 1, errors.New("rule description is required")
	}

	rule := rules.RuleSpec{
		Name:        name,
		Description: strings.TrimSpace(*description),
		Enabled:     *enabled,
		Source:      "user",
		Spec:        rules.SyncSpecEnabled(*spec, *enabled),
		Recognizer:  *recognizer,
		Matcher:     *matcher,
		Selector:    *selector,
		Renderer:    *renderer,
	}
	if err := store.AddRule(rule); err != nil {
		return 1, err
	}

	_, err := fmt.Fprintf(a.stdout, "added rule: %s\n", rule.Name)
	return 0, err
}

func (a *App) runRuleRemoveCommand(store *registry.Store, args []string) (int, error) {
	if len(args) != 1 {
		return 1, errors.New("rule remove requires exactly one rule name")
	}
	if err := store.RemoveRule(args[0]); err != nil {
		return 1, err
	}
	_, err := fmt.Fprintf(a.stdout, "removed rule: %s\n", args[0])
	return 0, err
}

func (a *App) runRuleUpdateCommand(store *registry.Store, resolvedConfig config.Resolved, args []string) (int, error) {
	if len(args) == 0 {
		if err := a.printRuleUpdateUsage(""); err != nil {
			return 1, err
		}
		return 1, errors.New("rule update requires exactly one rule name")
	}

	name := ""
	parseArgs := args
	if !strings.HasPrefix(args[0], "-") {
		name = args[0]
		parseArgs = args[1:]
	}

	fs := flag.NewFlagSet("rule update", flag.ContinueOnError)
	fs.SetOutput(a.stderr)
	description := fs.String("description", "", "updated description for the stored rule")
	enable := fs.Bool("enable", false, "mark the rule enabled")
	disable := fs.Bool("disable", false, "mark the rule disabled")
	spec := fs.String("spec", "", "updated executable rule spec")
	specStdin := fs.Bool("spec-stdin", false, "read the updated rule spec from stdin")
	recognizer := fs.String("recognizer", "", "updated recognizer definition")
	matcher := fs.String("matcher", "", "updated matcher definition")
	selector := fs.String("selector", "", "updated selector definition")
	renderer := fs.String("renderer", "", "updated renderer definition")
	if err := fs.Parse(parseArgs); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0, nil
		}
		return 2, err
	}

	remaining := fs.Args()
	if name == "" {
		if len(remaining) != 1 {
			if err := a.printRuleUpdateUsage(""); err != nil {
				return 1, err
			}
			return 1, errors.New("rule update requires exactly one rule name")
		}
		name = remaining[0]
		remaining = remaining[1:]
	}
	if len(remaining) != 0 {
		if err := a.printRuleUpdateUsage(name); err != nil {
			return 1, err
		}
		return 1, errors.New("rule update requires exactly one rule name")
	}
	if *enable && *disable {
		if err := a.printRuleUpdateUsage(name); err != nil {
			return 1, err
		}
		return 1, errors.New("rule update cannot set --enable and --disable together")
	}

	currentRule, err := store.Rule(name)
	if err != nil {
		return 1, err
	}

	visited := make(map[string]bool)
	fs.Visit(func(f *flag.Flag) {
		visited[f.Name] = true
	})
	if visited["spec"] && *specStdin {
		return 1, errors.New("rule update cannot use --spec and --spec-stdin together")
	}

	var descriptionValue *string
	if visited["description"] {
		trimmedDescription := strings.TrimSpace(*description)
		descriptionValue = &trimmedDescription
	}

	var enabledValue *bool
	if *enable {
		value := true
		enabledValue = &value
	}
	if *disable {
		value := false
		enabledValue = &value
	}

	var specValue *string
	if *specStdin {
		data, err := io.ReadAll(a.in)
		if err != nil {
			return 1, err
		}
		if strings.TrimSpace(string(data)) == "" {
			return 1, errors.New("rule update --spec-stdin requires non-empty input")
		}
		synced := rules.SyncSpecEnabled(string(data), effectiveEnabled(currentRule.Enabled, enabledValue))
		if err := rules.ValidateSpec(synced); err != nil {
			return 1, err
		}
		specValue = &synced
	} else if visited["spec"] {
		synced := rules.SyncSpecEnabled(*spec, effectiveEnabled(currentRule.Enabled, enabledValue))
		if err := rules.ValidateSpec(synced); err != nil {
			return 1, err
		}
		specValue = &synced
	} else if enabledValue != nil {
		synced := rules.SyncSpecEnabled(currentRule.Spec, *enabledValue)
		specValue = &synced
	}

	var recognizerValue *string
	if visited["recognizer"] {
		recognizerValue = recognizer
	}

	var matcherValue *string
	if visited["matcher"] {
		matcherValue = matcher
	}

	var selectorValue *string
	if visited["selector"] {
		selectorValue = selector
	}

	var rendererValue *string
	if visited["renderer"] {
		rendererValue = renderer
	}
	if descriptionValue == nil && enabledValue == nil && specValue == nil && recognizerValue == nil && matcherValue == nil && selectorValue == nil && rendererValue == nil {
		if resolvedConfig.SyncEditor != "" {
			return a.launchSyncedEditor(resolvedConfig.SyncEditor, name)
		}
		if err := a.printRuleUpdateUsage(name); err != nil {
			return 1, err
		}
		return 1, errors.New("rule update requires at least one of --description, --enable, --disable, --spec, --spec-stdin, --recognizer, --matcher, --selector, or --renderer")
	}

	if err := store.UpdateRule(name, descriptionValue, enabledValue, specValue, recognizerValue, matcherValue, selectorValue, rendererValue); err != nil {
		return 1, err
	}

	_, err = fmt.Fprintf(a.stdout, "updated rule: %s\n", name)
	return 0, err
}

func effectiveEnabled(current bool, update *bool) bool {
	if update != nil {
		return *update
	}
	return current
}

func indentBlock(value, prefix string) string {
	trimmed := strings.TrimRight(value, "\n")
	if trimmed == "" {
		return prefix
	}
	lines := strings.Split(trimmed, "\n")
	for i, line := range lines {
		lines[i] = prefix + line
	}
	return strings.Join(lines, "\n")
}

func (a *App) printRuleUpdateUsage(name string) error {
	target := "<name>"
	if name != "" {
		target = name
	}
	_, err := fmt.Fprintln(a.stderr, "Usage:")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(a.stderr, "  mynamr rule update %s [--description <text>] [--enable|--disable]\n", target)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(a.stderr, "                         [--spec <text>|--spec-stdin] [--recognizer <text>] [--matcher <text>] [--selector <text>] [--renderer <text>]")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(a.stderr)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(a.stderr, "At least one update flag is required.")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(a.stderr, "Example: mynamr rule update %s --description \"Stored registry rule\"\n", target)
	return err
}

func (a *App) runSyncRegistration(editor string) (int, error) {
	switch editor {
	case "gdedit":
		exePath, err := a.exe()
		if err != nil {
			return 1, err
		}
		readCmd := formatEditorCommand(exePath, "rule", "show", "{name}", "--spec-only")
		writeCmd := formatEditorCommand(exePath, "rule", "update", "{name}", "--spec-stdin")
		stdoutMirror := &bytes.Buffer{}
		stderrMirror := &bytes.Buffer{}
		reusedExisting := false
		if err := a.runner.Run("gdedit", []string{"--sync-register", "mynamr", "--read", readCmd, "--write", writeCmd}, a.in, stdoutMirror, stderrMirror); err != nil {
			combined := stdoutMirror.String() + "\n" + stderrMirror.String()
			if !isIgnorableSyncRegistrationConflict(combined) {
				if stdoutMirror.Len() > 0 {
					_, _ = io.Copy(a.stdout, stdoutMirror)
				}
				if stderrMirror.Len() > 0 {
					_, _ = io.Copy(a.stderr, stderrMirror)
				}
				return 1, err
			}
			reusedExisting = true
		} else {
			if stdoutMirror.Len() > 0 {
				_, _ = io.Copy(a.stdout, stdoutMirror)
			}
			if stderrMirror.Len() > 0 {
				_, _ = io.Copy(a.stderr, stderrMirror)
			}
		}
		resolved, err := config.SetSyncEditor(editor)
		if err != nil {
			return 1, err
		}
		if reusedExisting {
			_, err = fmt.Fprintf(a.stdout, "synced editor already registered: %s\nconfig: %s\n", resolved.SyncEditor, resolved.ConfigPath)
			return 0, err
		}
		_, err = fmt.Fprintf(a.stdout, "synced editor registered: %s\nconfig: %s\n", resolved.SyncEditor, resolved.ConfigPath)
		return 0, err
	default:
		return 1, fmt.Errorf("unsupported sync editor %q", editor)
	}
}

func isIgnorableSyncRegistrationConflict(output string) bool {
	normalized := strings.ToLower(output)
	return strings.Contains(normalized, "already exists") && strings.Contains(normalized, "different formats")
}

func (a *App) launchSyncedEditor(editor, ruleName string) (int, error) {
	switch editor {
	case "gdedit":
		if err := a.runner.Run("gdedit", []string{"--sync", "mynamr", ruleName}, a.in, a.stdout, a.stderr); err != nil {
			return 1, err
		}
		return 0, nil
	default:
		return 1, fmt.Errorf("unsupported sync editor %q", editor)
	}
}

func formatEditorCommand(exePath string, args ...string) string {
	parts := []string{quoteCommandPart(exePath)}
	for _, arg := range args {
		parts = append(parts, quoteCommandPart(arg))
	}
	return strings.Join(parts, " ")
}

func quoteCommandPart(value string) string {
	if value == "{name}" {
		return value
	}
	clean := filepath.Clean(value)
	if strings.ContainsAny(clean, " {}\t") || strings.Contains(clean, "\\") || strings.Contains(clean, "/") {
		return strconv.Quote(clean)
	}
	return clean
}
