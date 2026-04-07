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
skills project sync --verbose
```

```stdout
SOURCES
SOURCE  STATUS  REF  COMMIT  STORED  REPO_PATH  WORKTREE_PATH  MESSAGE
repo-one  resolved  main  <sha>  -  <data>/repos/repo-one  <data>/worktrees/project-<sha>/repo-one/<sha>  -

SKILLS
SOURCE  SKILL  STATUS  PATH  TARGET  MESSAGE
repo-one  analytics  created  <project>/.agents/skills/analytics  <data>/worktrees/project-<sha>/repo-one/<sha>/analytics  -

CLAUDE
SOURCE  SKILL  STATUS  PATH  TARGET  MESSAGE
repo-one  analytics  created  <project>/.claude/skills/analytics  <project>/.agents/skills/analytics  -
```

```stderr
```

```command
skills project status --verbose
```

```stdout
SOURCES
SOURCE  STATUS  REF  COMMIT  STORED  REPO_PATH  WORKTREE_PATH  MESSAGE
repo-one  up-to-date  main  <sha>  <sha>  <data>/repos/repo-one  <data>/worktrees/project-<sha>/repo-one/<sha>  -

SKILLS
SOURCE  SKILL  STATUS  PATH  TARGET  MESSAGE
repo-one  analytics  linked  <project>/.agents/skills/analytics  <data>/worktrees/project-<sha>/repo-one/<sha>/analytics  -

CLAUDE
SOURCE  SKILL  STATUS  PATH  TARGET  MESSAGE
repo-one  analytics  linked  <project>/.claude/skills/analytics  <project>/.agents/skills/analytics  -
```

```stderr
```
