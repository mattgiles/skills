# Use Local Agent Directories

Project installs always use repo-local canonical directories. You do not need per-agent overrides for the standard workflow.

## Project-Local Canonical Paths

`skills project init` and `skills project sync` use:

- `.agents/skills/<skill-name>` for canonical installed skills
- `.claude/skills/<skill-name>` for Claude adapters

When `.agents/local.yaml` selects `cache.mode: local`, project commands also use:

- `.agents/cache/repos/` for canonical source clones
- `.agents/cache/worktrees/` for worktrees

When `.agents/local.yaml` selects `cache.mode: global`, project installs stay repo-local but clone/worktree storage moves to the global config roots.

## Inspect The Links

```bash
ls -l .agents/skills
ls -l .claude/skills
```

This is the standard local workflow because it keeps project-specific installed skills inside the repo and separate from shared home installs.
