# Global Config Reference

## File Location

Default config path:

- `$XDG_CONFIG_HOME/skills/config.yaml` when `XDG_CONFIG_HOME` is set
- otherwise `~/.config/skills/config.yaml`

## Schema

```yaml
repo_root: ~/.local/share/skills/repos
worktree_root: ~/.local/share/skills/worktrees
agents:
  claude:
    skills_dir: ~/.claude/skills
  codex:
    skills_dir: ~/.codex/skills
sources:
  repo-one:
    url: git@github.com:example/repo-one.git
```

## Fields

| Field | Type | Required | Meaning |
| --- | --- | --- | --- |
| `repo_root` | string | no | Canonical clone root |
| `worktree_root` | string | no | Root for project-pinned worktrees |
| `agents` | map | no | Global agent install roots |
| `sources` | map | no | Registered source aliases and URLs |

### `agents.<name>.skills_dir`

| Field | Type | Required | Meaning |
| --- | --- | --- | --- |
| `skills_dir` | string | yes for each agent entry | Destination directory for that agent's symlinked skills |

### `sources.<alias>.url`

| Field | Type | Required | Meaning |
| --- | --- | --- | --- |
| `url` | string | yes for each source entry | Git URL or local repo path |

## Defaults

Built-in default config:

```yaml
repo_root: ~/.local/share/skills/repos
worktree_root: ~/.local/share/skills/worktrees
agents:
  claude:
    skills_dir: ~/.claude/skills
  codex:
    skills_dir: ~/.codex/skills
sources: {}
```

If `XDG_DATA_HOME` is set, the default storage roots become:

- `$XDG_DATA_HOME/skills/repos`
- `$XDG_DATA_HOME/skills/worktrees`

## Path Resolution

The current implementation:

- expands `~`
- expands environment variables
- resolves relative paths to absolute paths

Agent `skills_dir` values in global config are treated as path values and resolved before use.
