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
- human-facing command output is rendered through `pterm`
- machine-readable output is intentionally deferred to a separate later feature

## Command Tree

```text
skills
├── add <source> <skill> [--url <git-url>] [--ref <ref>] [--path <dir>] [--global]
├── completion
│   ├── bash
│   ├── fish
│   ├── powershell
│   └── zsh
├── cache
│   └── clean [--global]
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
│   ├── add <alias> <git-url> [--ref <ref>] [--include <dir>]... [--exclude <dir>]... [--global]
│   ├── list [--global]
│   └── sync [alias...] [--global]
├── skill
│   └── list [--global] [--source <alias>] [--all]
└── version
```

## `skills add <source> <skill>`

Adds a skill to the active manifest and immediately runs sync for the same scope.

Behavior:

- if the source alias already exists, only the skill declaration is added
- if the source alias does not exist, `--url` is required; `--include`/`--exclude`
  scope flags are not accepted here — register the source with `skills source
  add` first when scoping is needed
- when creating a new source and `--ref` is omitted, `skills` infers the remote default branch
- if the `(source, skill)` pair is already declared and `--path` is omitted or
  equals the entry's current `path`, the command prints a no-op message and
  exits successfully
- if the `(source, skill)` pair is already declared and `--path` differs, the
  entry's `path:` is updated in place (comments preserved) and sync runs
- if sync fails after the manifest edit, the command restores the previous manifest bytes

`--path` selects a discovered skill by its repo-relative directory when several
share a name (see the `Path` column of `skills skill list`). The value is
normalized (`./`, trailing `/`, and a trailing `SKILL.md` are dropped) before it
is written. `<skill>` stays the link directory under `.agents/skills/`, so
`--path` also lets you install a skill under a different name.

Flags:

| Flag | Meaning |
| --- | --- |
| `--url <git-url>` | Source Git URL or local repo path for a new source |
| `--ref <ref>` | Source ref for a new source; defaults to the remote's default branch |
| `--path <dir>` | Repo-relative skill directory to select when multiple skills share a name (name stays the link directory) |
| `--global` | Operate on shared home/global installs |

## `skills completion`

Prints shell completion scripts for supported shells.

Subcommands:

- `skills completion bash`
- `skills completion fish`
- `skills completion powershell`
- `skills completion zsh`

Shell-specific help describes how to load the generated script for the current shell and how to install it persistently.

## `skills cache clean`

Deletes cached canonical source repos and worktrees for the active scope so the next `skills sync` re-downloads sources and rebuilds worktrees.

Behavior:

- repo mode uses the current repo's active cache backend from `.agents/local.yaml`
- in a repo configured with `cache.mode: global`, plain `skills cache clean` cleans the shared global cache
- `--global` explicitly targets the shared home/global cache
- the command does not edit manifests, state files, or installed skill symlinks

Flags:

| Flag | Meaning |
| --- | --- |
| `--global` | Clean the shared home/global cache roots |

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
- when cache mode is not configured yet and stdin/stdout are interactive, `skills init` prompts for `local` or `global`
- in non-interactive repo initialization, pass `--cache=<local|global>` the first time
- outside a Git repo, `skills init` requires `--global`

## `skills source add <alias> <git-url>`

Registers a source under an alias in the active manifest.

`--include` and `--exclude` are repeatable and set the source's discovery scope
(repo-relative directory prefixes; see [Project Manifest](project-manifest.md)).
When a flag is **not** passed, the manifest's current list for that field is
carried forward, so re-registering a source (for example to change its `ref`)
does not drop its scope. Passing the flag replaces the whole list. Clearing a
list is a manual manifest edit. Values are normalized and deduplicated before
writing; `--exclude .` is rejected.

Flags:

| Flag | Meaning |
| --- | --- |
| `--ref <ref>` | Source ref to store in the manifest; defaults to the remote's default branch |
| `--include <dir>` | Repo-relative directory to search for skills (repeatable; omit to keep the manifest's current list) |
| `--exclude <dir>` | Repo-relative directory to skip when discovering skills (repeatable; omit to keep the manifest's current list) |
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

By default it uses the current repo manifest sources. Use `--global` to inspect the shared home manifest instead. Discovery resolves each source's manifest ref against the fetched canonical repo state, so newly fetched upstream skills appear even if the local checkout `HEAD` has not moved.

The listing honors each source's `include`/`exclude` scope by default. `--all`
bypasses the scope and appends a `Scope` column (`included`/`excluded`) so you
can see what to include or exclude.

Columns: `Source`, `Name`, `Path` (repo-relative skill directory), then `Scope`
with `--all`, then `Abs Path` with `--verbose`.

Flags:

| Flag | Meaning |
| --- | --- |
| `--global` | List skills from shared global sources instead of the current repo |
| `--source <alias>` | Only list skills from the named source |
| `--all` | Ignore source include/exclude scope and list every discovered skill |

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
| `--update` | Advance stored source commits before syncing links |

Repo mode syncs:

- canonical repo links in `.agents/skills`
- Claude adapter links in `.claude/skills`
- using either:
  - repo-local clones and worktrees under `.agents/cache/` when `.agents/local.yaml` selects `local`
  - global clone and worktree roots from config when `.agents/local.yaml` selects `global`

Global mode syncs:

- `~/.agents/skills`
- `~/.claude/skills`

By default `skills sync` preserves the currently stored resolved commit for each source. Use `skills sync --update` when you want sync to both advance source state and update installed links in one step.

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
