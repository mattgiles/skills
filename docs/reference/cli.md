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

## Exit Codes

`skills` uses a small stable exit-code policy:

| Code | Meaning |
| --- | --- |
| `0` | Success |
| `2` | Usage or argument error |
| `3` | `skills doctor` found problems |
| `1` | Other runtime failure |

Notes:

- `--help` exits with `0`
- command output and diagnostics still follow the existing stdout/stderr split
- machine-readable output is intentionally deferred to a separate later feature

## Command Tree

```text
skills
├── init [--global] [--cache local|global]
├── status [--global]
├── sync [--global] [--dry-run]
├── update [source...] [--global] [--dry-run] [--sync]
├── doctor [--global]
├── self
│   └── update [--version <tag>]
├── config
│   └── init
├── source
│   ├── add <alias> <git-url> [--ref <ref>] [--global]
│   ├── list [--global]
│   └── sync [alias...] [--global]
├── skill
│   └── list [--global] [--source <alias>]
└── version
```

## `skills config init`

Creates the default global config file if it does not already exist.

## `skills self update`

Downloads the latest published macOS release and replaces the currently running `skills` binary.

Flags:

| Flag | Meaning |
| --- | --- |
| `--version <tag>` | Install a specific release version instead of the latest one |

## `skills doctor`

Runs a read-only diagnostic pass for the current project workspace by default.

Checks:

- `git` availability
- project manifest and state parsing
- project local settings parsing in `.agents/local.yaml`
- project `.gitignore` ownership for managed runtime paths
- source readiness and ref resolution
- active project cache mode and cache-root health
- canonical `.agents/skills` link health
- Claude `.claude/skills` adapter health

Flags:

| Flag | Meaning |
| --- | --- |
| `--global` | Inspect global config and the shared home workspace instead of the current project |

## `skills init`

Initializes repo-local state by default, or shared home/global state with `--global`.

Flags:

| Flag | Meaning |
| --- | --- |
| `--global` | Initialize shared home/global state explicitly |
| `--cache <local\|global>` | Choose the cache backend for project mode |

Behavior:

- inside a Git repo, `skills init` initializes or repairs repo-local state
- project mode records the chosen cache backend in untracked `.agents/local.yaml`
- in non-interactive repo initialization, pass `--cache=<local|global>` the first time
- outside a Git repo, `skills init` requires `--global`

## `skills source add <alias> <git-url>`

Registers a source under an alias in the active manifest.

Flags:

| Flag | Meaning |
| --- | --- |
| `--ref <ref>` | Source ref to store in the manifest; defaults to the remote's default branch |
| `--global` | Write to the shared home manifest instead of the current repo manifest |

## `skills source list`

Lists sources declared in the active manifest.

Flags:

| Flag | Meaning |
| --- | --- |
| `--global` | List sources from the shared home manifest instead of the current repo manifest |

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

Clones missing sources and fetches existing ones from the active manifest.

Flags:

| Flag | Meaning |
| --- | --- |
| `--global` | Sync sources from the shared home manifest instead of the current repo manifest |

## `skills skill list`

Lists discovered skills from synced source repos.

By default it uses the current repo manifest sources. Use `--global` to inspect the shared home manifest instead.

Flags:

| Flag | Meaning |
| --- | --- |
| `--global` | List skills from shared global sources instead of the current repo |
| `--source <alias>` | Only list skills from the named source |

## `skills status`

Shows installed skill status for the current repo by default.

Flags:

| Flag | Meaning |
| --- | --- |
| `--global` | Show status for shared home/global installs instead of the current repo |

Repo mode shows:

- source resolution state
- canonical skill link state in `.agents/skills`
- Claude adapter link state in `.claude/skills`
- stale managed links for both sections

Global mode shows the same sections for `~/.agents/skills` and `~/.claude/skills`.

## `skills sync`

Enforces the declared skills state for the current repo by default.

Flags:

| Flag | Meaning |
| --- | --- |
| `--global` | Sync shared home/global installs instead of the current repo |
| `--dry-run` | Preview sync actions without changing state or links |

Repo mode syncs:

- canonical repo links in `.agents/skills`
- Claude adapter links in `.claude/skills`
- using either:
  - repo-local clones and worktrees under `.agents/cache/` when `.agents/local.yaml` selects `local`
  - global clone and worktree roots from config when `.agents/local.yaml` selects `global`

Global mode syncs:

- `~/.agents/skills`
- `~/.claude/skills`

## `skills update [source...]`

Resolves newer commits for the current repo by default and optionally runs `sync`.

Flags:

| Flag | Meaning |
| --- | --- |
| `--global` | Update shared home/global installs instead of the current repo |
| `--dry-run` | Preview update actions without changing state or links |
| `--sync` | Run `sync` after updating source state |

## Repo-Local Initialization Details

`skills init` in a repo creates a project-local standardized workspace:

Creates a project-local standardized workspace:

- `.agents/manifest.yaml`
- `.agents/local.yaml`
- `.agents/skills/`
- `.claude/skills/`
- `.agents/cache/repos/` and `.agents/cache/worktrees/` when `--cache=local`
- ignore rules for:
  - `.agents/state.yaml`
  - `.agents/local.yaml`
  - `.agents/cache/`
  - `.agents/skills/`
  - `.claude/skills/`

Flags:

| Flag | Meaning |
| --- | --- |
| `--cache <local\|global>` | Choose the cache backend for this repo user |

If the project lives inside a Git repo, `skills init` updates the repo-root `.gitignore`. It fails if those managed runtime paths already contain tracked Git content.

## Shared Home Initialization Details

`skills init --global` creates the shared home manifest at `~/.agents/manifest.yaml` by default and ensures the shared canonical directories exist.

## `skills version`

Prints build metadata for the installed binary.

Current fields:

- `version`
- `commit`
- `date`
- `go`
- `platform`
