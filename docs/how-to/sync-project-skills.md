# Sync Project Skills

Use `skills project sync` to ensure sources exist, resolve the correct commit, materialize worktrees in the active project cache backend, and link the selected skills into the project’s canonical `.agents/skills` directory. The same sync also creates Claude adapter links in `.claude/skills`.

Project cache backends:

- `local`: worktrees under `.agents/cache/worktrees`
- `global`: worktrees under the global `worktree_root`

## Run A Normal Sync

From the project directory:

```bash
skills project sync
```

Typical first-run results:

- source status `resolved`
- canonical skill status `created`
- Claude adapter status `created`

Typical later-run results:

- source status `up-to-date`
- canonical skill status `linked`
- Claude adapter status `linked`

## Preview Without Changing State

```bash
skills project sync --dry-run
```

Dry-run behavior:

- prints `dry-run`
- does not write `.agents/state.yaml`
- does not create, replace, or remove symlinks

## Understand Pruning

If `.agents/state.yaml` contains managed links that are no longer declared in the manifest, sync removes them and reports them in `PRUNED_SKILLS` or `PRUNED_CLAUDE`.
