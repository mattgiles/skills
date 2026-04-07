# Project State Reference

## Project File Name

```text
.agents/state.yaml
```

## Home File Name

```text
~/.agents/state.yaml
```

This file is managed by `skills`. It records resolved source commits plus the managed canonical and Claude adapter links for the active scope.

## Schema

```yaml
sources:
  - source: repo-one
    ref: main
    resolved_commit: 0123456789abcdef
skill_links:
  - path: /abs/path/to/.agents/skills/analytics
    target: /abs/path/to/worktree/analytics
    source: repo-one
    skill: analytics
claude_links:
  - path: /abs/path/to/.claude/skills/analytics
    target: /abs/path/to/.agents/skills/analytics
    source: repo-one
    skill: analytics
```

## `sources[]`

| Field | Type | Meaning |
| --- | --- | --- |
| `source` | string | Source alias |
| `ref` | string | Ref used when the commit was resolved |
| `resolved_commit` | string | Full resolved commit hash |

## `skill_links[]`

Canonical managed symlinks in the scope’s `.agents/skills` directory.

## `claude_links[]`

Managed Claude adapter symlinks in the scope’s `.claude/skills` directory.

## Notes

- project scope writes `.agents/state.yaml` in the repo
- home scope writes `~/.agents/state.yaml` by default
- stale-link detection is tracked separately for canonical skill links and Claude adapter links
