# Skill Discovery Reference

## Discovery Rule

A skill is discovered when a directory contains a file named `SKILL.md`.

## Skill Identity

The current CLI identifies a skill by:

- source alias
- directory name

The discovered record also includes the relative path within the repo.

## Discovery Sources

There are two discovery modes in the current implementation:

- `skills skill list` scans the checked-out canonical source repo
- project workflows inspect the file list for the resolved commit and map the discovered relative paths into the project's worktree

## Consequences

- one repo can contain many skills
- nested skill directories are allowed
- duplicate directory names within a single repo are ambiguous for project sync, even if they live at different relative paths

If more than one discovered directory has the same name for the selected commit, project link resolution fails with `ambiguous-skill`.
