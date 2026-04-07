# Troubleshoot Common Failures

This page covers the most common current failure modes.

## `manifest not found`

Cause:

- `.skills.yaml` does not exist in the current directory

Fix:

```bash
skills project init
```

Then fill in the file.

## `invalid alias`

Cause:

- a source or agent alias contains unsupported characters

Fix:

- use lowercase letters, numbers, `_`, and `-`
- start with a lowercase letter or number

## `warning: skipping unsynced source`

Cause:

- `skills skill list` saw a configured source that has not been cloned yet

Fix:

```bash
skills source sync <alias>
```

## `missing-source`

Cause:

- a project source is declared, but the canonical repo is not available locally

Fix:

```bash
skills source sync
```

or run:

```bash
skills project sync
```

which will clone missing sources as part of sync.

## `invalid-ref`

Cause:

- the project's `ref` could not be resolved in the source repo

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

- the destination path exists and is not a managed symlink to the expected target

Fix:

- remove or move the conflicting path
- re-run `skills project sync`

## `stale`

Cause:

- the symlink points at an older managed target than the project currently wants

Fix:

```bash
skills project sync
```

## `inspect-failed`

Cause:

- `skills` could not inspect the desired commit or worktree state

Fix:

- check the error text in the `MESSAGE` column
- verify the stored commit is valid
- re-run `skills project sync` if the project state is inconsistent
