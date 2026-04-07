# Find Skills In Sources

Use `skills skill list` to inspect discovered skills in synced source repos.

By default, `skill list` uses the current repo manifest sources. Use `--global` when you want to inspect globally configured sources instead.

## List All Discovered Skills

```bash
skills skill list
```

This scans the synced repos for sources declared in `.agents/manifest.yaml` and lists directories that contain `SKILL.md`.

## Filter By Source

```bash
skills skill list --source repo-one
```

## List Global Sources Instead

```bash
skills skill list --global
skills skill list --global --source repo-one
```

## Use Verbose Output

```bash
skills --verbose skill list
```

Verbose output adds the absolute discovered path.

## Understand Unsynced-Source Warnings

If a selected source has not been cloned yet, `skill list` skips it and writes a warning to stderr:

```text
warning: skipping unsynced source "repo-one"
```

Sync the source first.

Repo mode:

```bash
skills sync
```

Global mode:

```bash
skills source sync repo-one
```

For the exact discovery rule, see [Skill Discovery](../reference/skill-discovery.md).
