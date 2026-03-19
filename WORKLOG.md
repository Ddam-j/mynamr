# WORKLOG

- Updated: 2026-03-19 14:46 KST
- OpenCode Session ID: `ses_2fb61dd72ffen9ywR8t5JWY4Sx`
- Project: `mynamr`
- Current status: Go module bootstrap, CLI scaffold, GoReleaser config, and release helper scripts are in place

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

## MVP target

- CLI input handling
- `--clip` and `--outclip`
- automatic rule matching
- builtin rules: `name_alias_bundle`, `catalog_code_title`
- template rendering
- `rule list` / `rule show`
- SQLite-backed registry

## Recommended next steps

1. Implement real clipboard adapters for `--clip` and `--outclip`
2. Build the recognizer, matcher, selector, and renderer flow
3. Add builtin rule loading beyond static metadata and wire SQLite registry support
4. Expand CLI coverage for `rule enable`, `rule disable`, and `rule add/remove`
5. Cut the first tagged release with `scripts/release-tag.sh` or `scripts/release-tag.ps1`

## Resume note

When continuing work in OpenCode, use the saved Session ID above as the primary reference for session history and transcript recovery.
