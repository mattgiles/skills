# Live Real-World Repos

## Syncs Dagster Expert From Dagster Skills

`project/.skills.yaml`:
```yaml
sources:
  dagster:
    url: https://github.com/dagster-io/skills
    ref: master
agents:
  codex:
    skills_dir: ./agent-skills
skills:
  - source: dagster
    name: dagster-expert
    agents: [codex]
```

```command
skills project sync --verbose
```

```stdout
SOURCES
SOURCE  STATUS  REF  COMMIT  STORED  REPO_PATH  WORKTREE_PATH  MESSAGE
dagster  resolved  master  <sha>  -  <data>/repos/dagster  <data>/worktrees/project-<sha>/dagster/<sha>  -

LINKS
AGENT  SOURCE  SKILL  STATUS  PATH  TARGET  MESSAGE
codex  dagster  dagster-expert  created  <project>/agent-skills/dagster-expert  <data>/worktrees/project-<sha>/dagster/<sha>/skills/dagster-expert/skills/dagster-expert  -
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

LINKS
AGENT  SOURCE  SKILL  STATUS  PATH  TARGET  MESSAGE
codex  dagster  dagster-expert  linked  <project>/agent-skills/dagster-expert  <data>/worktrees/project-<sha>/dagster/<sha>/skills/dagster-expert/skills/dagster-expert  -
```

```stderr
```

## Syncs Find Skills From Vercel Skills

`project/.skills.yaml`:
```yaml
sources:
  vercel:
    url: https://github.com/vercel-labs/skills
    ref: main
agents:
  codex:
    skills_dir: ./agent-skills
skills:
  - source: vercel
    name: find-skills
    agents: [codex]
```

```command
skills project sync --verbose
```

```stdout
SOURCES
SOURCE  STATUS  REF  COMMIT  STORED  REPO_PATH  WORKTREE_PATH  MESSAGE
vercel  resolved  main  <sha>  -  <data>/repos/vercel  <data>/worktrees/project-<sha>/vercel/<sha>  -

LINKS
AGENT  SOURCE  SKILL  STATUS  PATH  TARGET  MESSAGE
codex  vercel  find-skills  created  <project>/agent-skills/find-skills  <data>/worktrees/project-<sha>/vercel/<sha>/skills/find-skills  -
```

```stderr
```
