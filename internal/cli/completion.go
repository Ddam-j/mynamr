package cli

import (
	"fmt"
	"strings"

	"github.com/Ddam-j/mynamr/internal/registry"
)

var rootFlags = []string{
	"--clip",
	"--help",
	"--json",
	"--list-rules",
	"--outclip",
	"--rule",
	"--status",
	"--version",
	"-h",
}

var rootCommands = []string{
	"completion",
	"rule",
}

var ruleCommands = []string{
	"add",
	"list",
	"remove",
	"show",
	"update",
	"help",
	"--help",
	"-h",
}

var ruleUpdateFlags = []string{
	"--description",
	"--disable",
	"--enable",
	"--spec",
	"--spec-stdin",
	"--matcher",
	"--recognizer",
	"--renderer",
	"--selector",
}

var ruleAddFlags = []string{
	"--description",
	"--enabled",
	"--spec",
	"--matcher",
	"--recognizer",
	"--renderer",
	"--selector",
}

var completionShells = []string{
	"bash",
	"powershell",
}

func (a *App) runCompletionCommand(args []string) (int, error) {
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" || args[0] == "help" {
		if err := a.printCompletionUsage(); err != nil {
			return 1, err
		}
		return 0, nil
	}

	if len(args) != 1 {
		return 1, fmt.Errorf("completion requires exactly one shell name")
	}

	script, err := completionScript(args[0])
	if err != nil {
		return 1, err
	}

	_, err = fmt.Fprint(a.stdout, script)
	if err != nil {
		return 1, err
	}
	return 0, nil
}

func (a *App) printCompletionUsage() error {
	_, err := fmt.Fprintln(a.stderr, "Usage:")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(a.stderr, "  mynamr completion bash")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(a.stderr, "  mynamr completion powershell")
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
	_, err = fmt.Fprintln(a.stderr, "  source <(mynamr completion bash)")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(a.stderr, "  mynamr completion powershell | Out-String | Invoke-Expression")
	return err
}

func (a *App) runInternalCompletion(store *registry.Store, args []string) (int, error) {
	items, err := a.complete(store, args)
	if err != nil {
		return 1, err
	}

	for _, item := range items {
		if _, err := fmt.Fprintln(a.stdout, item); err != nil {
			return 1, err
		}
	}
	return 0, nil
}

func (a *App) complete(store *registry.Store, args []string) ([]string, error) {
	words := append([]string(nil), args...)
	prefix := ""
	if len(words) > 0 {
		prefix = words[len(words)-1]
	}

	if len(words) >= 2 && words[len(words)-2] == "--rule" {
		ruleNames, err := a.ruleNames(store)
		if err != nil {
			return nil, err
		}
		return filterByPrefix(ruleNames, prefix), nil
	}

	if len(words) == 0 {
		return append(rootFlags, rootCommands...), nil
	}

	if len(words) == 1 {
		return filterByPrefix(append(rootFlags, rootCommands...), prefix), nil
	}

	switch words[0] {
	case "rule":
		if len(words) == 2 {
			return filterByPrefix(ruleCommands, prefix), nil
		}
		if words[1] == "add" {
			if len(words) >= 4 {
				return filterByPrefix(ruleAddFlags, prefix), nil
			}
		}
		if words[1] == "update" {
			if len(words) == 3 {
				ruleNames, err := a.ruleNames(store)
				if err != nil {
					return nil, err
				}
				return filterByPrefix(ruleNames, prefix), nil
			}
			if len(words) >= 4 {
				return filterByPrefix(ruleUpdateFlags, prefix), nil
			}
		}
		if words[1] == "show" || words[1] == "remove" || words[1] == "update" {
			if len(words) > 3 {
				return nil, nil
			}
			ruleNames, err := a.ruleNames(store)
			if err != nil {
				return nil, err
			}
			return filterByPrefix(ruleNames, prefix), nil
		}
		return nil, nil
	case "completion":
		if len(words) > 2 {
			return nil, nil
		}
		return filterByPrefix(completionShells, prefix), nil
	default:
		return filterByPrefix(rootFlags, prefix), nil
	}
}

func (a *App) ruleNames(store *registry.Store) ([]string, error) {
	rules, err := store.ListRules()
	if err != nil {
		return nil, err
	}

	names := make([]string, 0, len(rules))
	for _, rule := range rules {
		names = append(names, rule.Name)
	}
	return names, nil
}

func filterByPrefix(items []string, prefix string) []string {
	filtered := make([]string, 0, len(items))
	for _, item := range items {
		if strings.HasPrefix(item, prefix) {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func completionScript(shell string) (string, error) {
	switch shell {
	case "bash":
		return bashCompletionScript(), nil
	case "powershell":
		return powershellCompletionScript(), nil
	default:
		return "", fmt.Errorf("unsupported completion shell %q", shell)
	}
}

func bashCompletionScript() string {
	return strings.Join([]string{
		"_mynamr_completion() {",
		"  local suggestions",
		"  local args=()",
		"  COMPREPLY=()",
		"  local i",
		"  for ((i=1; i<${#COMP_WORDS[@]}; i++)); do",
		"    args+=(\"${COMP_WORDS[i]}\")",
		"  done",
		"  if (( COMP_CWORD >= ${#COMP_WORDS[@]} )); then",
		"    args+=(\"\")",
		"  fi",
		"  suggestions=\"$(${COMP_WORDS[0]} __complete \"${args[@]}\")\"",
		"  while IFS= read -r line; do",
		"    [[ -z \"$line\" ]] && continue",
		"    COMPREPLY+=(\"$line\")",
		"  done <<< \"$suggestions\"",
		"}",
		"complete -o default -F _mynamr_completion mynamr",
		"",
	}, "\n")
}

func powershellCompletionScript() string {
	return strings.Join([]string{
		"Register-ArgumentCompleter -Native -CommandName @('mynamr', 'mynamr.exe', '.\\mynamr.exe') -ScriptBlock {",
		"    param($wordToComplete, $commandAst, $cursorPosition)",
		"    $commandName = $commandAst.CommandElements[0].Extent.Text",
		"    $parts = @()",
		"    foreach ($element in $commandAst.CommandElements | Select-Object -Skip 1) {",
		"        $parts += $element.Extent.Text",
		"    }",
		"    if ($parts.Count -eq 0 -or $parts[$parts.Count - 1] -ne $wordToComplete) {",
		"        $parts += $wordToComplete",
		"    }",
		"    foreach ($item in (& $commandName __complete @parts)) {",
		"        [System.Management.Automation.CompletionResult]::new($item, $item, 'ParameterValue', $item)",
		"    }",
		"}",
		"",
	}, "\n")
}
