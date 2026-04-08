# Local Snapshot Suites

## Project Syncs A Local Skill Repo

`repo/repo-one/analytics/SKILL.md`:
```md
# analytics
```

```repo repo-one
commit "initial"
```

`project/.agents/manifest.yaml`:
```yaml
sources:
  repo-one:
    url: {{repo:repo-one}}
    ref: main
skills:
  - source: repo-one
    name: analytics
```

```command
skills sync --verbose
```

```stdout
# Workspace
Scope  repo
Root  <project>
Installs  <project>/.agents/skills
Cache  local
Worktrees  <project>/.agents/cache/worktrees
Repos  <project>/.agents/cache/repos


# Sources
Source  Status  Ref  Commit  Stored  Repo Path  Worktree Path  Message
repo-one  resolved  main  <sha>  -  <project>/.agents/cache/repos/repo-one  <project>/.agents/cache/worktrees/project-<sha>/repo-one/<sha>  -


# Skills
Source  Skill  Status  Path  Target  Message
repo-one  analytics  created  <project>/.agents/skills/analytics  <project>/.agents/cache/worktrees/project-<sha>/repo-one/<sha>/analytics  -


# Claude
Source  Skill  Status  Path  Target  Message
repo-one  analytics  created  <project>/.claude/skills/analytics  <project>/.agents/skills/analytics  -
```

```stderr
```

```command
skills status --verbose
```

```stdout
# Workspace
Scope  repo
Root  <project>
Installs  <project>/.agents/skills
Cache  local
Worktrees  <project>/.agents/cache/worktrees
Repos  <project>/.agents/cache/repos


# Sources
Source  Status  Ref  Commit  Stored  Repo Path  Worktree Path  Message
repo-one  up-to-date  main  <sha>  <sha>  <project>/.agents/cache/repos/repo-one  <project>/.agents/cache/worktrees/project-<sha>/repo-one/<sha>  -


# Skills
Source  Skill  Status  Path  Target  Message
repo-one  analytics  linked  <project>/.agents/skills/analytics  <project>/.agents/cache/worktrees/project-<sha>/repo-one/<sha>/analytics  -


# Claude
Source  Skill  Status  Path  Target  Message
repo-one  analytics  linked  <project>/.claude/skills/analytics  <project>/.agents/skills/analytics  -
```

```stderr
```
