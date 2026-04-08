# Config Vs Project

`skills` separates machine concerns, shared home concerns, and project concerns.

## Global Config

Global config owns machine-level shared defaults:

- where shared/home canonical repos live
- where shared/home worktrees live
- where shared home skills live
- where shared home Claude adapters live
- default shared install and storage paths for home scope

## Project Manifest

`.agents/manifest.yaml` owns project intent:

- which sources the project needs
- which ref each source should resolve
- which URLs those project sources come from
- which skills should exist in the project’s canonical `.agents/skills`

`.agents/local.yaml` owns repo-local user preference:

- whether this repo user wants `cache.mode: local`
- or `cache.mode: global`

## Home Manifest

`~/.agents/manifest.yaml` owns shared home intent:

- which sources should be available at home scope
- which shared skills should exist in `~/.agents/skills`

## Why Keep Them Separate

This keeps project workflows self-contained, while still allowing a separate shared home workflow.

Project syncs should not mutate shared home installs. Home syncs should not mutate project installs. Project installs are always repo-local, while cache storage can be repo-local or shared depending on the repo user's local settings.
