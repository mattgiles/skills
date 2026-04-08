# First Global Workflow

This tutorial walks through the first successful shared home workflow for `skills`.

By the end you will:

- create a test skill repo
- initialize the shared home workspace
- declare a shared source and skill
- sync shared installs into the home directories
- inspect the canonical and Claude links

## Prerequisites

- Git installed
- `skills` installed

Install the latest public release on macOS:

```bash
curl -fsSL https://raw.githubusercontent.com/mattgiles/skills/main/scripts/install.sh | sh
export SKILLS_BIN="skills"
```

## 1. Create A Clean Working Area

```bash
mkdir -p /tmp/skills-global-tutorial
cd /tmp/skills-global-tutorial
```

## 2. Create A Local Skill Repository

```bash
mkdir -p repos/repo-one/analytics
printf '# analytics\n' > repos/repo-one/analytics/SKILL.md

cd repos/repo-one
git init -b main
git config user.name "Tutorial User"
git config user.email "tutorial@example.com"
git add .
git commit -m "initial"
cd /tmp/skills-global-tutorial
```

## 3. Initialize The Shared Home Workspace

```bash
$SKILLS_BIN init --global
```

Edit `~/.agents/manifest.yaml`:

```yaml
sources:
  repo-one:
    url: /tmp/skills-global-tutorial/repos/repo-one
    ref: main
skills:
  - source: repo-one
    name: analytics
```

## 4. Sync Shared Home Installs

```bash
$SKILLS_BIN sync --global --verbose
```

On the first run you should see:

- a `SOURCES` section with `repo-one` in status `resolved`
- a `SKILLS` section with `analytics` in status `created`
- a `CLAUDE` section with `analytics` in status `created`

## 5. Inspect The Result

```bash
$SKILLS_BIN status --global --verbose
ls -l ~/.agents/skills
ls -l ~/.claude/skills
```

The canonical link in `~/.agents/skills/analytics` should point into the configured worktree root. The Claude adapter in `~/.claude/skills/analytics` should point at the canonical `~/.agents/skills/analytics` path.
