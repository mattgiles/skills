# Manage Updates

This tutorial shows how a project moves from one resolved commit to another while keeping canonical `.agents/skills` links separate from Claude adapter links.

## Prerequisites

Complete [First Project Sync](first-project.md), or start from an equivalent working project with:

- a synced source
- a `.agents/manifest.yaml` file
- an existing `.agents/state.yaml`

## 1. Advance The Source Repository

```bash
cd /tmp/skills-tutorial/repos/repo-one
printf 'next\n' > README.md
git add README.md
git commit -m "advance main"
cd /tmp/skills-tutorial/project
```

## 2. Preview The Update

```bash
$SKILLS_BIN project update --dry-run
```

Expected behavior:

- the command prints `dry-run`
- the source row shows status `updated`
- the canonical `.agents/skills` link target does not change

## 3. Record The New Commit

```bash
$SKILLS_BIN project update
```

The stored project state now points at the newer commit, but the canonical `.agents/skills` symlink may still point at the old worktree.

## 4. Confirm The Canonical Link Is Stale

```bash
$SKILLS_BIN project status
```

Expected behavior after `project update` and before `project sync`:

- source status `up-to-date`
- canonical skill status `stale`
- Claude adapter status `linked`

That is expected because the Claude adapter points to the canonical `.agents/skills` path, whose pathname did not change.

## 5. Preview And Apply The Canonical Link Update

```bash
$SKILLS_BIN project sync --dry-run
$SKILLS_BIN project sync
```

The canonical skill row should move from `would-update` to `updated`.
