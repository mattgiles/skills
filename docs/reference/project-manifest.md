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

skills:
  - source: repo-one
    name: analytics
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

Notes:

- alias validation uses the same rules as global config aliases
- `ref` must not be empty
- both repo and home/global manifests require `url`
- project cache backend is not declared here; each repo user chooses it in `.agents/local.yaml`

## `skills[]`

| Field | Type | Required | Meaning |
| --- | --- | --- | --- |
| `source` | string | yes | Source alias |
| `name` | string | yes | Skill directory name |

Validation rules:

- `source` must not be empty
- `name` must not be empty
- the same `(source, name)` pair cannot appear more than once
- each skill must reference a declared source

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
  wins, with the main manifest taking precedence over fragments
- the final merged manifest must satisfy all normal validation rules

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
