# First Project Sync

This tutorial walks through the first successful `skills` workflow with a local Git repo. It uses the new standardized project layout:

- `AGENTS.md`
- `CLAUDE.md`
- `.agents/manifest.yaml`
- `.agents/skills/`
- `.claude/skills/`

By the end you will:

- create a test skill repo
- register it as a source
- initialize a standardized project workspace
- sync a canonical skill into `.agents/skills`
- inspect the Claude adapter in `.claude/skills`

## Prerequisites

- Go installed
- Git installed
- a clone of this repository

## 1. Build The CLI

From the repository root:

```bash
mkdir -p ./bin
go build -o ./bin/skills ./cmd/skills
export SKILLS_BIN="/absolute/path/to/skills-repo/bin/skills"
```

## 2. Create A Clean Working Area

```bash
mkdir -p /tmp/skills-tutorial
cd /tmp/skills-tutorial

export SKILLS_CONFIG_HOME="$PWD/skills-config"
export SKILLS_DATA_HOME="$PWD/skills-data"
```

## 3. Create A Local Skill Repository

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

## 4. Initialize Global Config And Source

```bash
$SKILLS_BIN config init
$SKILLS_BIN source add repo-one "$PWD/repos/repo-one"
$SKILLS_BIN source sync
```

## 5. Initialize A Standardized Project Workspace

```bash
mkdir -p project
cd project
$SKILLS_BIN project init
```

Edit `.agents/manifest.yaml`:

```yaml
sources:
  repo-one:
    url: /tmp/skills-tutorial/repos/repo-one
    ref: main
skills:
  - source: repo-one
    name: analytics
```

## 6. Sync The Project

```bash
$SKILLS_BIN project sync --verbose
```

On the first run you should see:

- a `SOURCES` section with `repo-one` in status `resolved`
- a `SKILLS` section with `analytics` in status `created`
- a `CLAUDE` section with `analytics` in status `created`

## 7. Inspect The Result

```bash
$SKILLS_BIN project status --verbose
ls -l .agents/skills
ls -l .claude/skills
```

The canonical link in `.agents/skills/analytics` should point into the worktree root under `SKILLS_DATA_HOME`. The Claude adapter in `.claude/skills/analytics` should point at the canonical `.agents/skills/analytics` path.
