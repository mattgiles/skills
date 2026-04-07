# Find Skills In Sources

Use `skills skill list` to inspect discovered skills in synced source repos.

## List All Discovered Skills

```bash
skills skill list
```

This scans synced source repos and lists directories that contain `SKILL.md`.

## Filter By Source

```bash
skills skill list --source repo-one
```

## Use Verbose Output

```bash
skills --verbose skill list
```

Verbose output adds the absolute discovered path.

## Understand Unsynced-Source Warnings

If a configured source has not been cloned yet, `skill list` skips it and writes a warning to stderr:

```text
warning: skipping unsynced source "repo-one"
```

Sync the source first:

```bash
skills source sync repo-one
```

For the exact discovery rule, see [Skill Discovery](../reference/skill-discovery.md).
