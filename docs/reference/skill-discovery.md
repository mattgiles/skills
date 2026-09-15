# Skill Discovery Reference

## Discovery Rule

A skill is discovered when a directory contains a file named `SKILL.md`.

## Skill Identity

The current CLI identifies a skill by:

- source alias
- directory name

Special case:

- if `SKILL.md` lives at the repository root, the skill name is the repository basename from the source URL/path

The discovered record also includes the relative path within the repo (the
`Path` column of `skills skill list`). A manifest entry may select a skill by
that relative path instead of its directory name via `path:` — see
[Selectors](#selectors).

## Source Scope

A source may declare `include` and/or `exclude` lists of repo-relative
directory prefixes in the manifest. The scope is applied to discovery results
*before* any selector runs: a discovered skill is kept iff it lies under an
`include` entry (or `include` is empty) and not under any `exclude` entry.
Exclude wins.

The scope affects every place discovery feeds a decision:

- `sync`, `status`, and `doctor` link resolution only see in-scope skills
- `skills skill list` only lists in-scope skills; `--all` bypasses the scope
  and adds a `Scope` column (`included`/`excluded`)

One line — `exclude: [plugins]` — is enough to fix a source that mirrors every
skill under two trees.

## Selectors

A manifest skill entry is matched against in-scope discovery results by its
*selector*:

- with `path:` set, the entry matches the discovered skill whose normalized
  relative path is exactly equal (regardless of directory basename); `name:`
  is then only the link directory label
- without `path:`, the entry matches by directory name as before

Path normalization trims whitespace, removes `./` and trailing `/`, and drops
a trailing `SKILL.md` element, so a copied file path works. `path: .` selects
a repository-root skill.

## Discovery Sources

There are two discovery modes in the current implementation:

- `skills skill list` inspects the fetched manifest ref for each canonical source repo in the active scope:
  - repo mode uses sources from `.agents/manifest.yaml`
  - `--global` uses sources from `~/.agents/manifest.yaml`
- project workflows inspect the file list for the resolved commit and map the discovered relative paths into the project's worktree

## Consequences

- one repo can contain many skills
- a repo-root `SKILL.md` is a valid single-skill source
- nested skill directories are allowed
- duplicate directory names within a single repo are ambiguous for name-based
  matching, even if they live at different relative paths; resolve them with
  a source scope (`include`/`exclude`) or a per-skill `path:` selector

If more than one in-scope discovered directory has the same name for the
selected commit and the manifest entry has no `path:`, project link resolution
fails with `ambiguous-skill` and lists the candidate paths. Pick one with
`skills add <source> <name> --path <candidate>`, install both under distinct
`name`s, or narrow the source scope.
