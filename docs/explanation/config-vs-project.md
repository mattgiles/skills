# Config Vs Project

`skills` separates machine concerns, shared home concerns, and project concerns.

## Global Config

Global config owns machine-level shared defaults:

- where shared/home canonical repos live
- where shared/home worktrees live
- where shared home skills live
- where shared home Claude adapters live
- which source aliases are already registered

## Project Manifest

`.agents/manifest.yaml` owns project intent:

- which sources the project needs
- which ref each source should resolve
- which URLs those project sources come from
- which skills should exist in the project’s canonical `.agents/skills`
- project-local clone/worktree storage under `.agents/cache`

## Home Manifest

`~/.agents/manifest.yaml` owns shared home intent:

- which sources should be available at home scope
- which shared skills should exist in `~/.agents/skills`

## Why Keep Them Separate

This keeps project workflows self-contained, while still allowing a separate shared home workflow.

Project syncs should not mutate shared home installs. Home syncs should not mutate project installs. Project mode uses repo-local storage; home mode uses shared machine-level storage.
