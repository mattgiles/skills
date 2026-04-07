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
| `url` | string | no | Scope-local source URL override |
| `ref` | string | yes | Branch, tag, or commit to resolve |

Notes:

- alias validation uses the same rules as global config aliases
- `ref` must not be empty
- a source must either declare a `url` here or exist in global config

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

`skills project init` and `skills home init` currently create:

```yaml
sources: {}
skills: []
```
