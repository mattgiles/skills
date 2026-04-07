# Project Manifest Reference

## File Name

```text
.skills.yaml
```

## Schema

```yaml
sources:
  repo-one:
    url: git@github.com:example/repo-one.git
    ref: main

agents:
  codex:
    skills_dir: ./agent-skills

skills:
  - source: repo-one
    name: analytics
    agents: [codex]
```

## Top-Level Fields

| Field | Type | Required | Meaning |
| --- | --- | --- | --- |
| `sources` | map | yes in practice | Project source declarations |
| `agents` | map | no | Project-level agent overrides |
| `skills` | list | yes in practice | Declared skills to install |

## `sources.<alias>`

| Field | Type | Required | Meaning |
| --- | --- | --- | --- |
| `url` | string | no | Project-local source URL override |
| `ref` | string | yes | Branch, tag, or commit to resolve |

Notes:

- alias validation uses the same rules as global config aliases
- `ref` must not be empty
- project source aliases must exist before any skill can reference them

## `agents.<name>`

| Field | Type | Required | Meaning |
| --- | --- | --- | --- |
| `skills_dir` | string | yes | Project-local agent destination root |

Notes:

- project agent overrides are optional
- when present, they override the matching global agent root
- relative paths are resolved from the project directory

## `skills[]`

| Field | Type | Required | Meaning |
| --- | --- | --- | --- |
| `source` | string | yes | Source alias |
| `name` | string | yes | Skill directory name |
| `agents` | list of strings | yes | Agents that should receive this skill |

Validation rules:

- `source` must not be empty
- `name` must not be empty
- `agents` must contain at least one entry
- the same `(source, name)` pair cannot appear more than once
- a skill cannot repeat the same agent name within one entry

## Default Manifest

`skills project init` currently creates:

```yaml
sources: {}
skills: []
```

The `agents` map is omitted until you add overrides.
