# WORKLOG

- Updated: 2026-03-19 14:46 KST
- OpenCode Session ID: `ses_2fb61dd72ffen9ywR8t5JWY4Sx`
- Project: `mynamr`
- Current status: Go module bootstrap, DB-backed rule registry, executable rule specs, clipboard I/O, gdedit sync flow, GoReleaser config, and release helper scripts are in place

## Continuation context

- The project guides in `dev-guide/` were reviewed before starting development.
- Primary workflow rules come from `dev-guide/COWORK_COMMIT_GUIDE.md`.
- Product scope and architecture come from `dev-guide/mynamr_project_plan_and_work_order.md`.

## Rules to follow

- Use Conventional Commits: `type(scope): summary`
- Keep commits atomic and PR scope small
- Explain `why` clearly in PR/body context
- Build a deterministic Go CLI-first tool
- Do not use AI runtime logic or external script-language dependencies
- Base recognition only on the input string itself
- Preserve the core pipeline: `input -> recognizer -> matcher -> selector -> renderer -> output`
- Persist rule registry state via the config-managed SQLite path
- Let `mynamr` own editor sync registration and spec save/load over stdio

## MVP target

- CLI input handling
- `--clip` and `--outclip`
- automatic rule matching
- builtin rules: `name_alias_bundle`, `catalog_code_title`
- template rendering
- `rule list` / `rule show`
- SQLite-backed registry with config-managed DB path
- gdedit sync registration and editor-driven `rule update <name>` flow

## Recommended next steps

1. Expand the executable rule engine beyond the currently supported detect/source vocabulary
2. Add richer rule validation so unsupported DSL tokens fail before save
3. Decide whether builtin RuleSpec seeds stay in code or move to embedded spec assets
4. Polish `--help` and docs around sync, clip, and spec editing workflows
5. Cut the first tagged release with `scripts/release-tag.sh` or `scripts/release-tag.ps1`

## Resume note

When continuing work in OpenCode, use the saved Session ID above as the primary reference for session history and transcript recovery.
