# First Project Sync

This tutorial walks through the first successful `skills` workflow with a local Git repo. It avoids network dependencies and uses project-local paths so you can see exactly what the CLI creates.

By the end you will:

- create a test skill repo
- register it as a source
- create a project manifest
- sync a skill into an agent directory
- inspect the resulting state

## Prerequisites

- Go installed
- Git installed
- a clone of this repository

## 1. Build The CLI

From the repository root:

```bash
mkdir -p ./bin
go build -o ./bin/skills ./cmd/skills
```

You can also use `go run ./cmd/skills ...`, but a local binary keeps the examples shorter.

Set a shell variable for the binary path:

```bash
export SKILLS_BIN="/absolute/path/to/skills-repo/bin/skills"
```

## 2. Create A Clean Working Area

This tutorial keeps config and data in a temporary directory instead of your real home directory.

```bash
mkdir -p /tmp/skills-tutorial
cd /tmp/skills-tutorial

export SKILLS_CONFIG_HOME="$PWD/skills-config"
export SKILLS_DATA_HOME="$PWD/skills-data"
```

## 3. Create A Local Skill Repository

Make a small repo with one skill directory:

```bash
mkdir -p repos/repo-one/analytics
printf '# analytics\n' > repos/repo-one/analytics/SKILL.md

cd repos/repo-one
git init -b main
git config user.name "Tutorial User"
git config user.email "tutorial@example.com"
git add .
git commit -m "initial"
cd /tmp/skills-tutorial
```

## 4. Initialize Global Config

Create the default config file:

```bash
$SKILLS_BIN config init
```

You should see output like:

```text
created config: /tmp/skills-tutorial/skills-config/skills/config.yaml
```

## 5. Register And Sync The Source

Register the repo under an alias, then clone or fetch it into the canonical source store:

```bash
$SKILLS_BIN source add repo-one "$PWD/repos/repo-one"
$SKILLS_BIN source sync
$SKILLS_BIN source list
```

The source should show up with a status of `cloned` or `synced`.

## 6. Create A Project Directory

Now create a project that wants to use the skill:

```bash
mkdir -p project
cd project
$SKILLS_BIN project init
```

Edit `.skills.yaml` so it declares the source, a local agent directory, and the skill:

```yaml
sources:
  repo-one:
    url: /tmp/skills-tutorial/repos/repo-one
    ref: main
agents:
  codex:
    skills_dir: ./agent-skills
skills:
  - source: repo-one
    name: analytics
    agents: [codex]
```

## 7. Sync The Project

Run the sync:

```bash
$SKILLS_BIN project sync --verbose
```

On the first run you should see:

- a `SOURCES` table with `repo-one` in status `resolved`
- a `LINKS` table with `analytics` in status `created`

The command also writes `.skills/state.yaml` and creates a symlink at `./agent-skills/analytics`.

## 8. Inspect The Result

Check project status:

```bash
$SKILLS_BIN project status --verbose
```

After a successful sync, the usual steady-state output is:

- source status `up-to-date`
- link status `linked`

You can also inspect the symlink directly:

```bash
ls -l ./agent-skills
```

The link target should point into the worktree root under `SKILLS_DATA_HOME`.

## What You Learned

- `skills` stores canonical source clones separately from project worktrees
- project sync installs skills by symlink, not by copy
- `.skills.yaml` declares what a project wants
- `.skills/state.yaml` records the resolved commit and managed links

Next:

- [Manage Updates](manage-updates.md)
- [Sync Project Skills](../how-to/sync-project-skills.md)
- [Project Manifest Reference](../reference/project-manifest.md)
