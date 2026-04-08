# Read Status And Sync Output

Use `skills status`, `skills sync`, and `skills update` together to answer three different questions:

- what is currently installed
- what changed
- what still needs attention

## Read `skills status`

```bash
skills status
```

Typical sections:

- `SOURCES`: ref and commit state for each declared source
- `SKILLS`: canonical `.agents/skills` links
- `CLAUDE`: Claude adapter links

Status is read-only. It tells you whether the stored state and managed links already match the declared manifest.

## Read `skills sync`

```bash
skills sync
```

`sync` is the mutating command that:

- resolves commits when needed
- creates or updates managed links
- prunes old managed links that are no longer declared

When pruning happens, look for:

- `PRUNED_SKILLS`
- `PRUNED_CLAUDE`

## Read `skills update`

```bash
skills update
```

`update` moves the stored resolved commit forward without necessarily changing the installed symlinks.

That means a normal workflow can be:

```bash
skills update
skills status
skills sync
```

## Understand `stale`

After `skills update` and before `skills sync`, a canonical skill link can show `stale`.

That means:

- the manifest and stored source state now point at a newer commit
- the existing canonical symlink still points at the older managed target

Claude links can remain `linked` during this state because they point at the canonical `.agents/skills/<name>` path, not directly at the worktree.

## Preview Before Changing Anything

```bash
skills sync --dry-run
skills update --dry-run
```

Dry-run prints `dry-run` before the tables and does not write state or replace links.

## Use Verbose Output For Paths

```bash
skills --verbose status
skills --verbose sync
skills --verbose update
```

Verbose output adds filesystem paths, targets, and stored commit details that are omitted from the compact tables.

For the exact status vocabulary, see [Output And Status Reference](../reference/output-and-status.md).
