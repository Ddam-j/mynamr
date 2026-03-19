package rules

import (
	"fmt"
	"sort"
)

type RuleSpec struct {
	Name        string
	Description string
	Enabled     bool
}

var builtins = map[string]RuleSpec{
	"catalog_code_title": {
		Name:        "catalog_code_title",
		Description: "Wrap a leading catalog code in brackets and append the remaining title text.",
		Enabled:     true,
	},
	"name_alias_bundle": {
		Name:        "name_alias_bundle",
		Description: "Compose representative ko/jp/en aliases from the outer text and parenthesized aliases.",
		Enabled:     true,
	},
}

func ListBuiltinNames() []string {
	names := make([]string, 0, len(builtins))
	for name := range builtins {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func Builtin(name string) (RuleSpec, error) {
	rule, ok := builtins[name]
	if !ok {
		return RuleSpec{}, fmt.Errorf("unknown rule %q", name)
	}
	return rule, nil
}
