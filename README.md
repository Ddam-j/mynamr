# mynamr

`mynamr` is a CLI-first renaming tool for people who keep seeing the same messy title patterns and want them to become clean, predictable names.

Instead of hand-editing every string, you teach `mynamr` a few executable rules and let it turn noisy input into a consistent output format.

One of its biggest advantages is that those executable rule specs are not trapped in flat files. `mynamr` stores them in its local registry and can open them directly in `gdedit`, so you can program the transformation logic for a specific DB-backed rule without exporting, editing, and importing temporary files.

## What it feels like

Give it a raw title like this:

```text
SNOS-084 - "여름 창가의 편지" 오래된 친구가 남긴 기록
```

And get back a normalized result like this:

```text
[SNOS-084] "여름 창가의 편지" 오래된 친구가 남긴 기록
```

Or turn alias-heavy text into a compact multilingual name bundle:

```text
하늘 정원 아카이브 최신작 & 프로필 (Sky Garden Archive, 青空庭園)
-> 하늘 정원 아카이브.青空庭園.Sky Garden Archive
```

## Common flows

### 1. Quick clipboard cleanup

You copy a messy title from a browser or metadata page, run one command, and paste the cleaned result somewhere else.

```powershell
.\mynamr.exe --clip --outclip
```

Example feeling:

```text
copied:  하늘 정원 아카이브 최신작 & 프로필 (Sky Garden Archive, 青空庭園)
result:  하늘 정원 아카이브.青空庭園.Sky Garden Archive
```

Payoff: no temp files, no manual copy-edit loop, just copy -> transform -> paste.

### 2. Let rules choose, or override them

If your input matches a stored rule, `mynamr` can pick it automatically:

```bash
mynamr "SNOS-084 - Summer Window Letter"
```

If you already know which rule should apply, force it explicitly:

```bash
mynamr --rule catalog_code_title "SNOS-084 - Summer Window Letter"
```

Payoff: use auto-detect for the fast path, and keep `--rule` for the moments when you want full control.

### 3. Edit the rule itself, not an exported file

Register `gdedit` once:

```powershell
.\mynamr.exe --sync gdedit
```

Then open a stored rule directly:

```powershell
.\mynamr.exe rule update catalog_code_title
```

`mynamr` launches `gdedit`, `gdedit` edits the raw YAML spec over stdio, and `mynamr` validates and saves it back into the registry.

Payoff: you modify the actual DB-backed transformation logic in place instead of exporting, editing, and importing temporary files.

## Quick start

```bash
go test ./...
go run ./cmd/mynamr --version
go run ./cmd/mynamr rule list
go run ./cmd/mynamr "SNOS-084 - Summer Window Letter"
```

## Built-in examples

- `catalog_code_title` — extracts a leading catalog code and rebuilds a bracketed title
- `name_alias_bundle` — extracts ko/jp/en aliases from outer text and parenthesized alias bundles

## Learn more

- `docs/usage.md` — day-to-day CLI usage, clipboard flow, rule lifecycle, and `gdedit` sync workflow
- `docs/spec-scripting.md` — executable YAML spec reference for the currently supported rule language
- `docs/release.md` — release and tagging flow

## Notes

- Rules are persisted locally in SQLite at `~/.config/mynamr/rules.db`
- `mynamr` can register and launch `gdedit` through `mynamr --sync gdedit`
- On PowerShell, use `.\mynamr.exe` unless `mynamr` is already on your `PATH`
