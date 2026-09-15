# Create A Project Manifest

Each project declares the sources, refs, and canonical installed skills it wants in `.agents/manifest.yaml`.

## Initialize The Workspace

From the project directory:

```bash
skills init --cache=local
```

This creates:

- `.agents/manifest.yaml`
- `.agents/local.yaml`
- `.agents/skills/`
- `.claude/skills/`
- `.agents/cache/repos/`
- `.agents/cache/worktrees/`
- `.gitignore` coverage for:
  - `.agents/state.yaml`
  - `.agents/local.yaml`
  - `.agents/cache/`
  - `.agents/skills/`
  - `.claude/skills/`

Use `--cache=global` instead when you want this repo to install skills into `.agents/skills` but reuse the global clone and worktree roots from your machine config.

`skills` treats `.agents/state.yaml`, `.agents/local.yaml`, `.agents/skills/`, and `.claude/skills/` as generated or user-local runtime artifacts. They are not meant to be checked into Git.

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

## Optional Scope And Path Fields

When a source ships the same skill under more than one directory, two optional
fields disambiguate it:

```yaml
sources:
  repo-one:
    url: git@github.com:example/repo-one.git
    ref: main
    include: [skills]      # only search these directory prefixes
    exclude: [skills/wip]  # ...and skip these (exclude wins)

skills:
  - source: repo-one
    name: analytics                # link directory under .agents/skills/
    path: skills/analytics         # exact repo-relative skill directory
```

- `include`/`exclude` scope discovery for the whole source
- `path` selects one discovered skill by its repo-relative directory; `name`
  then only names the link directory

See [Project Manifest](../reference/project-manifest.md) for the full rules.

## Manifest Rules

- every source must have a `ref`
- every project source must have a `url`
- every skill must name a declared source
- the same `(source, name)` pair cannot be declared twice
- `include`, `exclude`, and `path` entries must be repo-relative and must not
  escape the repository; `exclude: ["."]` is rejected
- two skills in the same source cannot declare the same `path`

## Check The Manifest

Run:

```bash
skills status
```

If the manifest is missing or invalid, the command fails with an error.
