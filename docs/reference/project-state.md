# Project State Reference

## File Name

```text
.skills/state.yaml
```

This file is managed by `skills`. It records resolved source commits and the managed symlinks created by project sync.

## Schema

```yaml
sources:
  - source: repo-one
    ref: main
    resolved_commit: 0123456789abcdef
links:
  - path: /abs/path/to/project/agent-skills/analytics
    target: /abs/path/to/worktree/analytics
    source: repo-one
    skill: analytics
    agent: codex
```

## `sources[]`

| Field | Type | Meaning |
| --- | --- | --- |
| `source` | string | Source alias |
| `ref` | string | Ref used when the commit was resolved |
| `resolved_commit` | string | Full resolved commit hash |

## `links[]`

| Field | Type | Meaning |
| --- | --- | --- |
| `path` | string | Managed destination symlink path |
| `target` | string | Desired worktree target |
| `source` | string | Source alias |
| `skill` | string | Skill name |
| `agent` | string | Agent name |

## Notes

- this file is created during `skills project sync`
- `skills project update` updates stored source commits
- stale link detection compares the desired links from the manifest with the managed links recorded here
- if the file becomes inconsistent with reality, `project status` may report `inspect-failed` or `stale`
