# Manage Updates

This tutorial shows how a project moves from one resolved commit to another.

By the end you will:

- detect that an update is available
- preview a commit update without changing project state
- record a newer commit
- re-sync links to the new worktree

## Prerequisites

Complete [First Project Sync](first-project.md), or start from an equivalent working project with:

- a synced source
- a `.skills.yaml` file
- an existing `.skills/state.yaml`

## 1. Advance The Source Repository

Make a new commit in the source repo:

```bash
cd /tmp/skills-tutorial/repos/repo-one
printf 'next\n' > README.md
git add README.md
git commit -m "advance main"
cd /tmp/skills-tutorial/project
```

## 2. Check Project Status

Before updating, inspect the project:

```bash
$SKILLS_BIN project status
```

If the project has already recorded a commit, the source may still show `up-to-date` until you run `project update`. That is expected: status compares the manifest and stored state, not your installed links alone.

## 3. Preview The Update

Resolve the newer commit without changing state:

```bash
$SKILLS_BIN project update --dry-run
```

Expected behavior:

- the command prints `dry-run`
- the source row shows status `updated`
- the existing symlink target does not change

## 4. Record The New Commit

Apply the update:

```bash
$SKILLS_BIN project update
```

The source state now points at the newer commit, but the existing symlink may still point at the old worktree.

## 5. Confirm The Link Is Stale

Check status again:

```bash
$SKILLS_BIN project status
```

Expected behavior after `project update` and before `project sync`:

- source status `up-to-date`
- link status `stale`

That means the project state has moved forward but the managed symlink still points at the previous commit's worktree.

## 6. Preview And Apply The Link Update

First preview:

```bash
$SKILLS_BIN project sync --dry-run
```

The link row should show `would-update`.

Then apply it:

```bash
$SKILLS_BIN project sync
```

The link row should show `updated`.

## What You Learned

- `project update` changes resolved source state
- `project sync` changes installed symlinks
- dry-run modes let you preview both steps
- a `stale` link means the project knows about a newer target than the symlink currently uses

Next:

- [Update Project Pins](../how-to/update-project-pins.md)
- [Output And Status Reference](../reference/output-and-status.md)
- [Why Worktrees And Pins](../explanation/why-worktrees-and-pins.md)
