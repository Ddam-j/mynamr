package cli

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/Ddam-j/mynamr/internal/rules"
	"github.com/Ddam-j/mynamr/internal/version"
)

type App struct {
	in     io.Reader
	stdout io.Writer
	stderr io.Writer
}

func NewApp(in io.Reader, stdout, stderr io.Writer) *App {
	return &App{in: in, stdout: stdout, stderr: stderr}
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

	useClipboard := fs.Bool("clip", false, "read input from the clipboard")
	writeClipboard := fs.Bool("outclip", false, "write output to the clipboard")
	showJSON := fs.Bool("json", false, "print structured JSON output")
	showStatus := fs.Bool("status", false, "print status information")
	forcedRule := fs.String("rule", "", "force a specific rule")
	listRules := fs.Bool("list-rules", false, "list registered rules")
	showVersion := fs.Bool("version", false, "print build version")

	if err := fs.Parse(args); err != nil {
		return 2, err
	}

	remaining := fs.Args()
	if len(remaining) > 0 && remaining[0] == "rule" {
		return a.runRuleCommand(remaining[1:])
	}

	switch {
	case *showVersion:
		_, err := fmt.Fprintln(a.stdout, version.Detailed())
		return 0, err
	case *listRules:
		return a.printRuleList()
	case *useClipboard:
		return 1, errors.New("--clip is scaffolded but clipboard input is not implemented yet")
	case *writeClipboard:
		return 1, errors.New("--outclip is scaffolded but clipboard output is not implemented yet")
	case *showJSON:
		return 1, errors.New("--json is scaffolded but JSON status output is not implemented yet")
	case *showStatus:
		return a.printStatus(*forcedRule)
	}

	input, err := a.resolveInput(remaining)
	if err != nil {
		return 1, err
	}

	_, err = fmt.Fprintln(a.stdout, input)
	return 0, err
}

func (a *App) printRuleList() (int, error) {
	for _, name := range rules.ListBuiltinNames() {
		if _, err := fmt.Fprintln(a.stdout, name); err != nil {
			return 1, err
		}
	}
	return 0, nil
}

func (a *App) printStatus(forcedRule string) (int, error) {
	status := "scaffold"
	if forcedRule != "" {
		status = status + ", forced rule=" + forcedRule
	}
	_, err := fmt.Fprintf(a.stdout, "mynamr status: %s\n", status)
	return 0, err
}

func (a *App) resolveInput(args []string) (string, error) {
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

func (a *App) runRuleCommand(args []string) (int, error) {
	if len(args) == 0 {
		return 1, errors.New("rule subcommand required: list or show <name>")
	}

	switch args[0] {
	case "list":
		return a.printRuleList()
	case "show":
		if len(args) < 2 {
			return 1, errors.New("rule show requires a rule name")
		}

		rule, err := rules.Builtin(args[1])
		if err != nil {
			return 1, err
		}

		_, err = fmt.Fprintf(a.stdout, "name: %s\nenabled: %t\ndescription: %s\n", rule.Name, rule.Enabled, rule.Description)
		return 0, err
	default:
		return 1, fmt.Errorf("unknown rule subcommand %q", args[0])
	}
}
