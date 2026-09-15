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

Cause (the status message says which):

- the declared skill name does not match any in-scope discovered directory name at the resolved commit
- the declared `path:` does not exist in the source (`no skill directory at path "..."`)
- the declared `path:` exists but is filtered out by the source's `include`/`exclude` scope (`skill path "..." is excluded by the source's include/exclude scope`)
- a same-named directory exists only outside the source scope (`skill directory exists only outside the source's include/exclude scope: ...`)

Fix:

```bash
skills skill list --source <alias> --all
```

`--all` shows every discovered skill with a `Scope` column so you can see what
the source's `include`/`exclude` lists are hiding. Then update the manifest:
fix the `name`/`path`, or widen the scope with `skills source add <alias> <url>
--include ... --exclude ...`.

## `ambiguous-skill`

Cause:

- more than one in-scope directory with the same name contains `SKILL.md` in the source repo, and the manifest entry has no `path:`

The status message lists the candidate paths:

```text
multiple skills share this directory name; set path: to one of: plugins/aws-core/skills/amazon-bedrock, skills/core-skills/amazon-bedrock
```

Fix — pick one of two options:

1. Pin this entry to one candidate path (`name` stays the link directory):

   ```bash
   skills add <alias> <name> --path skills/core-skills/amazon-bedrock
   ```

2. Scope the source so only one copy is discovered (fixes every duplicate in
   the source at once):

   ```bash
   skills source add <alias> <url> --exclude plugins
   skills sync
   ```

To install *both* copies, give each its own `name` and `path`:

```bash
skills add <alias> amazon-bedrock --path skills/core-skills/amazon-bedrock
skills add <alias> amazon-bedrock-plugin --path plugins/aws-core/skills/amazon-bedrock
```

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
