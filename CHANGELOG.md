# Changelog

All notable changes to `skills` are documented in this file.

## [Unreleased]

## [0.6.0] - 2026-08-12

### Added

- Support committed `.agents/manifest.d/*.yaml` fragments that are merged with
  `.agents/manifest.yaml` for project reads, allowing tools to contribute
  sources and skills without rewriting the main manifest.

### Fixed

- Recover status and sync after a source URL is repointed when the previously
  resolved commit is unavailable in the new repository.
