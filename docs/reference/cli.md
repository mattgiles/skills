# CLI Reference

## Root Command

```text
skills
```

Global flags:

| Flag | Meaning |
| --- | --- |
| `--verbose` | Show detailed diagnostic output |
| `-h`, `--help` | Show help |

## Command Tree

```text
skills
├── init [--project|--global]
├── doctor [--global]
├── config
│   └── init
├── source
│   ├── add <alias> <git-url>
│   ├── list
│   └── sync [alias...]
├── skill
│   └── list [--source <alias>]
├── home
│   ├── init
│   ├── status
│   ├── sync [--dry-run]
│   └── update [source...] [--dry-run] [--sync]
├── version
└── project
    ├── init
    ├── status
    ├── sync [--dry-run]
    └── update [source...] [--dry-run] [--sync]
```

## `skills config init`

Creates the default global config file if it does not already exist.

## `skills doctor`

Runs a read-only diagnostic pass for the current project workspace by default.

Checks:

- `git` availability
- project manifest and state parsing
- project `.gitignore` ownership for managed runtime paths
- source readiness and ref resolution
- project-local cache health under `.agents/cache/`
- canonical `.agents/skills` link health
- Claude `.claude/skills` adapter health

Flags:

| Flag | Meaning |
| --- | --- |
| `--global` | Inspect global config and the shared home workspace instead of the current project |

## `skills init`

Initializes either repo-local project state or shared home/global state.

Flags:

| Flag | Meaning |
| --- | --- |
| `--project` | Initialize repo-local project state explicitly |
| `--global` | Initialize shared home/global state explicitly |

Behavior:

- inside a Git repo, `skills init` routes to repo-local initialization automatically when repo-local `skills` artifacts already exist
- inside a Git repo with no existing `skills` artifacts, `skills init` prompts on an interactive TTY
- in non-interactive contexts, pass `--project` or `--global`
- outside a Git repo, `skills init` requires explicit scope

## `skills source add <alias> <git-url>`

Registers a source under an alias in the global config.

## `skills source list`

Lists configured sources.

Default columns:

| Column | Meaning |
| --- | --- |
| `ALIAS` | Source alias |
| `STATUS` | Source state |
| `REMOTE` | Default remote ref and commit, when known |
| `LOCAL` | Local HEAD ref and commit, when known |

Verbose-only columns:

| Column | Meaning |
| --- | --- |
| `PATH` | Canonical local repo path |
| `URL` | Registered source URL |

## `skills source sync [alias...]`

Clones missing sources and fetches existing ones.

## `skills skill list`

Lists discovered skills from synced source repos.

Flags:

| Flag | Meaning |
| --- | --- |
| `--source <alias>` | Only list skills from the named source |

## `skills project init`

Creates a project-local standardized workspace:

- `.agents/manifest.yaml`
- `.agents/cache/repos/`
- `.agents/cache/worktrees/`
- `.agents/skills/`
- `.claude/skills/`
- ignore rules for:
  - `.agents/state.yaml`
  - `.agents/cache/`
  - `.agents/skills/`
  - `.claude/skills/`

If the project lives inside a Git repo, `skills project init` updates the repo-root `.gitignore`. It fails if those managed runtime paths already contain tracked Git content.

## `skills project status`

Shows:

- source resolution state
- canonical skill link state in `.agents/skills`
- Claude adapter link state in `.claude/skills`
- stale managed links for both sections

Sections:

- `SOURCES`
- `SKILLS`
- `CLAUDE`
- `STALE_SKILLS` when present
- `STALE_CLAUDE` when present

## `skills project sync`

Syncs the declared project skills into:

- canonical project links in `.agents/skills`
- Claude adapter links in `.claude/skills`
- using project-local clones and worktrees under `.agents/cache/`

Flags:

| Flag | Meaning |
| --- | --- |
| `--dry-run` | Preview sync actions without changing state or links |

Sections:

- `SOURCES`
- `SKILLS`
- `CLAUDE`
- `PRUNED_SKILLS` when present
- `PRUNED_CLAUDE` when present

## `skills project update [source...]`

Resolves newer commits for project sources and optionally runs `project sync`.

Flags:

| Flag | Meaning |
| --- | --- |
| `--dry-run` | Preview update actions without changing state or links |
| `--sync` | Run `project sync` after updating source state |

## `skills home init`

Creates the shared home manifest at `~/.agents/manifest.yaml` by default and ensures the shared canonical directories exist.

## `skills home status`

Shows source state, canonical shared-skill state in `~/.agents/skills`, and Claude adapter state in `~/.claude/skills`.

## `skills home sync`

Syncs the shared home manifest into:

- `~/.agents/skills`
- `~/.claude/skills`

Flags:

| Flag | Meaning |
| --- | --- |
| `--dry-run` | Preview sync actions without changing state or links |

## `skills home update [source...]`

Resolves newer commits for shared home sources and optionally runs `home sync`.

## `skills version`

Prints build metadata for the installed binary.

Current fields:

- `version`
- `commit`
- `date`
- `go`
- `platform`
