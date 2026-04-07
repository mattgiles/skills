# Output And Status Reference

This page documents the current user-visible status vocabulary.

## Source Statuses In `source list`

| Status | Meaning |
| --- | --- |
| `missing` | Source is configured but the canonical repo path does not exist |
| `cloned` | Local repo exists, but no default remote commit was reported |
| `synced` | Local repo exists and default remote metadata is available |
| `invalid` | Path exists but is not a usable Git repo |
| `invalid: ...` | Same as `invalid`, with the underlying error text |

## Source Statuses In `project status`

| Status | Meaning |
| --- | --- |
| `missing-source` | Canonical source repo is not cloned |
| `invalid-source` | Canonical source path is not a usable Git repo |
| `invalid-ref` | The declared ref could not be resolved |
| `not-synced` | No resolved commit is stored yet |
| `update-available` | The stored state does not match the currently resolved ref |
| `up-to-date` | Stored state matches the currently resolved ref |
| `inspect-failed` | Commit or worktree inspection failed |

## Source Statuses In `project sync`

| Status | Meaning |
| --- | --- |
| `missing-source` | Canonical source repo is not cloned and could not be prepared |
| `invalid-source` | Canonical source path is not a usable Git repo |
| `invalid-ref` | The declared ref could not be resolved |
| `not-synced` | A commit was resolved for a source with no stored state yet |
| `update-available` | The stored state differs from the commit that sync is about to use |
| `up-to-date` | Stored state already matches the desired commit |
| `resolved` | Final sync output label for a newly resolved source row |
| `inspect-failed` | Commit or worktree inspection failed |

## Source Statuses In `project update`

| Status | Meaning |
| --- | --- |
| `resolved` | No prior stored commit existed; one was resolved |
| `updated` | Stored commit changed |
| `up-to-date` | Stored commit already matched the resolved ref |
| `missing-source` | Canonical source repo is not cloned |
| `invalid-source` | Canonical source path is not a usable Git repo |
| `invalid-ref` | The declared ref could not be resolved |

## Link Statuses

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

## Extra Sections

`project status` may include:

- a link table
- `STALE_PATH`

`project sync` may include:

- a link table
- `PRUNED_PATH`

Verbose mode adds section headings such as `SOURCES`, `LINKS`, `STALE`, and `PRUNED`.

`project sync --dry-run` and `project update --dry-run` print:

```text
dry-run
```

before their tables.
