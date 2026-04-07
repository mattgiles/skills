# Why Symlinks

`skills` installs project skills by symlink rather than by copying directory contents.

## The Main Reasons

- copying loses provenance unless the tool stores extra metadata elsewhere
- copied skill directories drift from their source unless you replace them explicitly
- symlinks make the installed path point directly at the materialized worktree for the selected commit

## What Symlinks Buy You

- one source of truth for installed content
- easy inspection of where a skill came from
- cheap updates when the project moves from one commit to another
- less filesystem duplication across agents and projects

## The Tradeoff

Symlinks require the destination path to remain a symlink managed by `skills`. If something else writes a real directory or an unmanaged symlink at that path, the CLI correctly treats it as a conflict instead of overwriting it blindly.
