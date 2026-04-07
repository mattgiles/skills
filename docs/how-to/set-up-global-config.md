# Set Up Global Config

Use global config for machine-level defaults: canonical clone storage, worktree storage, and shared home install roots.

## Create The Default Config

```bash
skills config init
```

Default config path:

- `$SKILLS_CONFIG_HOME/skills/config.yaml` when `SKILLS_CONFIG_HOME` is set
- otherwise `~/.config/skills/config.yaml`

## Start From The Default Shape

```yaml
repo_root: ~/.local/share/skills/repos
worktree_root: ~/.local/share/skills/worktrees
shared_skills_dir: ~/.agents/skills
shared_claude_skills_dir: ~/.claude/skills
```

If `SKILLS_DATA_HOME` is set, the default `repo_root` and `worktree_root` move under that directory instead.

## Change Storage Roots

```yaml
repo_root: ~/src/skills/repos
worktree_root: ~/src/skills/worktrees
```

## Change Shared Home Install Roots

```yaml
shared_skills_dir: ~/agent-config/.agents/skills
shared_claude_skills_dir: ~/agent-config/.claude/skills
```

`skills` expands `~`, environment variables, and relative paths.
