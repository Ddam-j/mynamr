# mynamr

`mynamr`는 입력된 문자열을 기계적으로 분석하여, 내장된 규칙에 따라 재구성된 결과 문자열을 반환하는 CLI-first 텍스트 리네이밍 도구입니다.

## Development bootstrap

```bash
go test ./...
go run ./cmd/mynamr --version
go run ./cmd/mynamr rule list
go run ./cmd/mynamr --status
```

## Rule registry

- Rules are now persisted in SQLite through the internal registry layer
- The app bootstraps `~/.config/mynamr/config.json`
- The config file stores the registry DB path as `db_path`
- The same config file stores `sync_editor` when editor sync is registered
- The default registry DB lives at `~/.config/mynamr/rules.db`
- Each stored rule can carry `recognizer`, `matcher`, `selector`, and `renderer` definition fields
- Inspect them with `mynamr rule show <name>`
- Write them with `mynamr rule add ... --recognizer ... --matcher ... --selector ... --renderer ...`

## Sync with gdedit

`mynamr` owns the sync registration and launch flow. Users do not need to run `gdedit --sync-register ...` manually once `mynamr` is built.

The intended experience is simple:
- register `gdedit` once from `mynamr`
- keep using `mynamr` as the user-facing command
- let `mynamr` open `gdedit` and persist edited specs back into the registry

After sync is configured, the normal edit entry point is still `mynamr rule update <name>`, not a raw `gdedit` command.

### One-time registration

PowerShell:

```powershell
.\mynamr.exe --sync gdedit
```

This does three things:
- registers the `mynamr` sync target in `gdedit`
- stores `sync_editor: "gdedit"` in `~/.config/mynamr/config.json`
- remembers that future rule edits should open through `gdedit`

If the sync target already exists, `mynamr` keeps the existing registration and reports that it is already registered instead of failing hard.

Typical output looks like one of these:

```text
synced editor registered: gdedit
config: C:\Users\<user>\.config\mynamr\config.json
```

or, if the sync target was already present:

```text
synced editor already registered: gdedit
config: C:\Users\<user>\.config\mynamr\config.json
```

What `mynamr` persists is the editor choice itself:

```json
{
  "db_path": "C:\\Users\\<user>\\.config\\mynamr\\rules.db",
  "sync_editor": "gdedit"
}
```

### Daily editing flow

```powershell
.\mynamr.exe rule update catalog_code_title
```

When no direct update flags are provided, `mynamr` launches:

```text
gdedit --sync mynamr <rule-name>
```

This is the recommended day-to-day editing path. If sync is registered, you normally do not need to call `gdedit --sync ...` yourself.

Under the hood, `gdedit` reads and writes through these `mynamr` commands:

```text
mynamr rule show <name> --spec-only
mynamr rule update <name> --spec-stdin
```

That means:
- `gdedit` does not need direct SQLite access
- `mynamr` remains the owner of config lookup, validation, and DB updates
- multiline YAML specs are edited over stdio without temp files

The actual open/save flow is:
1. you run `mynamr rule update <name>`
2. `mynamr` launches `gdedit --sync mynamr <name>`
3. `gdedit` reads the raw spec through `mynamr rule show <name> --spec-only`
4. you edit the YAML spec in `gdedit`
5. `gdedit` saves through `mynamr rule update <name> --spec-stdin`

This separation is intentional:
- `gdedit` provides the editing UI
- `mynamr` owns validation, DB path discovery, migrations, and persistence

### What opens gdedit and what does not

These commands open `gdedit` after sync is registered:

```powershell
.\mynamr.exe rule update catalog_code_title
.\mynamr.exe rule update name_alias_bundle
```

These commands do not open `gdedit`; they update directly from the CLI:

```powershell
.\mynamr.exe rule update catalog_code_title --description "new description"
.\mynamr.exe rule update catalog_code_title --enable
.\mynamr.exe rule update catalog_code_title --disable
.\mynamr.exe rule update catalog_code_title --spec-stdin
```

If `gdedit` did not open, first check whether you passed one of the direct update flags above.

### Direct CLI editing still works

You can still update specs without `gdedit`:

```powershell
$spec = @"
name: catalog_code_title
enabled: true
description: leading catalog code and title are recomposed into a bracketed title.
compose:
  template: "[{selected.code[0]}] {selected.title[0]}"
"@

$spec | .\mynamr.exe rule update catalog_code_title --spec-stdin
```

`--spec-stdin` is the same save path used by `gdedit`, so it is the best manual debugging tool for sync behavior.

Important rules for this save path:
- `--spec-stdin` requires non-empty input
- malformed YAML is rejected before save
- missing `compose.template` is rejected before save
- `--spec` and `--spec-stdin` cannot be used together

If you want to see the exact stored executable spec without labels, use:

```powershell
.\mynamr.exe rule show catalog_code_title --spec-only
```

That output is the same raw text that `gdedit` reads.

### Sync scope and limitations

The current sync contract edits only the raw executable `spec` text.

It does not currently provide a structured editor flow for:
- `description`
- `source`
- `recognizer`
- `matcher`
- `selector`
- `renderer`

Those fields are still visible in `mynamr rule show <name>`, but the `gdedit` sync bridge is intentionally limited to `spec`.

Also note that `enabled:` in YAML is normalized by `mynamr` to the rule's current enabled state. If you want to change enablement, use CLI flags such as:

```powershell
.\mynamr.exe rule update catalog_code_title --enable
.\mynamr.exe rule update catalog_code_title --disable
```

### Troubleshooting

- `mynamr` is not recognized in PowerShell
  - Use `.\mynamr.exe ...` unless the binary is on your `PATH`.

- `gdedit` does not open on `rule update <name>`
  - Re-run `.\mynamr.exe --sync gdedit`
  - Check that `sync_editor` exists in `~/.config/mynamr/config.json`
  - Check that you did not pass direct update flags

- save fails from `gdedit`
  - Inspect the current stored spec with `rule show <name> --spec-only`
  - Validate that the edited YAML still has a `compose.template`
  - Remember that syntactically valid YAML can still describe behavior the current engine does not support yet

- PowerShell examples with `&` or parentheses break
  - Quote the whole argument as one string:

```powershell
.\mynamr.exe "코요이 코난 최신작 & 프로필 (Koyoi Konan, 小宵こなん (こよいこなん))"
```

### PowerShell note

Use `.\mynamr.exe`, not bare `mynamr`, unless the binary is on your `PATH`.

## Shell completion

```bash
source <(mynamr completion bash)
```

- Add that line to your shell profile to keep it across new sessions

```powershell
mynamr completion powershell | Out-String | Invoke-Expression
```

- PowerShell registration above is session-scoped unless you add it to your `$PROFILE`

- Completion suggests commands, flags, and rule names from the registry

## Release flow

- Release automation is configured with `.goreleaser.yaml`
- GitHub tag releases are handled by `.github/workflows/release.yml`
- Local tag helpers live in `scripts/release-tag.sh` and `scripts/release-tag.ps1`
- Detailed release instructions are in `docs/release.md`
