# Troubleshoot Common Failures

This page covers the most common current failure modes.

Start with:

```bash
skills doctor
```

Or for shared home installs:

```bash
skills doctor --global
```

For the full command surface and output structure, see [Doctor Reference](../reference/doctor.md).

## `ignore-rules-missing`

Cause:

- the effective `.gitignore` does not ignore one or more managed runtime paths:
  - `.agents/state.yaml`
  - `.agents/local.yaml`
  - `.agents/cache/`
  - `.agents/skills/`
  - `.claude/skills/`

Fix:

```bash
skills init --cache=local
```

## `tracked-managed-path`

Cause:

- a file inside a `skills`-managed runtime path is already tracked by Git

Fix:

- move or remove the tracked content from the managed path
- re-run `skills init`

## `local-config-missing`

Cause:

- the repo does not yet have an explicit `.agents/local.yaml`
- `skills` is falling back to implicit local cache mode for compatibility

Fix:

```bash
skills init --cache=local
```

or:

```bash
skills init --cache=global
```

## `manifest not found`

Cause:

- `.agents/manifest.yaml` does not exist in the current scope

Fix:

```bash
skills init --cache=local
```

or:

```bash
skills init --global
```

## `missing-source`

Cause:

- a declared source is not cloned in the canonical repo store

Fix:

```bash
skills source sync
```

or re-run the relevant sync command.

For shared home/global sources:

```bash
skills source sync --global
```

## `invalid-ref`

Cause:

- the declared ref could not be resolved in the source repo

Fix:

- confirm the branch, tag, or commit exists
- make sure the source repo has been fetched recently

## `missing-skill`

Cause:

- the declared skill name does not match any discovered directory name at the resolved commit

Fix:

```bash
skills skill list --source <alias>
```

Then update the manifest.

## `ambiguous-skill`

Cause:

- more than one directory with the same name contains `SKILL.md` in the source repo

Fix:

- rename one of the skill directories upstream
- or choose a source repo that does not contain duplicate directory names

## `conflict`

Cause:

- a managed destination in `.agents/skills` or `.claude/skills` exists as a non-symlink or unmanaged symlink

Fix:

- remove or move the conflicting path
- re-run sync

## `stale`

Cause:

- the canonical `.agents/skills` link points at an older managed worktree target than the scope currently wants

Fix:

```bash
skills sync
```

or:

```bash
skills sync --global
```
