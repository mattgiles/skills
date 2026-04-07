# Create A Project Manifest

Each project declares the sources, refs, and canonical installed skills it wants in `.agents/manifest.yaml`.

## Initialize The Workspace

From the project directory:

```bash
skills project init
```

This creates:

- `.agents/manifest.yaml`
- `.agents/skills/`
- `.claude/skills/`

## Add The Required Sections

Minimal working example:

```yaml
sources:
  repo-one:
    url: git@github.com:example/repo-one.git
    ref: main

skills:
  - source: repo-one
    name: analytics
```

## Manifest Rules

- every source must have a `ref`
- every skill must name a declared source
- the same `(source, name)` pair cannot be declared twice

## Check The Manifest

Run:

```bash
skills project status
```

If the manifest is missing or invalid, the command fails with an error.
