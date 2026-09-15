# AGENTS

<!-- BEGIN perk managed -->
## perk conventions (managed by `perk init` — do not edit between these markers)

This repo is wired for the **perk** plan-oriented workflow on Pi.

- **`perk init` owns all Pi wiring and the `.perk/` dot-directory** — `.pi/settings.json`
  package entries, `.perk/config.toml`, `.gitignore` entries, this block. Re-run `perk init`
  to converge (idempotent); `perk doctor --fix` repairs oddities.
- **GitHub access goes through the `gh` CLI.** Never fetch `github.com` over raw HTTPS
  (curl/fetch) — private repos reject unauthenticated requests. Read-only `gh` query
  subcommands (view/list/diff/status/checks/search) work even in perk read-only sessions.
- **Prefer ast-grep for code search.** Structural/AST queries go through `ast-grep` (see the
  `ast-grep` skill). For literal text use the `grep`/`find` tools or `rg`/`fd` — they honor
  `.gitignore`. Recursive `grep -r…` and `find` without `-maxdepth` also walk `node_modules`,
  `.venv` and `.worktrees`; perk caps them at a 30s `timeout` unless the bash call passes its own.

perk version: 3.3.0
<!-- END perk managed -->
