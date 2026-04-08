# Add And Sync A Source

Use source commands to declare a Git repo under an alias in the active manifest and keep a canonical local clone up to date.

If you already know the exact skill you want, [Add A Skill Quickly](add-a-skill-quickly.md) is the shorter path.

## Register A Source

```bash
skills source add --ref main repo-one git@github.com:example/repo-one.git
```

For shared home/global sources:

```bash
skills source add --global --ref main repo-one git@github.com:example/repo-one.git
```

Alias rules:

- lowercase letters, numbers, `_`, and `-`
- must start with a lowercase letter or number

## List Sources

```bash
skills source list
```

Or for the shared home manifest:

```bash
skills source list --global
```

Typical states:

- `missing`: configured but not cloned locally
- `cloned`: local repo exists but no default remote commit was resolved
- `synced`: local repo exists and default remote metadata is available
- `invalid` or `invalid: ...`: path exists but is not a usable Git repo

## Sync All Sources

```bash
skills source sync
```

This clones missing sources and fetches existing ones.

## Sync Selected Sources

```bash
skills source sync repo-one repo-two
skills source sync --global repo-one repo-two
```

## Inspect More Detail

```bash
skills --verbose source list
skills --verbose source sync
```

Verbose output adds local paths and URLs.

For table field definitions, see [Output And Status](../reference/output-and-status.md).
