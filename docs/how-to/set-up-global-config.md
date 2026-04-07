# Set Up Global Config

Use global config for machine-level defaults: canonical source storage, worktree storage, agent roots, and registered sources.

## Create The Default Config

```bash
skills config init
```

Default config path:

- `$XDG_CONFIG_HOME/skills/config.yaml` when `XDG_CONFIG_HOME` is set
- otherwise `~/.config/skills/config.yaml`

## Start From The Default Shape

The default file contains:

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

If `XDG_DATA_HOME` is set, the default `repo_root` and `worktree_root` move under that directory instead.

## Change Storage Roots

Set custom locations by editing the file:

```yaml
repo_root: ~/src/skills/repos
worktree_root: ~/src/skills/worktrees
```

`skills` expands `~`, environment variables, and relative paths.

## Add Agent Roots

You can define more agents or change existing roots:

```yaml
agents:
  codex:
    skills_dir: ~/.codex/skills
  claude:
    skills_dir: ~/.claude/skills
  demo:
    skills_dir: ~/tmp/demo-skills
```

For field-level details, see [Global Config Reference](../reference/config.md).
