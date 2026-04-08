# Add A Skill Quickly

Use `skills add` when you already know the source alias and skill name you want. It updates the manifest and immediately syncs the result.

## Add A Skill From An Existing Source

```bash
skills add repo-one analytics
```

This:

- adds the `(repo-one, analytics)` skill declaration if it is not already present
- runs sync for the current scope
- creates or updates the canonical and Claude links if the source resolves successfully

## Add A Skill And Create The Source At The Same Time

```bash
skills add repo-one analytics --url git@github.com:example/repo-one.git --ref main
```

Use this when the source alias is not yet declared.

If you omit `--ref`, `skills` infers the remote default branch:

```bash
skills add repo-one analytics --url git@github.com:example/repo-one.git
```

## Add A Skill In Shared Home Scope

```bash
skills add --global repo-one analytics --url git@github.com:example/repo-one.git --ref main
```

This writes to the shared home manifest and installs into the shared home directories.

## Understand No-Op Behavior

If the same `(source, skill)` pair is already declared, the command prints a message and exits successfully:

```text
skill "analytics" from source "repo-one" is already declared
```

## Verify The Result

Run:

```bash
skills status
```

Or in home scope:

```bash
skills status --global
```

For exact flags and command semantics, see [CLI Reference](../reference/cli.md).
