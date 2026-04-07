# Output And Status Reference

This page documents the current user-visible status vocabulary.

## Source Statuses

| Status | Meaning |
| --- | --- |
| `missing-source` | Canonical source repo is not cloned |
| `invalid-source` | Canonical source path is not a usable Git repo |
| `invalid-ref` | The declared ref could not be resolved |
| `not-synced` | No resolved commit is stored yet |
| `update-available` | The stored state does not match the currently resolved ref |
| `up-to-date` | Stored state matches the currently resolved ref |
| `resolved` | Sync resolved a source with no stored commit |
| `updated` | Sync or update moved the stored commit forward |
| `inspect-failed` | Commit or worktree inspection failed |

## Canonical Skill Link Statuses

| Status | Meaning |
| --- | --- |
| `missing` | Destination path does not exist |
| `linked` | Destination symlink already points at the desired target |
| `created` | Sync created a new symlink |
| `updated` | Sync replaced a managed symlink to point at a newer target |
| `would-create` | Dry-run would create the symlink |
| `would-update` | Dry-run would replace the symlink |
| `stale` | Existing managed symlink points at an older managed target |
| `conflict` | Destination exists as a non-symlink or unmanaged symlink |
| `invalid` | Destination path could not be inspected |
| `unknown-source` | Skill references a source that is not declared |
| `source-not-ready` | Source has no usable desired commit yet |
| `missing-skill` | No discovered skill matched the declared name |
| `ambiguous-skill` | More than one discovered skill matched the declared name |
| `inspect-failed` | Source inspection failed before link resolution |

## Claude Adapter Statuses

Claude adapter links use the same symlink lifecycle statuses as canonical skill links, but target the canonical `.agents/skills/<skill-name>` paths instead of worktree directories.

## Output Sections

Status and sync commands use:

- `SOURCES`
- `SKILLS`
- `CLAUDE`

Status may also include:

- `STALE_SKILLS`
- `STALE_CLAUDE`

Sync may also include:

- `PRUNED_SKILLS`
- `PRUNED_CLAUDE`

Dry-run commands print:

```text
dry-run
```

before their tables.
