# Add And Sync A Source

Use source commands to register a Git repo under an alias and keep a canonical local clone up to date.

## Register A Source

```bash
skills source add repo-one git@github.com:example/repo-one.git
```

Alias rules:

- lowercase letters, numbers, `_`, and `-`
- must start with a lowercase letter or number

## List Sources

```bash
skills source list
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
```

## Inspect More Detail

```bash
skills --verbose source list
skills --verbose source sync
```

Verbose output adds local paths and URLs.

For table field definitions, see [Output And Status](../reference/output-and-status.md).
