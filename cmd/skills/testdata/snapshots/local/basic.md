# Local Snapshot Suites

## Project Syncs A Local Skill Repo

`repo/repo-one/analytics/SKILL.md`:
```md
# analytics
```

```repo repo-one
commit "initial"
```

`project/.skills.yaml`:
```yaml
sources:
  repo-one:
    url: {{repo:repo-one}}
    ref: main
agents:
  codex:
    skills_dir: ./agent-skills
skills:
  - source: repo-one
    name: analytics
    agents: [codex]
```

```command
skills project sync --verbose
```

```stdout
SOURCES
SOURCE  STATUS  REF  COMMIT  STORED  REPO_PATH  WORKTREE_PATH  MESSAGE
repo-one  resolved  main  <sha>  -  <data>/repos/repo-one  <data>/worktrees/project-<sha>/repo-one/<sha>  -

LINKS
AGENT  SOURCE  SKILL  STATUS  PATH  TARGET  MESSAGE
codex  repo-one  analytics  created  <project>/agent-skills/analytics  <data>/worktrees/project-<sha>/repo-one/<sha>/analytics  -
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

LINKS
AGENT  SOURCE  SKILL  STATUS  PATH  TARGET  MESSAGE
codex  repo-one  analytics  linked  <project>/agent-skills/analytics  <data>/worktrees/project-<sha>/repo-one/<sha>/analytics  -
```

```stderr
```
