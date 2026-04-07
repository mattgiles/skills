# Roadmap: `skills`

## Summary

`skills` is a Go CLI for managing agent skills from Git repositories without copying them around by hand.

The product should stay centered on one simple model:

- A Git repo is a **source**
- A `SKILL.md` directory inside that repo is a **skill**
- A project manifest declares which `(source, skill)` pairs it needs
- Installation is done by **symlink**, not copy

V1 should solve two concrete problems well:

- Maintain a canonical local store of skill repositories that can be cloned and updated with Git
- Sync project-declared skills into one or more agent-specific skill directories

## Principles

- Prefer the Unix standard locations for config, data, and cache
- Keep project declarations explicit and machine-readable
- Make updates intentional and reproducible
- Treat one repo as potentially containing many skills
- Support multiple agent targets, but keep the adapter model thin
- Optimize for a simple mental model over ecosystem-style complexity

## Filesystem Layout

Use standard XDG-style defaults on Unix-like systems:

- Config: `~/.config/skills/config.yaml`
- Data: `~/.local/share/skills/repos/`
- Worktrees for pinned project refs: `~/.local/share/skills/worktrees/`
- Cache/index: `~/.cache/skills/`

The global config owns machine-level concerns only:

- Canonical clone root
- Registered sources
- Known agent install roots
- Optional Git transport defaults

Projects own project-level concerns only:

- Required sources
- Pinned refs
- Required skills
- Target agents

## Core Model

The core types for the system:

- `Source`: alias, git URL, canonical clone path
- `DiscoveredSkill`: source alias, skill name, relative path, metadata
- `ProjectSource`: alias, git URL optional, pinned ref
- `ProjectSkill`: source alias, skill name, target agents
- `AgentTarget`: agent name, destination root, install mode
- `ProjectSyncResult`: linked, skipped, warnings, errors

Identity rules:

- A source is identified locally by a stable alias plus Git URL
- A skill is identified by `(source alias, skill directory name)`
- Discovery means finding directories that contain `SKILL.md`
- A repo and a skill are not the same thing

## Config Shapes

Global config:

```yaml
repo_root: ~/.local/share/skills/repos
worktree_root: ~/.local/share/skills/worktrees
agents:
  codex:
    skills_dir: ~/.codex/skills
  claude:
    skills_dir: ~/.claude/skills
sources:
  dbt-agent-skills:
    url: git@github.com:dbt-labs/dbt-agent-skills.git
```

Project manifest:

```yaml
sources:
  dbt-agent-skills:
    url: git@github.com:dbt-labs/dbt-agent-skills.git
    ref: main

skills:
  - source: dbt-agent-skills
    name: dbt-core
    agents: [codex, claude]
  - source: dbt-agent-skills
    name: dbt-cloud
    agents: [codex]
```

Design constraints:

- Keep the project manifest dedicated and tool-owned
- Do not overload `AGENTS.md` with dependency data
- Allow project manifests to declare sources not yet present in global config
- Resolve refs to concrete commits for status and output

## Command Surface

V1 commands:

- `skills config init`
- `skills source add <alias> <git-url>`
- `skills source list`
- `skills source sync [alias...]`
- `skills skill list`
- `skills skill list --source <alias>`
- `skills project init`
- `skills project status`
- `skills project sync`
- `skills project update [source...]`

Expected behavior:

- `source sync` clones missing sources and fetches updates for existing ones
- `skill list` reads discovered skills from all synced sources
- `project sync` ensures sources exist, materializes pinned refs, and links declared skills into agent directories
- `project update` is the explicit act that moves project pins forward
- Re-running sync should be idempotent

## Git and Ref Strategy

Projects should be reproducible by default.

That means:

- Project manifests declare a ref per source
- Sync installs from that declared ref, not from whatever is currently checked out in the canonical clone
- Updating upstream content is explicit, not implicit

Use this implementation strategy in v1:

- Maintain one canonical clone per source
- Fetch updates into the canonical clone
- Materialize project-pinned refs using `git worktree`

Why `git worktree`:

- It preserves a clean single-source-of-truth clone
- It supports different projects pinning different refs without fighting over checkout state
- It keeps the install model simple because symlinks can target a stable checked-out tree

## Multi-Agent Model

V1 should support multiple agent targets from day one, but only at the install-root level.

Each agent adapter should answer one question:

- Where should this agent's skills be linked?

V1 should not attempt:

- Agent-specific packaging formats
- Agent-specific metadata transformations
- Rich lifecycle semantics per agent

If the skill directory is valid, the installer links it into each selected target root.

## Implementation Plan

### Phase 1: Core repo and discovery layer

- Initialize the Go module and CLI entrypoint
- Implement config loading and path resolution
- Implement source registry management
- Implement clone, fetch, and local source status
- Implement skill discovery by scanning for `SKILL.md`

Exit criteria:

- A user can register a source, sync it locally, and list all discovered skills

### Phase 2: Project manifest and sync

- Define the `.skills.yaml` schema
- Implement project init, load, validate, and status
- Implement pinned-ref materialization via `git worktree`
- Implement symlink creation into configured agent roots
- Make sync safe and idempotent

Exit criteria:

- A project can declare skills and sync them into one or more agent directories

### Phase 3: Update workflow and ergonomics

- Implement `project update`
- Show resolved commits in status and sync output
- Improve missing-skill and invalid-ref diagnostics
- Add dry-run and machine-readable output if needed after the base workflow works

Exit criteria:

- Updating a project’s skill pins is explicit, inspectable, and low-friction

## Testing

Core scenarios to cover:

- Register a source and clone it into the canonical repo root
- Discover multiple skills inside one source repo
- List skills across multiple source repos
- Initialize a project manifest and sync skills from multiple sources
- Auto-clone a source during `project sync` if the manifest declares it
- Re-run `project sync` and confirm nothing unnecessary changes
- Pin a project to a specific ref and install from that pinned state
- Update a source upstream without changing a project pin
- Move a project pin forward with `project update`
- Detect invalid refs, missing skills, broken symlinks, and removed sources
- Sync the same skill into multiple agent targets

Test strategy:

- Unit tests for config parsing, manifest validation, and discovery
- Integration tests for Git clone/fetch/worktree flows using fixture repos
- Integration tests for symlink installation and idempotent sync behavior

## Non-Goals for V1

- Remote skill registries
- Dependency resolution between skills
- Automatic updates on every sync
- Copy-mode installs as a first-class path
- Team policy encoded in `AGENTS.md`
- Cross-platform abstraction beyond sane Unix-like behavior

## Defaults and Assumptions

- The CLI name is `skills`
- The project manifest is `.skills.yaml`
- Symlink is the default and only required install mode in v1
- Project manifests pin refs by default
- Source aliases are the primary local handle
- A single source repo may expose many skills
- Canonical sources are local machine state; project manifests are repo state

## Definition of Done for V1

V1 is complete when a user can:

- Register one or more GitHub skill repos as sources
- Clone and update those sources in one canonical location
- List every available skill across those repos
- Declare required skills in a project-local manifest
- Sync those skills into one or more agent-specific directories by symlink
- Reproduce the same project skill set later from pinned refs
