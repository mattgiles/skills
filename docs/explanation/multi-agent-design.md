# Multi-Agent Design

The current install model is intentionally thin.

`skills` does not currently expose a generic multi-agent configuration layer. Its user-facing model is simpler:

- canonical installs live in `.agents/skills` for project scope or `shared_skills_dir` for home scope
- Claude adapter links live in `.claude/skills` for project scope or `shared_claude_skills_dir` for home scope

## What The Current Design Supports

- one declared skill can be installed in project scope
- the same declared skill can also be installed separately in shared home scope
- Claude compatibility is handled as a second managed link layer that targets the canonical install path

## What The Current Design Does Not Do

- transform skill contents per agent
- convert between packaging formats
- manage arbitrary agent-specific install roots beyond the canonical and Claude paths
- expose a public `skills_dir` or generalized per-agent schema

This keeps the system aligned with its main job: source management, commit resolution, and link orchestration.
