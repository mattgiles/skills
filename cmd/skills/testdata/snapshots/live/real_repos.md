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
skills project sync --verbose
```

```stdout
SOURCES
SOURCE  STATUS  REF  COMMIT  STORED  REPO_PATH  WORKTREE_PATH  MESSAGE
dagster  resolved  master  <sha>  -  <data>/repos/dagster  <data>/worktrees/project-<sha>/dagster/<sha>  -

SKILLS
SOURCE  SKILL  STATUS  PATH  TARGET  MESSAGE
dagster  dagster-expert  created  <project>/.agents/skills/dagster-expert  <data>/worktrees/project-<sha>/dagster/<sha>/skills/dagster-expert/skills/dagster-expert  -

CLAUDE
SOURCE  SKILL  STATUS  PATH  TARGET  MESSAGE
dagster  dagster-expert  created  <project>/.claude/skills/dagster-expert  <project>/.agents/skills/dagster-expert  -
```

```stderr
```

```command
skills project status --verbose
```

```stdout
SOURCES
SOURCE  STATUS  REF  COMMIT  STORED  REPO_PATH  WORKTREE_PATH  MESSAGE
dagster  up-to-date  master  <sha>  <sha>  <data>/repos/dagster  <data>/worktrees/project-<sha>/dagster/<sha>  -

SKILLS
SOURCE  SKILL  STATUS  PATH  TARGET  MESSAGE
dagster  dagster-expert  linked  <project>/.agents/skills/dagster-expert  <data>/worktrees/project-<sha>/dagster/<sha>/skills/dagster-expert/skills/dagster-expert  -

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
skills project sync --verbose
```

```stdout
SOURCES
SOURCE  STATUS  REF  COMMIT  STORED  REPO_PATH  WORKTREE_PATH  MESSAGE
vercel  resolved  main  <sha>  -  <data>/repos/vercel  <data>/worktrees/project-<sha>/vercel/<sha>  -

SKILLS
SOURCE  SKILL  STATUS  PATH  TARGET  MESSAGE
vercel  find-skills  created  <project>/.agents/skills/find-skills  <data>/worktrees/project-<sha>/vercel/<sha>/skills/find-skills  -

CLAUDE
SOURCE  SKILL  STATUS  PATH  TARGET  MESSAGE
vercel  find-skills  created  <project>/.claude/skills/find-skills  <project>/.agents/skills/find-skills  -
```

```stderr
```
