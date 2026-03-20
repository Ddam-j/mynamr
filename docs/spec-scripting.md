# Spec Scripting Guide

This guide documents the executable YAML rule language that `mynamr` supports today.

The runtime executes the `spec` field stored in the rule registry. It does **not** execute the separate metadata fields `recognizer`, `matcher`, `selector`, or `renderer` shown by `rule show`.

## Top-level structure

Supported top-level keys:

- `name`
- `enabled`
- `description`
- `detect`
- `drop_words`
- `prefix_drop_words`
- `ignore_symbols`
- `groups`
- `select`
- `compose`

Minimal valid spec:

```yaml
name: sample_rule
enabled: true
description: sample executable rule

detect:
  all:
    - has_leading_catalog_code

groups:
  code:
    sources:
      - input.leading_catalog_code

compose:
  template: "{selected.code[0]}"
```

Validation rules today:
- YAML must parse
- `compose.template` must exist

## `detect`

`detect` controls whether a rule matches the current input.

Supported keys:
- `all`
- `any`
- `none`
- `priority`

Supported detect conditions today:
- `has_parenthesized_text`
- `has_alias_separator`
- `has_leading_catalog_code`

Example:

```yaml
detect:
  all:
    - has_parenthesized_text
  any:
    - has_alias_separator
  none:
    - has_leading_catalog_code
  priority: 80
```

Notes:
- `priority` is used when multiple enabled rules match the same input
- higher `priority` wins
- unsupported detect tokens are not part of the documented executable language

## Cleanup fields

### `drop_words`

`drop_words` removes configured text from the cleaned value. Use it when broad replacement is acceptable.

Example:

```yaml
drop_words:
  - Latest
  - Profile
```

### `prefix_drop_words`

`prefix_drop_words` removes only leading configured text from the extracted value.

This is useful when you want to remove a front separator like `" - "` but keep inner `" - "` inside the title.

Example:

```yaml
prefix_drop_words:
  - " - "
```

### `ignore_symbols`

`ignore_symbols` removes configured symbols during cleanup.

Example:

```yaml
ignore_symbols:
  - "("
  - ")"
  - ","
```

## `groups`

`groups` declare how named values are extracted.

Example:

```yaml
groups:
  code:
    sources:
      - input.leading_catalog_code
  title:
    sources:
      - input.after_catalog_code
```

Supported source selectors today:
- `outer.leading_hangul_sequence`
- `paren.jp_sequence`
- `nested.jp_sequence`
- `paren.latin_sequence`
- `nested.latin_sequence`
- `input.leading_catalog_code`
- `input.after_catalog_code`

How they behave:
- `outer.leading_hangul_sequence` reads the outer text before parenthesized aliases
- `paren.jp_sequence` and `paren.latin_sequence` read the first matching Japanese or Latin sequence inside parenthesized content
- `nested.jp_sequence` and `nested.latin_sequence` read the next matching sequence inside parenthesized content
- `input.leading_catalog_code` reads the detected catalog code
- `input.after_catalog_code` reads the title text after the detected catalog code

## `select`

`select` declares group policies.

Example:

```yaml
select:
  groups:
    ko:
      policy: first
    jp:
      policy: first
    en:
      policy: first
```

Important note:
- `policy` is stored in the spec and should currently be treated as documentation of intent
- the current runtime always takes the first resolved value for each group

## `compose`

`compose.template` builds the final output string.

Supported placeholders today:
- `{selected.<group>[0]}`
- `{<group>}`

Examples:

```yaml
compose:
  template: "[{selected.code[0]}] {selected.title[0]}"
```

```yaml
compose:
  template: "{selected.ko[0]}.{selected.jp[0]}.{selected.en[0]}"
```

## Worked examples

### `catalog_code_title`

```yaml
name: catalog_code_title
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
```

### `name_alias_bundle`

```yaml
name: name_alias_bundle
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
```

## Current limitations

- The engine supports a narrow detect/source vocabulary; document and use only the tokens listed in this guide
- `recognizer`, `matcher`, `selector`, and `renderer` fields shown by `rule show` are stored metadata, not the executable runtime language
- `--json` exists as a CLI flag but is not implemented as a structured rule API
- Some future-looking DSL ideas in planning documents are not executable yet; this guide documents only the current runtime language
