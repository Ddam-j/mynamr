package rules

import (
	"fmt"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

type detectSpec struct {
	All      []string `yaml:"all"`
	Any      []string `yaml:"any"`
	None     []string `yaml:"none"`
	Priority int      `yaml:"priority"`
}

type groupSpec struct {
	Sources []string `yaml:"sources"`
}

type selectGroupSpec struct {
	Policy string `yaml:"policy"`
}

type selectSpec struct {
	Groups map[string]selectGroupSpec `yaml:"groups"`
}

type composeSpec struct {
	Template string `yaml:"template"`
}

type executableSpec struct {
	Name          string               `yaml:"name"`
	Enabled       bool                 `yaml:"enabled"`
	Description   string               `yaml:"description"`
	Detect        detectSpec           `yaml:"detect"`
	DropWords     []string             `yaml:"drop_words"`
	PrefixDrop    []string             `yaml:"prefix_drop_words"`
	IgnoreSymbols []string             `yaml:"ignore_symbols"`
	Groups        map[string]groupSpec `yaml:"groups"`
	Select        selectSpec           `yaml:"select"`
	Compose       composeSpec          `yaml:"compose"`
}

type MatchResult struct {
	Matched  bool
	Priority int
	Output   string
}

func SelectAndExecute(ruleSet []RuleSpec, input string) (string, bool, error) {
	bestPriority := -1
	bestOutput := ""
	matched := false

	for _, rule := range ruleSet {
		if strings.TrimSpace(rule.Spec) == "" {
			continue
		}
		result, err := Match(rule, input)
		if err != nil {
			return "", false, err
		}
		if !result.Matched {
			continue
		}
		if !matched || result.Priority > bestPriority {
			matched = true
			bestPriority = result.Priority
			bestOutput = result.Output
		}
	}

	if !matched {
		return "", false, nil
	}
	return bestOutput, true, nil
}

var (
	latinSequencePattern  = regexp.MustCompile(`[A-Za-z]+(?:[ -][A-Za-z]+)*`)
	parenthesizedPattern  = regexp.MustCompile(`\(([^()]*)\)`)
	leadingCatalogPattern = regexp.MustCompile(`^([A-Za-z]{2,10}-?\d{2,}[A-Za-z0-9-]*)(?:\s+-\s+|[\s_]+)(.+)$`)
	hangulWordsPattern    = regexp.MustCompile(`[가-힣]+(?:\s+[가-힣]+)*`)
	japaneseWordsPattern  = regexp.MustCompile(`[ぁ-ゖァ-ヺ一-龯]+(?:\s*[ぁ-ゖァ-ヺ一-龯]+)*`)
)

func Execute(rule RuleSpec, input string) (string, error) {
	if !rule.Enabled {
		return "", fmt.Errorf("rule %q is disabled", rule.Name)
	}
	if strings.TrimSpace(rule.Spec) == "" {
		return "", fmt.Errorf("rule %q has no executable spec", rule.Name)
	}

	var spec executableSpec
	if err := yaml.Unmarshal([]byte(rule.Spec), &spec); err != nil {
		return "", fmt.Errorf("parse rule spec %q: %w", rule.Name, err)
	}

	ctx := buildExecutionContext(input, spec)
	if !ctx.matches(spec.Detect) {
		return "", fmt.Errorf("rule %q did not match the provided input", rule.Name)
	}

	return executeParsedSpec(rule.Name, spec, ctx)
}

func Match(rule RuleSpec, input string) (MatchResult, error) {
	if !rule.Enabled {
		return MatchResult{}, nil
	}
	if strings.TrimSpace(rule.Spec) == "" {
		return MatchResult{}, fmt.Errorf("rule %q has no executable spec", rule.Name)
	}

	var spec executableSpec
	if err := yaml.Unmarshal([]byte(rule.Spec), &spec); err != nil {
		return MatchResult{}, fmt.Errorf("parse rule spec %q: %w", rule.Name, err)
	}

	ctx := buildExecutionContext(input, spec)
	if !ctx.matches(spec.Detect) {
		return MatchResult{Matched: false, Priority: spec.Detect.Priority}, nil
	}

	output, err := executeParsedSpec(rule.Name, spec, ctx)
	if err != nil {
		return MatchResult{}, err
	}
	return MatchResult{Matched: true, Priority: spec.Detect.Priority, Output: output}, nil
}

func executeParsedSpec(ruleName string, spec executableSpec, ctx executionContext) (string, error) {

	selected := make(map[string]string)
	for groupName, group := range spec.Groups {
		values := ctx.valuesForSources(group.Sources)
		if len(values) == 0 {
			return "", fmt.Errorf("rule %q could not resolve group %q", ruleName, groupName)
		}
		selected[groupName] = values[0]
	}

	result := spec.Compose.Template
	for groupName, value := range selected {
		result = strings.ReplaceAll(result, fmt.Sprintf("{selected.%s[0]}", groupName), value)
		result = strings.ReplaceAll(result, fmt.Sprintf("{%s}", groupName), value)
	}

	if strings.Contains(result, "{") {
		return "", fmt.Errorf("rule %q left unresolved compose placeholders", ruleName)
	}
	return strings.TrimSpace(result), nil
}

func SyncSpecEnabled(spec string, enabled bool) string {
	trimmed := strings.TrimSpace(spec)
	if trimmed == "" {
		return spec
	}

	lines := strings.Split(spec, "\n")
	for i, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "enabled:") {
			indent := line[:len(line)-len(strings.TrimLeft(line, " "))]
			lines[i] = fmt.Sprintf("%senabled: %t", indent, enabled)
			return strings.Join(lines, "\n")
		}
	}

	return fmt.Sprintf("enabled: %t\n%s", enabled, spec)
}

func ValidateSpec(spec string) error {
	trimmed := strings.TrimSpace(spec)
	if trimmed == "" {
		return nil
	}

	var parsed executableSpec
	if err := yaml.Unmarshal([]byte(spec), &parsed); err != nil {
		return fmt.Errorf("parse rule spec: %w", err)
	}
	if strings.TrimSpace(parsed.Compose.Template) == "" {
		return fmt.Errorf("parse rule spec: missing compose.template")
	}
	return nil
}

type executionContext struct {
	input         string
	outer         string
	parenContents []string
	dropWords     []string
	prefixDrop    []string
	ignoreSymbols []string
	leadingCode   string
	afterCode     string
}

func buildExecutionContext(input string, spec executableSpec) executionContext {
	ctx := executionContext{
		input:         strings.TrimSpace(input),
		dropWords:     spec.DropWords,
		prefixDrop:    spec.PrefixDrop,
		ignoreSymbols: spec.IgnoreSymbols,
	}

	matches := parenthesizedPattern.FindAllStringSubmatch(input, -1)
	if first := strings.Index(input, "("); first >= 0 {
		if last := strings.LastIndex(input, ")"); last > first {
			ctx.parenContents = append(ctx.parenContents, strings.TrimSpace(input[first+1:last]))
		}
	}
	for _, match := range matches {
		content := strings.TrimSpace(match[1])
		if content == "" {
			continue
		}
		alreadyPresent := false
		for _, existing := range ctx.parenContents {
			if existing == content {
				alreadyPresent = true
				break
			}
		}
		if !alreadyPresent {
			ctx.parenContents = append(ctx.parenContents, content)
		}
	}
	ctx.outer = input
	if idx := strings.Index(input, "("); idx >= 0 {
		ctx.outer = strings.TrimSpace(input[:idx])
	}
	if catalogMatch := leadingCatalogPattern.FindStringSubmatch(strings.TrimSpace(input)); len(catalogMatch) == 3 {
		ctx.leadingCode = catalogMatch[1]
		ctx.afterCode = strings.TrimSpace(catalogMatch[2])
	}
	return ctx
}

func (c executionContext) matches(d detectSpec) bool {
	for _, cond := range d.All {
		if !c.check(cond) {
			return false
		}
	}
	if len(d.Any) > 0 {
		ok := false
		for _, cond := range d.Any {
			if c.check(cond) {
				ok = true
				break
			}
		}
		if !ok {
			return false
		}
	}
	for _, cond := range d.None {
		if c.check(cond) {
			return false
		}
	}
	return true
}

func (c executionContext) check(cond string) bool {
	switch cond {
	case "has_parenthesized_text":
		return len(c.parenContents) > 0
	case "has_alias_separator":
		for _, item := range c.parenContents {
			if strings.Contains(item, ",") {
				return true
			}
		}
		return false
	case "has_leading_catalog_code":
		return c.leadingCode != ""
	default:
		return false
	}
}

func (c executionContext) valuesForSources(sources []string) []string {
	values := make([]string, 0)
	for _, source := range sources {
		if value := strings.TrimSpace(c.valueForSource(source)); value != "" {
			values = append(values, value)
		}
	}
	return values
}

func (c executionContext) valueForSource(source string) string {
	switch source {
	case "outer.leading_hangul_sequence":
		return firstMatch(hangulWordsPattern, c.cleanedOuter())
	case "paren.jp_sequence":
		return firstMatch(japaneseWordsPattern, strings.Join(c.parenContents, " "))
	case "nested.jp_sequence":
		return nthMatch(japaneseWordsPattern, strings.Join(c.parenContents, " "), 1)
	case "paren.latin_sequence":
		return firstMatch(latinSequencePattern, strings.Join(c.parenContents, " "))
	case "nested.latin_sequence":
		return nthMatch(latinSequencePattern, strings.Join(c.parenContents, " "), 1)
	case "input.leading_catalog_code":
		return c.leadingCode
	case "input.after_catalog_code":
		return c.cleanedAfterCode()
	default:
		return ""
	}
}

func (c executionContext) cleanedOuter() string {
	return c.clean(c.outer)
}

func (c executionContext) cleanedAfterCode() string {
	return c.cleanLeading(c.afterCode)
}

func (c executionContext) clean(value string) string {
	for _, word := range c.dropWords {
		value = strings.ReplaceAll(value, word, " ")
	}
	for _, symbol := range c.ignoreSymbols {
		value = strings.ReplaceAll(value, symbol, " ")
	}
	return strings.Join(strings.Fields(value), " ")
}

func (c executionContext) cleanLeading(value string) string {
	value = strings.TrimSpace(value)
	for {
		changed := false
		for _, word := range c.prefixDrop {
			token := strings.TrimSpace(word)
			if token == "" {
				continue
			}
			if strings.HasPrefix(value, token) {
				value = strings.TrimSpace(strings.TrimPrefix(value, token))
				changed = true
			}
		}
		for _, word := range c.dropWords {
			token := strings.TrimSpace(word)
			if token == "" {
				continue
			}
			if strings.HasPrefix(value, token) {
				value = strings.TrimSpace(strings.TrimPrefix(value, token))
				changed = true
			}
		}
		for _, symbol := range c.ignoreSymbols {
			token := strings.TrimSpace(symbol)
			if token == "" {
				continue
			}
			if strings.HasPrefix(value, token) {
				value = strings.TrimSpace(strings.TrimPrefix(value, token))
				changed = true
			}
		}
		if !changed {
			break
		}
	}
	return strings.Join(strings.Fields(value), " ")
}

func firstMatch(pattern *regexp.Regexp, input string) string {
	return nthMatch(pattern, input, 0)
}

func nthMatch(pattern *regexp.Regexp, input string, index int) string {
	matches := pattern.FindAllString(input, -1)
	if index >= len(matches) {
		return ""
	}
	return strings.TrimSpace(matches[index])
}
