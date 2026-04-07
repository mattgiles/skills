# `skills`

`skills` is a Go CLI for managing reusable agent skills from Git repositories.

It is designed to make skill installs reproducible and explicit:

- sources come from Git
- refs resolve to pinned commits
- worktrees materialize those commits
- installed skills are symlinks to those pinned worktrees, not mutable source clones

`skills` supports two equally valid workflows:

- project installs: a repo keeps its own manifest and installed skills inside the repo
- global/home installs: a machine keeps shared installed skills for reuse across many repos

For project installs, each user can independently choose whether that repo uses:

- a repo-local cache under `.agents/cache/`
- a shared global cache from the user's `skills` config

## Project Goals

- Make agent skills easy to install from public or private Git repos.
- Keep installs reproducible by resolving refs to concrete commits.
- Separate tracked declarations from generated runtime state.
- Support both self-contained project workflows and shared machine-level workflows.
- Keep the resulting skill layout simple for downstream tools to consume.

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

The installer downloads a prebuilt binary from GitHub Releases, verifies its checksum, and installs it into a writable directory already on `PATH` when possible.

Check the install:

```bash
skills version
skills --help
```

## Initialize

Inside a Git repo, `skills init` will prompt you to choose between repo-local and global initialization if the repo does not already contain `skills` artifacts.

You can always choose explicitly:

```bash
skills init --project --cache=local
skills init --project --cache=global
skills init --global
```

## Quickstart

### Repo-Local Workflow

Use this when you want a repo to declare and install its own skills in `.agents/skills`.

Initialize the project with a repo-local cache:

```bash
skills init --project --cache=local
```

That creates:

- `.agents/manifest.yaml`
- `.agents/local.yaml`
- `.agents/cache/repos/`
- `.agents/cache/worktrees/`
- `.agents/skills/`
- `.claude/skills/`

It also ensures generated runtime paths are gitignored:

- `.agents/state.yaml`
- `.agents/local.yaml`
- `.agents/cache/`
- `.agents/skills/`
- `.claude/skills/`

Add a source and a skill by editing `.agents/manifest.yaml`:

```yaml
sources:
  repo-one:
    url: git@github.com:example/repo-one.git
    ref: main

skills:
  - source: repo-one
    name: analytics
```

Sync the project:

```bash
skills project sync
```

Inspect the result:

```bash
skills project status
skills doctor
```

Project installs are canonical symlinks in `.agents/skills/`.

If you want the repo to keep its installs in `.agents/skills` but reuse a shared machine-level Git cache, initialize it with:

```bash
skills init --project --cache=global
```

That writes an untracked `.agents/local.yaml` file for your user and makes future `project` commands use the global clone/worktree roots from your `skills` config, while still installing into the repo.

### Global / Home Workflow

Use this when you want one machine-level skill installation shared across multiple repos.

Initialize the shared home workspace:

```bash
skills init --global
```

Create or inspect global config when you want to customize storage roots or shared install locations:

```bash
skills config init
```

Register a source alias in global config:

```bash
skills source add repo-one git@github.com:example/repo-one.git
```

Sync the canonical local source clone:

```bash
skills source sync
```

Edit `~/.agents/manifest.yaml`:

```yaml
sources:
  repo-one:
    ref: main

skills:
  - source: repo-one
    name: analytics
```

Sync shared home installs:

```bash
skills home sync
```

Inspect the result:

```bash
skills home status
skills doctor --global
```

Global installs live in `~/.agents/skills/` by default, with Claude adapter links in `~/.claude/skills/`.

## Adding And Syncing Skills

The lifecycle is the same in both workflows:

1. declare the source and desired skills
2. resolve each source ref to a concrete commit
3. materialize that commit in a worktree
4. create canonical symlinks in `.agents/skills` or `~/.agents/skills`
5. create Claude adapter links in `.claude/skills` or `~/.claude/skills`

When you want newer commits for the same refs, run:

```bash
skills project update --sync
skills home update --sync
```

## Project Installs vs Global Installs

Choose project installs when:

- the repo should declare and install its own skills
- other contributors should not need machine-level `skills` setup
- you want `.agents/skills` inside the repo

Choose a repo-local project cache when:

- you want the repo's clone/worktree data isolated inside `.agents/cache/`
- you want the project to stay fully local even at the cache layer

Choose a global project cache when:

- you want repo installs in `.agents/skills` but shared clone/worktree storage
- you work across many repos and want to reuse fetched Git data

Choose global/home when:

- you want one shared install for many repos
- you prefer a machine-level source registry
- you want shared clone and worktree storage outside individual repos
- you want the installed skills themselves to live in `~/.agents/skills`

You can use both. A project install can use either a repo-local cache or a global cache, and that per-repo cache choice is a local user preference, not tracked repo config.

## Documentation

- [Documentation Home](docs/index.md)
- [Tutorial: First Project Sync](docs/tutorials/first-project.md)
- [How-to: Install The CLI](docs/how-to/install-the-cli.md)
- [How-to: Set Up Global Config](docs/how-to/set-up-global-config.md)
- [How-to: Add And Sync A Source](docs/how-to/add-and-sync-a-source.md)
- [Reference: CLI](docs/reference/cli.md)
- [Reference: Project Manifest](docs/reference/project-manifest.md)
