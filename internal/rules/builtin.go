package rules

import (
	"fmt"
	"regexp"
	"sort"
)

var ruleNamePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]*$`)

type RuleSpec struct {
	Name        string
	Description string
	Enabled     bool
	Source      string
	Spec        string
	Recognizer  string
	Matcher     string
	Selector    string
	Renderer    string
}

var builtins = map[string]RuleSpec{
	"catalog_code_title": {
		Name:        "catalog_code_title",
		Description: "Wrap a leading catalog code in brackets and append the remaining title text.",
		Enabled:     true,
		Source:      "seeded",
		Spec: `name: catalog_code_title
enabled: true
description: leading catalog code and title are recomposed into a bracketed title.

detect:
  all:
    - has_leading_catalog_code
  priority: 70

prefix_drop_words:
  - " - "

groups:
  code:
    sources:
      - input.leading_catalog_code
  title:
    sources:
      - input.after_catalog_code

select:
  groups:
    code:
      policy: first
    title:
      policy: first

compose:
  template: "[{selected.code[0]}] {selected.title[0]}"
`,
		Recognizer: "pattern: ^(?P<code>[A-Za-z0-9_-]+)[\\s_-]+(?P<title>.+)$",
		Matcher:    "require captures: code, title",
		Selector:   "use first capture set from recognizer result",
		Renderer:   "[{code}] {title}",
	},
	"name_alias_bundle": {
		Name:        "name_alias_bundle",
		Description: "Compose representative ko/jp/en aliases from the outer text and parenthesized aliases.",
		Enabled:     true,
		Source:      "seeded",
		Spec: `name: name_alias_bundle
enabled: true
description: 바깥 대표 한글명과 괄호 안 별칭에서 ko/jp/en을 선택해 조립한다.

detect:
  all:
    - has_parenthesized_text
  any:
    - has_alias_separator
  none:
    - has_leading_catalog_code
  priority: 80

drop_words:
  - 최신작
  - 프로필
  - "&"

ignore_symbols:
  - "("
  - ")"
  - ","

groups:
  ko:
    sources:
      - outer.leading_hangul_sequence

  jp:
    sources:
      - paren.jp_sequence
      - nested.jp_sequence

  en:
    sources:
      - paren.latin_sequence
      - nested.latin_sequence

select:
  groups:
    ko:
      policy: first
    jp:
      policy: first
    en:
      policy: first

compose:
  template: "{selected.ko[0]}.{selected.jp[0]}.{selected.en[0]}"
`,
		Recognizer: "split primary text and parenthesized alias bundle: primary + (en_alias, jp_alias)",
		Matcher:    "require primary text plus at least one alias token",
		Selector:   "map primary->ko, latin alias->en, kanji/kana alias->jp when present",
		Renderer:   "{ko} | {en} | {jp}",
	},
}

func Builtins() []RuleSpec {
	names := ListBuiltinNames()
	result := make([]RuleSpec, 0, len(names))
	for _, name := range names {
		result = append(result, builtins[name])
	}
	return result
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

func ValidateRuleName(name string) error {
	if !ruleNamePattern.MatchString(name) {
		return fmt.Errorf("invalid rule name %q", name)
	}
	return nil
}
