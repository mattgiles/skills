# Create A Project Manifest

Each project declares the sources, refs, agents, and skills it wants in `.skills.yaml`.

## Initialize The Manifest

From the project directory:

```bash
skills project init
```

This creates `.skills.yaml` if it does not already exist.

## Add The Required Sections

Minimal working example:

```yaml
sources:
  repo-one:
    url: git@github.com:example/repo-one.git
    ref: main

agents:
  codex:
    skills_dir: ./agent-skills

skills:
  - source: repo-one
    name: analytics
    agents: [codex]
```

## Manifest Rules

- every source must have a `ref`
- every agent override must have `skills_dir`
- every skill must name a declared source
- every skill must include at least one agent
- the same `(source, name)` pair cannot be declared twice
- agents cannot repeat inside a single skill entry

## Check The Manifest

Run:

```bash
skills project status
```

If the manifest is missing or invalid, the command fails with an error.

For the full schema, see [Project Manifest Reference](../reference/project-manifest.md).
