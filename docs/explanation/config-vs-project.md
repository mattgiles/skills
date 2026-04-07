# Config Vs Project

`skills` separates machine concerns, shared home concerns, and project concerns.

## Global Config

Global config owns machine-level defaults:

- where canonical repos live
- where worktrees live
- where shared home skills live
- where shared home Claude adapters live
- which source aliases are already registered

## Project Manifest

`.agents/manifest.yaml` owns project intent:

- which sources the project needs
- which ref each source should resolve
- which skills should exist in the project’s canonical `.agents/skills`

## Home Manifest

`~/.agents/manifest.yaml` owns shared home intent:

- which sources should be available at home scope
- which shared skills should exist in `~/.agents/skills`

## Why Keep Them Separate

This keeps clone storage and worktree storage machine-local, while keeping actual installed skill sets isolated by scope.

Project syncs should not mutate shared home installs. Home syncs should not mutate project installs. Both scopes can still reuse the same canonical clone and worktree backend.
