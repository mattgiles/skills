# CLI Reference

## Root Command

```text
skills
```

Short description:

```text
Manage local agent skill sources
```

Global flags:

| Flag | Meaning |
| --- | --- |
| `--verbose` | Show detailed diagnostic output |
| `-h`, `--help` | Show help |

## Command Tree

```text
skills
├── config
│   └── init
├── source
│   ├── add <alias> <git-url>
│   ├── list
│   └── sync [alias...]
├── skill
│   └── list [--source <alias>]
└── project
    ├── init
    ├── status
    ├── sync [--dry-run]
    └── update [source...] [--dry-run] [--sync]
```

## `skills config init`

Creates the default global config file if it does not already exist.

Behavior:

- prints `created config: <path>` on first creation
- prints `config already exists: <path>` if the file is already present

## `skills source add <alias> <git-url>`

Registers a source under an alias in the global config.

Arguments:

| Argument | Meaning |
| --- | --- |
| `alias` | Local source alias |
| `git-url` | Git remote URL or local Git repo path |

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

Arguments:

| Argument | Meaning |
| --- | --- |
| `alias...` | Optional subset of configured sources |

Verbose-only columns:

| Column | Meaning |
| --- | --- |
| `ACTION` | `cloned` or `fetched` |
| `ALIAS` | Source alias |
| `REMOTE` | Default remote ref and commit |
| `LOCAL` | Local HEAD ref and commit |
| `PATH` | Canonical repo path |
| `URL` | Registered source URL |

## `skills skill list`

Lists discovered skills from synced source repos.

Flags:

| Flag | Meaning |
| --- | --- |
| `--source <alias>` | Only list skills from the named source |

Default columns:

| Column | Meaning |
| --- | --- |
| `SOURCE` | Source alias |
| `NAME` | Skill directory name |
| `PATH` | Relative path within the repo |

Verbose-only columns:

| Column | Meaning |
| --- | --- |
| `ABS_PATH` | Absolute discovered path |

## `skills project init`

Creates `.skills.yaml` in the current directory if it does not already exist.

Behavior:

- prints `created manifest: <path>` on first creation
- prints `manifest already exists: <path>` if the file already exists

## `skills project status`

Shows project source state, managed link state, and stale managed links.

Default output:

- a source table
- a link table
- a `STALE_PATH` table when stale managed links exist

Verbose mode also prints `SOURCES`, `LINKS`, and `STALE` headings.

Verbose source columns:

| Column | Meaning |
| --- | --- |
| `SOURCE` | Source alias |
| `STATUS` | Source status |
| `REF` | Declared ref |
| `COMMIT` | Desired or current short commit |
| `STORED` | Stored short commit from project state |
| `REPO_PATH` | Canonical repo path |
| `WORKTREE_PATH` | Desired worktree path |
| `MESSAGE` | Extra detail |

Verbose link columns:

| Column | Meaning |
| --- | --- |
| `AGENT` | Agent name |
| `SOURCE` | Source alias |
| `SKILL` | Skill name |
| `STATUS` | Link status |
| `PATH` | Destination symlink path |
| `TARGET` | Desired target path |
| `MESSAGE` | Extra detail |

## `skills project sync`

Syncs declared project skills into target agent directories.

Flags:

| Flag | Meaning |
| --- | --- |
| `--dry-run` | Preview sync actions without changing state or links |

Output:

- prints `dry-run` first when previewing
- prints a source table
- prints a link table
- prints a `PRUNED_PATH` table when stale managed links are removed or would be removed

Verbose mode also prints `SOURCES`, `LINKS`, and `PRUNED` headings.

## `skills project update [source...]`

Resolves newer commits for project sources.

Arguments:

| Argument | Meaning |
| --- | --- |
| `source...` | Optional subset of declared project sources |

Flags:

| Flag | Meaning |
| --- | --- |
| `--dry-run` | Preview update actions without changing state or links |
| `--sync` | Run `project sync` after updating source state |

Output:

- prints `dry-run` first when previewing
- prints a source table
- when `--sync` is used, prints a blank line and then the normal `project sync` output
