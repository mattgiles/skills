# `skills`

`skills` is a Go CLI for managing reusable agent skills from Git repositories.

Project mode is self-contained: it keeps canonical source clones and pinned worktrees under `.agents/cache/` inside the repo, installs canonical links in `.agents/skills`, and manages `.claude/skills` adapter links for Claude compatibility. Home mode remains available for shared machine-level installs.

## Install

macOS is supported in the first public release flow.

Install the latest release:

```bash
curl -fsSL https://raw.githubusercontent.com/mattgiles/skills/main/scripts/install.sh | sh
```

Install a specific version:

```bash
curl -fsSL https://raw.githubusercontent.com/mattgiles/skills/main/scripts/install.sh | VERSION=v0.1.0 sh
```

The installer downloads a prebuilt binary from GitHub Releases, verifies its checksum, installs it into a writable directory already on `PATH` when possible, and falls back to a user-local bin directory with a clear `PATH` hint if needed.

## Start Here

- [Documentation Home](docs/index.md)
- [Tutorial: First Project Sync](docs/tutorials/first-project.md)
- [How-to: Set Up Global Config](docs/how-to/set-up-global-config.md)
- [How-to: Install The CLI](docs/how-to/install-the-cli.md)
- [How-to: Release A Version](docs/how-to/release-a-version.md)
- [Reference: CLI](docs/reference/cli.md)
- [Reference: Project Manifest](docs/reference/project-manifest.md)

## Standard Model

- Project scope:
  - `.agents/manifest.yaml`
  - `.agents/state.yaml` as generated runtime state
  - `.agents/cache/repos/`
  - `.agents/cache/worktrees/`
  - `.agents/skills/<skill-name>`
  - `.claude/skills/<skill-name>`
- Home scope:
  - `~/.agents/manifest.yaml`
  - `~/.agents/state.yaml`
  - `~/.agents/skills/<skill-name>`
  - `~/.claude/skills/<skill-name>`

In both scopes, canonical skill links point to pinned worktree directories, not directly to mutable source clones.

In project scope, `skills project init` also ensures the generated runtime artifacts are gitignored. The tracked project input is `.agents/manifest.yaml`; `.agents/state.yaml`, `.agents/cache/`, `.agents/skills/`, and `.claude/skills/` are managed runtime paths.
