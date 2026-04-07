# Use Local Agent Directories

Project scope now defaults to project-local canonical directories. You do not need per-agent overrides for the standard workflow.

## Project-Local Canonical Paths

`skills project init` and `skills project sync` use:

- `.agents/cache/repos/` for project-local canonical source clones
- `.agents/cache/worktrees/` for project-local worktrees
- `.agents/skills/<skill-name>` for canonical installed skills
- `.claude/skills/<skill-name>` for Claude adapters

## Inspect The Links

```bash
ls -l .agents/skills
ls -l .claude/skills
```

This is the standard local workflow because it keeps project-specific installed skills inside the repo and separate from shared home installs.
