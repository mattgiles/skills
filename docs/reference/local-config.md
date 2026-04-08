# Project Local Config Reference

## File Name

```text
.agents/local.yaml
```

This file stores repo-local user preference for the project cache backend. It is separate from the tracked manifest and is not part of shared home scope.

## Schema

```yaml
cache:
  mode: local
```

## Fields

| Field | Type | Required | Meaning |
| --- | --- | --- | --- |
| `cache.mode` | string | no | Project cache backend: `local` or `global` |

## Allowed Values

| Value | Meaning |
| --- | --- |
| `local` | Use `.agents/cache/repos` and `.agents/cache/worktrees` inside the repo |
| `global` | Use the global config `repo_root` and `worktree_root`, while still installing into the repo-local `.agents/skills` and `.claude/skills` paths |

## Defaults

If `.agents/local.yaml` does not exist, the current implementation falls back to implicit local mode for compatibility:

```yaml
cache:
  mode: local
```

`skills doctor` reports this as `local-config-missing` so the repo user can make the choice explicit.

## How It Is Written

`skills init` writes this file in project scope.

Behavior:

- if `--cache` is passed, that value is used
- if the file already exists and `--cache` is omitted, the existing mode is reused
- if the file does not exist and the terminal is interactive, `skills init` prompts for the cache mode
- if the file does not exist and the command is non-interactive, `skills init` fails unless `--cache` is provided

## Validation

`cache.mode` must be one of:

- `local`
- `global`

Any other value is invalid.
