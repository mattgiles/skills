# Live Real-World Repos

## Syncs Dagster Expert From Dagster Skills

`project/.agents/manifest.yaml`:
```yaml
sources:
  dagster:
    url: https://github.com/dagster-io/skills
    ref: master
skills:
  - source: dagster
    name: dagster-expert
```

```command
skills sync --verbose
```

```stdout
scope: repo
root: <project>
installs: <project>/.agents/skills
cache: local
worktrees: <project>/.agents/cache/worktrees
repos: <project>/.agents/cache/repos

SOURCES
SOURCE  STATUS  REF  COMMIT  STORED  REPO_PATH  WORKTREE_PATH  MESSAGE
dagster  resolved  master  <sha>  -  <project>/.agents/cache/repos/dagster  <project>/.agents/cache/worktrees/project-<sha>/dagster/<sha>  -

SKILLS
SOURCE  SKILL  STATUS  PATH  TARGET  MESSAGE
dagster  dagster-expert  created  <project>/.agents/skills/dagster-expert  <project>/.agents/cache/worktrees/project-<sha>/dagster/<sha>/skills/dagster-expert/skills/dagster-expert  -

CLAUDE
SOURCE  SKILL  STATUS  PATH  TARGET  MESSAGE
dagster  dagster-expert  created  <project>/.claude/skills/dagster-expert  <project>/.agents/skills/dagster-expert  -
```

```stderr
```

```command
skills status --verbose
```

```stdout
scope: repo
root: <project>
installs: <project>/.agents/skills
cache: local
worktrees: <project>/.agents/cache/worktrees
repos: <project>/.agents/cache/repos

SOURCES
SOURCE  STATUS  REF  COMMIT  STORED  REPO_PATH  WORKTREE_PATH  MESSAGE
dagster  up-to-date  master  <sha>  <sha>  <project>/.agents/cache/repos/dagster  <project>/.agents/cache/worktrees/project-<sha>/dagster/<sha>  -

SKILLS
SOURCE  SKILL  STATUS  PATH  TARGET  MESSAGE
dagster  dagster-expert  linked  <project>/.agents/skills/dagster-expert  <project>/.agents/cache/worktrees/project-<sha>/dagster/<sha>/skills/dagster-expert/skills/dagster-expert  -

CLAUDE
SOURCE  SKILL  STATUS  PATH  TARGET  MESSAGE
dagster  dagster-expert  linked  <project>/.claude/skills/dagster-expert  <project>/.agents/skills/dagster-expert  -
```

```stderr
```

## Syncs Find Skills From Vercel Skills

`project/.agents/manifest.yaml`:
```yaml
sources:
  vercel:
    url: https://github.com/vercel-labs/skills
    ref: main
skills:
  - source: vercel
    name: find-skills
```

```command
skills sync --verbose
```

```stdout
scope: repo
root: <project>
installs: <project>/.agents/skills
cache: local
worktrees: <project>/.agents/cache/worktrees
repos: <project>/.agents/cache/repos

SOURCES
SOURCE  STATUS  REF  COMMIT  STORED  REPO_PATH  WORKTREE_PATH  MESSAGE
vercel  resolved  main  <sha>  -  <project>/.agents/cache/repos/vercel  <project>/.agents/cache/worktrees/project-<sha>/vercel/<sha>  -

SKILLS
SOURCE  SKILL  STATUS  PATH  TARGET  MESSAGE
vercel  find-skills  created  <project>/.agents/skills/find-skills  <project>/.agents/cache/worktrees/project-<sha>/vercel/<sha>/skills/find-skills  -

CLAUDE
SOURCE  SKILL  STATUS  PATH  TARGET  MESSAGE
vercel  find-skills  created  <project>/.claude/skills/find-skills  <project>/.agents/skills/find-skills  -
```

```stderr
```
