# Doctor Reference

## Command

```text
skills doctor
skills doctor --global
```

`skills doctor` runs a read-only diagnostic pass for the active scope and exits with:

- `0` when no doctor errors were found
- `3` when doctor findings include errors
- `1` for other runtime failures

## Scope

Default behavior:

- in a Git repo, inspect the current project workspace
- outside a Git repo, use `skills doctor --global`

Global mode inspects the shared config and shared home workspace.

## Output Sections

Doctor prints these sections in order:

- `ENVIRONMENT`
- `CONFIG`
- `WORKSPACE`
- `GIT`
- `SOURCES`
- `SKILLS`
- `CLAUDE`
- `HINTS`

Each non-hints section prints either:

- `INFO  ok  no issues found`
- or a table of findings

At the end, doctor prints a summary line:

```text
doctor: <errors> errors, <warnings> warnings
```

## Common Finding Codes

Configuration and workspace:

- `config-missing`
- `config-parse-failed`
- `config-invalid`
- `local-config-missing`
- `local-config-parse-failed`
- `project-cache-mode`
- `project-cache-roots`
- `manifest-missing`
- `manifest-parse-failed`
- `state-parse-failed`

Git and ownership:

- `git-missing`
- `git-unavailable`
- `git-repo-not-found`
- `ignore-rules-missing`
- `tracked-managed-path`

Source and link health:

- `missing-source`
- `invalid-source`
- `invalid-ref`
- `source-not-ready`
- `missing-skill`
- `ambiguous-skill`
- `conflict`
- `stale-managed-link`

Skipped checks:

- `not-checked`

## Verbose Output

```bash
skills --verbose doctor
```

Verbose mode adds a `DETAILS` column with path, target, or ref details when available.

## Relationship To Other Commands

- use `doctor` when you want diagnosis and hints
- use `status` when you want current declared-vs-installed state
- use `sync` and `update` when you want to change state

For fix-oriented guidance, see [Troubleshoot Common Failures](../how-to/troubleshoot-common-failures.md).
