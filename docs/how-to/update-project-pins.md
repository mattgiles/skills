# Update Project Pins

Use `skills project update` to resolve newer commits for project sources. Use `project sync` to move installed links afterward.

## Update All Project Sources

```bash
skills project update
```

Typical results:

- `resolved`: the source had no stored resolved commit yet
- `updated`: the stored commit changed
- `up-to-date`: the stored commit already matched the resolved ref

## Update Selected Sources

```bash
skills project update repo-one repo-two
```

Unknown aliases fail with `unknown project source`.

## Preview First

```bash
skills project update --dry-run
```

This resolves the newer commit and shows the result without changing `.skills/state.yaml`.

## Sync Immediately After Updating

```bash
skills project update --sync
```

This records the newer source state and then runs the equivalent of `project sync`.

## Verify The Result

After updating:

```bash
skills project status
```

If you updated without syncing links, link rows may show `stale`. Run `skills project sync` to bring symlinks forward.
