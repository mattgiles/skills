# Why Worktrees And Pins

The project separates canonical source clones from project installs.

## Canonical Clones

Project mode keeps one canonical local clone per source under `.agents/cache/repos`. Home/global mode keeps canonical clones under `repo_root`.

## Project Pins

Projects do not install from whatever happens to be checked out in the canonical clone. Instead, project state records a resolved commit for each source.

That gives the CLI a reproducible answer to:

```text
What exact content should this project install right now?
```

## Worktrees

The CLI materializes project content under a worktree root keyed by project identity, source alias, and commit:

- project mode: `.agents/cache/worktrees`
- home/global mode: `worktree_root`

This avoids two common problems:

- one project changing the checkout state for another project
- symlinks pointing into a mutable clone whose working tree may drift

The update flow is split on purpose:

- `project update` moves stored source state forward
- `project sync` moves installed symlinks forward

That split makes change review and dry-run behavior clearer.
