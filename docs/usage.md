# Usage Guide

## What mynamr does

`mynamr` reads text, selects a matching stored rule, and returns a transformed result. It can work from command-line arguments, standard input, or the clipboard on Windows.

## Common workflows

### Quick clipboard cleanup

When you already have the source text copied, clipboard mode is the shortest path:

```powershell
.\mynamr.exe --clip --outclip
```

Typical flow:
1. copy a noisy title from a browser or metadata page
2. run `mynamr --clip --outclip`
3. paste the normalized result wherever you need it

Example:

```text
copied:  하늘 정원 아카이브 최신작 & 프로필 (Sky Garden Archive, 青空庭園)
result:  하늘 정원 아카이브.青空庭園.Sky Garden Archive
```

### Auto-detect for the fast path

If your input clearly matches a stored rule, you can just pass the text:

```bash
mynamr "SNOS-084 - Summer Window Letter"
```

`mynamr` checks enabled rules, evaluates their `detect` conditions, and applies the highest-priority match.

### Forced rule when you want certainty

If multiple rules are possible or you already know the correct one, force it:

```bash
mynamr --rule catalog_code_title "SNOS-084 - Summer Window Letter"
```

This is the best way to test a rule while you are refining it.

### Edit the stored rule itself

If the output is close but not quite right, edit the rule instead of repeatedly compensating by hand:

```powershell
.\mynamr.exe --sync gdedit
.\mynamr.exe rule update catalog_code_title
```

That opens the stored spec in `gdedit`, lets you change the YAML, and saves it back through `mynamr`.

## Input modes

### Positional text

```bash
mynamr "SNOS-084 - Summer Window Letter"
```

### Standard input

```bash
printf 'SNOS-084 - Summer Window Letter\n' | mynamr
```

### Clipboard input and output (Windows)

```powershell
.\mynamr.exe --clip --outclip
```

- `--clip` reads the current clipboard text
- `--outclip` writes the transformed result back to the clipboard
- stdout still prints the result

## Output modes

### Auto-detect mode

```bash
mynamr "SNOS-084 - Summer Window Letter"
```

`mynamr` loads enabled stored rules, checks their `detect` conditions, and applies the highest-priority match.

If nothing matches, it returns the original input unchanged.

### Forced rule mode

```bash
mynamr --rule catalog_code_title "SNOS-084 - Summer Window Letter"
```

Use this when you want to bypass auto-detection and run a specific rule directly.

## Basic commands

### List rules

```bash
mynamr rule list
```

### Show a rule with labels

```bash
mynamr rule show catalog_code_title
```

### Show only the raw executable spec

```bash
mynamr rule show catalog_code_title --spec-only
```

This is the raw YAML spec text stored in the registry.

### Add a rule

```bash
mynamr rule add sample_rule --description "User-defined sample rule"
```

You can also add executable fields directly:

```bash
mynamr rule add sample_rule \
  --description "User-defined sample rule" \
  --spec "name: sample_rule\nenabled: true\ncompose:\n  template: '{selected.value[0]}'"
```

### Update a rule directly

```bash
mynamr rule update catalog_code_title --description "new description"
mynamr rule update catalog_code_title --enable
mynamr rule update catalog_code_title --disable
```

### Update a rule from stdin

```powershell
$spec = @"
name: catalog_code_title
enabled: true
description: updated over stdin
compose:
  template: "[{selected.code[0]}] {selected.title[0]}"
"@

$spec | .\mynamr.exe rule update catalog_code_title --spec-stdin
```

Rules for `--spec-stdin`:
- input must be non-empty
- YAML must parse successfully
- `compose.template` must exist
- `--spec` and `--spec-stdin` cannot be used together

### Remove a rule

```bash
mynamr rule remove sample_rule
```

## Rule editing with gdedit

## One-time registration

```powershell
.\mynamr.exe --sync gdedit
```

This registers `gdedit` as the editor sync target and stores that choice in `~/.config/mynamr/config.json`.

Possible messages:

```text
synced editor registered: gdedit
config: C:\Users\<user>\.config\mynamr\config.json
```

or:

```text
synced editor already registered: gdedit
config: C:\Users\<user>\.config\mynamr\config.json
```

## Daily editing flow

```powershell
.\mynamr.exe rule update catalog_code_title
```

If no direct update flags are given, `mynamr` launches:

```text
gdedit --sync mynamr <rule-name>
```

The bridge is:

```text
mynamr rule show <name> --spec-only
mynamr rule update <name> --spec-stdin
```

So:
- `gdedit` provides the editing UI
- `mynamr` owns config discovery, validation, and DB persistence
- `gdedit` does not need direct SQLite access

## What opens gdedit and what does not

These open `gdedit` when sync is configured:

```powershell
.\mynamr.exe rule update catalog_code_title
.\mynamr.exe rule update name_alias_bundle
```

These do not open `gdedit`; they update directly:

```powershell
.\mynamr.exe rule update catalog_code_title --description "new description"
.\mynamr.exe rule update catalog_code_title --enable
.\mynamr.exe rule update catalog_code_title --disable
.\mynamr.exe rule update catalog_code_title --spec-stdin
```

## Completion

### Bash

```bash
source <(mynamr completion bash)
```

### PowerShell

```powershell
.\mynamr.exe completion powershell | Out-String | Invoke-Expression
```

Completion supports commands, flags, and rule names from the registry.

## PowerShell notes

- Use `.\mynamr.exe`, not bare `mynamr`, unless the binary is on `PATH`
- Quote any single argument containing `&` or parentheses:

```powershell
.\mynamr.exe "하늘 정원 아카이브 최신작 & 프로필 (Sky Garden Archive, 青空庭園)"
```

## Troubleshooting

### `mynamr` is not recognized

Use `.\mynamr.exe ...` or add the binary directory to `PATH`.

### `gdedit` does not open on `rule update <name>`

- Run `.\mynamr.exe --sync gdedit` again
- Check that `sync_editor` is present in `~/.config/mynamr/config.json`
- Confirm that you did not pass direct update flags

### Save fails from gdedit

- Inspect the current stored spec with `rule show <name> --spec-only`
- Confirm the YAML still includes `compose.template`
- Remember that syntactically valid YAML can still describe behavior the current engine does not support yet
