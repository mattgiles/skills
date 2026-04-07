# Multi-Agent Design

The current multi-agent model is intentionally thin.

An agent entry answers one practical question:

```text
Where should this agent's skills be linked?
```

That is why the public agent interface is just `skills_dir`.

## What The Current Design Supports

- global agent roots in config
- project-level overrides for local testing or project-specific installs
- installing one declared skill into multiple agent roots

## What The Current Design Does Not Do

- transform skill contents per agent
- convert between packaging formats
- manage agent-specific lifecycle rules beyond filesystem linking

This keeps the system aligned with its main job: source management, commit resolution, and link orchestration.
