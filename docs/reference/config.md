# Global Config Reference

This file controls shared home/global behavior, the global source registry, and the clone/worktree roots used by project repos that opt into `--cache=global`.

## File Location

Default config path:

- `$SKILLS_CONFIG_HOME/skills/config.yaml` when `SKILLS_CONFIG_HOME` is set
- otherwise `~/.config/skills/config.yaml`

## Schema

```yaml
repo_root: ~/.local/share/skills/repos
worktree_root: ~/.local/share/skills/worktrees
shared_skills_dir: ~/.agents/skills
shared_claude_skills_dir: ~/.claude/skills
sources:
  repo-one:
    url: git@github.com:example/repo-one.git
```

## Fields

| Field | Type | Required | Meaning |
| --- | --- | --- | --- |
| `repo_root` | string | no | Canonical clone root for shared home/global workflows and project repos using global cache mode |
| `worktree_root` | string | no | Root for pinned worktrees for shared home/global workflows and project repos using global cache mode |
| `shared_skills_dir` | string | no | Canonical shared home skill directory |
| `shared_claude_skills_dir` | string | no | Shared home Claude adapter directory |
| `sources` | map | no | Registered source aliases and URLs for global/home workflows |

### `sources.<alias>.url`

| Field | Type | Required | Meaning |
| --- | --- | --- | --- |
| `url` | string | yes for each source entry | Git URL or local repo path |

## Defaults

```yaml
repo_root: ~/.local/share/skills/repos
worktree_root: ~/.local/share/skills/worktrees
shared_skills_dir: ~/.agents/skills
shared_claude_skills_dir: ~/.claude/skills
sources: {}
```

If `SKILLS_DATA_HOME` is set, the default storage roots become:

- `$SKILLS_DATA_HOME/skills/repos`
- `$SKILLS_DATA_HOME/skills/worktrees`

## Path Resolution

The current implementation:

- expands `~`
- expands environment variables
- resolves relative paths to absolute paths

Home scope derives its manifest and state paths from `shared_skills_dir`:

- manifest: sibling `manifest.yaml`
- state: sibling `state.yaml`

Project scope only uses this config file when a repo user's `.agents/local.yaml` selects `cache.mode: global`. Project installs still remain repo-local in `.agents/skills`.
