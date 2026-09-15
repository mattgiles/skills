# Project Manifest Reference

## Project File Name

```text
.agents/manifest.yaml
```

## Home File Name

```text
~/.agents/manifest.yaml
```

## Schema

```yaml
sources:
  repo-one:
    url: git@github.com:example/repo-one.git
    ref: main
  aws-toolkit:
    url: https://github.com/aws/agent-toolkit-for-aws.git
    ref: main
    include: [skills]
    exclude: [skills/experimental]

skills:
  - source: repo-one
    name: analytics
  - source: aws-toolkit
    name: amazon-bedrock
    path: skills/core-skills/amazon-bedrock
```

## Top-Level Fields

| Field | Type | Required | Meaning |
| --- | --- | --- | --- |
| `sources` | map | yes in practice | Source declarations |
| `skills` | list | yes in practice | Canonical skills to install into `.agents/skills` |

## `sources.<alias>`

| Field | Type | Required | Meaning |
| --- | --- | --- | --- |
| `url` | string | yes for project mode | Source Git URL or local repo path |
| `ref` | string | yes | Branch, tag, or commit to resolve |
| `include` | list of strings | no | Repo-relative directory prefixes to search for skills. When empty, the whole repository is searched. |
| `exclude` | list of strings | no | Repo-relative directory prefixes to skip when discovering skills. |

Notes:

- alias validation uses the same rules as global config aliases
- `ref` must not be empty
- both repo and home/global manifests require `url`
- project cache backend is not declared here; each repo user chooses it in `.agents/local.yaml`

Source scope (`include`/`exclude`):

- a discovered skill is kept iff it lies under an `include` entry (or `include`
  is empty) **and** not under any `exclude` entry — exclude wins
- entries are directory prefixes, not globs: a skill at `R` is "under" `P` when
  `R == P` or `R` starts with `P/`; `.` matches everything
- entries are normalized (trimmed, `./` and trailing `/` removed) before use
- each entry must be non-empty, repo-relative (not absolute), and must not
  escape the repository (`..`)
- `exclude: ["."]` is rejected because it would exclude the whole repository;
  `include: ["."]` is allowed (no-op)
- the scope applies everywhere discovery feeds a decision: `sync`, `status`,
  `doctor` link resolution, and `skills skill list` (use `--all` to bypass it)

## `skills[]`

| Field | Type | Required | Meaning |
| --- | --- | --- | --- |
| `source` | string | yes | Source alias |
| `name` | string | yes | Skill name; the link directory under `.agents/skills/`. Without `path`, also the discovered directory name to match. |
| `path` | string | no | Repo-relative skill directory (the `Path` column of `skills skill list`). When set, it selects the discovered skill regardless of its basename and `name` is only the link directory label. |

Validation rules:

- `source` must not be empty
- `name` must not be empty
- the same `(source, name)` pair cannot appear more than once
- each skill must reference a declared source
- `path`, when set, must be repo-relative (not absolute) and must not escape
  the repository (`..`)
- two entries in the same source cannot declare the same normalized `path`
  (`a/x` and `a/x/SKILL.md` count as the same path)

Selector semantics:

- with `path:` set, the skill is chosen by exact repo-relative directory after
  normalization (trim, `./`/trailing `/` removed, trailing `SKILL.md` dropped);
  `path: .` selects a repository-root skill
- without `path:`, the skill is chosen by directory name as before
- two same-named skills in one source can coexist by giving each entry a
  distinct `name` and its own `path`
- because `name` is only the link directory when `path` is set, `path` doubles
  as a link-rename mechanism

## Project Manifest Fragments

Project workspaces may also contain committed manifest fragments:

```text
.agents/manifest.d/*.yaml
```

Fragments use the same `sources` and `skills` schema as the main manifest. The
CLI reads `.agents/manifest.yaml` first, then merges fragment files in
lexicographic filename order. Directories and files without the lowercase
`.yaml` suffix are ignored.

Merge rules:

- every source alias must be declared exactly once across the main manifest and
  all fragments
- a fragment skill may reference a source declared in the main manifest or in
  another fragment
- repeated `(source, name)` skill pairs are deduplicated; the first declaration
  wins, with the main manifest taking precedence over fragments — the dedupe
  key is still `(source, name)` even when the entries declare different `path`
  values
- source `include`/`exclude` lists ride along on the source entry unchanged
- the final merged manifest must satisfy all normal validation rules, so two
  merged entries sharing a `path` under different names are rejected

Commands such as `status`, `sync`, `update`, `source list`, `skill list`, and
`doctor` read the merged effective manifest. Commands that add sources or skills
still write only to `.agents/manifest.yaml`.

Manifest fragments are supported only for project workspaces. The home/global
manifest at `~/.agents/manifest.yaml` has no fragment directory.

## Default Manifest

`skills init` and `skills init --global` currently create:

```yaml
sources: {}
skills: []
```
