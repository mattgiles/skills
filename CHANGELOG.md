# Changelog

All notable changes to `skills` are documented in this file.

## [Unreleased]

### Added

- Per-source discovery scope: `sources.<alias>.include` and
  `sources.<alias>.exclude` lists of repo-relative directory prefixes filter
  discovered skills before link resolution (exclude wins). One line such as
  `exclude: [plugins]` fixes a source that mirrors every skill under two trees.
  `skills source add` gains repeatable `--include`/`--exclude` flags and keeps
  the manifest's existing scope when the flags are omitted.
- Per-skill `path` selector: `skills[].path` picks a discovered skill by its
  exact repo-relative directory, while `name` stays the link directory. This
  disambiguates same-named skills, installs both copies under distinct names,
  or renames a single skill's link. `skills add` gains `--path <dir>`, which
  appends a new entry or updates an existing `(source, name)` entry in place.
- `skills skill list --all` bypasses the source scope and adds a `Scope`
  column (`included`/`excluded`).
- `ambiguous-skill` messages now list the candidate paths and the doctor hint
  suggests `skills add --path` or scoping the source; `missing-skill` messages
  explain when a `path` is absent, excluded by scope, or exists only outside
  the scope.

### Fixed

- Re-registering an existing block-style source with `skills source add` no
  longer produces invalid YAML indentation in the manifest.

## [0.6.0] - 2026-08-12

### Added

- Support committed `.agents/manifest.d/*.yaml` fragments that are merged with
  `.agents/manifest.yaml` for project reads, allowing tools to contribute
  sources and skills without rewriting the main manifest.

### Fixed

- Recover status and sync after a source URL is repointed when the previously
  resolved commit is unavailable in the new repository.
