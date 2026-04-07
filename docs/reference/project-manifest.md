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
- project mode requires `url` so the repo declaration is self-contained
- home/global manifests can still omit `url` and rely on the global source registry
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

## Default Manifest

`skills init` and `skills init --global` currently create:

```yaml
sources: {}
skills: []
```
