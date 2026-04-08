# Install Scope And Cache Mode

There are three related decisions in `skills`, and they solve different problems.

## Install Scope

Install scope answers:

```text
Where should the managed skill links live?
```

There are two scopes:

- project scope: installs into `.agents/skills` and `.claude/skills` inside the repo
- home scope: installs into `shared_skills_dir` and `shared_claude_skills_dir`

## Cache Mode

Cache mode only applies to project scope. It answers:

```text
Where should the source clones and worktrees live for this repo user?
```

Project scope supports:

- `local`: use `.agents/cache/repos` and `.agents/cache/worktrees`
- `global`: use the machine-level `repo_root` and `worktree_root`

This choice does not move the install location. Project installs stay inside the repo either way.

## Ownership

The files are split on purpose:

- `.agents/manifest.yaml`: tracked project intent
- `.agents/local.yaml`: repo-local user preference for project cache mode
- global config: machine-level defaults for shared storage and shared install roots

That split lets a repo declare what skills it needs without forcing every contributor to use the same clone and worktree storage backend.

## Common Misread

`--cache=global` does not mean home installs.

It means:

- install into the repo-local project paths
- reuse the machine-level clone and worktree roots
