# Sync Project Skills

Use `skills project sync` to ensure sources exist, resolve the correct commit, materialize worktrees, and link the selected skills into agent directories.

## Run A Normal Sync

From the project directory:

```bash
skills project sync
```

Typical first-run results:

- source status `resolved`
- link status `created`

Typical later-run results:

- source status `up-to-date`
- link status `linked`

## Preview Without Changing State

```bash
skills project sync --dry-run
```

Dry-run behavior:

- prints `dry-run`
- does not write `.skills/state.yaml`
- does not create, replace, or remove symlinks

## Inspect More Detail

```bash
skills --verbose project sync
```

Verbose output includes repo paths, worktree paths, link targets, and stored commits.

## Understand Pruning

If `.skills/state.yaml` contains managed links that are no longer declared in the manifest, sync removes them and prints a `PRUNED_PATH` table. In verbose mode it also prints a `PRUNED` heading.

This happens when, for example, you remove a skill from the manifest after a previous sync.

## Re-run Safely

Re-running `project sync` is intended to be idempotent. If nothing changed, link rows remain `linked`.

For exact status meanings, see [Output And Status](../reference/output-and-status.md).
