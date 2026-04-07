# Update Project Pins

Use `skills update` to resolve newer commits for project sources. Use `skills sync` to move canonical skill links afterward.

## Update All Project Sources

```bash
skills update
```

Typical results:

- `resolved`: the source had no stored resolved commit yet
- `updated`: the stored commit changed
- `up-to-date`: the stored commit already matched the resolved ref

## Update Selected Sources

```bash
skills update repo-one repo-two
```

## Preview First

```bash
skills update --dry-run
```

This resolves the newer commit and shows the result without changing `.agents/state.yaml`.

## Sync Immediately After Updating

```bash
skills update --sync
```

This records the newer source state and then runs the equivalent of `skills sync`.

## Verify The Result

After updating:

```bash
skills status
```

If you updated without syncing, canonical skill links in `.agents/skills` may show `stale`. Claude adapters can remain `linked` because they point at the canonical paths, not directly at worktree paths.
