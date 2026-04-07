# First Project Sync

This tutorial walks through the first successful `skills` workflow with a local Git repo. It uses the new standardized project layout:

- `.agents/manifest.yaml`
- `.agents/local.yaml`
- `.agents/cache/`
- `.agents/skills/`
- `.claude/skills/`

By the end you will:

- create a test skill repo
- register it as a source
- initialize a standardized project workspace
- sync a canonical skill into `.agents/skills`
- inspect the Claude adapter in `.claude/skills`

## Prerequisites

- Git installed
- `skills` installed

Install the latest public release on macOS:

```bash
curl -fsSL https://raw.githubusercontent.com/mattgiles/skills/main/scripts/install.sh | sh
export SKILLS_BIN="skills"
```

If you are working from a local clone as a contributor, building from source is still fine:

```bash
mkdir -p ./bin
go build -o ./bin/skills ./cmd/skills
export SKILLS_BIN="/absolute/path/to/skills-repo/bin/skills"
```

## 2. Create A Clean Working Area

```bash
mkdir -p /tmp/skills-tutorial
cd /tmp/skills-tutorial
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

## 4. Initialize A Standardized Project Workspace

```bash
mkdir -p project
cd project
$SKILLS_BIN init --cache=local
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

## 5. Sync The Project

```bash
$SKILLS_BIN sync --verbose
```

On the first run you should see:

- a `SOURCES` section with `repo-one` in status `resolved`
- a `SKILLS` section with `analytics` in status `created`
- a `CLAUDE` section with `analytics` in status `created`

## 6. Inspect The Result

```bash
$SKILLS_BIN status --verbose
ls -l .agents/cache
ls -l .agents/skills
ls -l .claude/skills
```

The canonical link in `.agents/skills/analytics` should point into `.agents/cache/worktrees`. The Claude adapter in `.claude/skills/analytics` should point at the canonical `.agents/skills/analytics` path.
