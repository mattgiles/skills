# Config Vs Project

`skills` separates machine concerns from project concerns.

## Global Config

Global config owns machine-level defaults:

- where canonical repos live
- where worktrees live
- which agent roots exist on this machine
- which source aliases are already registered

## Project Manifest

`.skills.yaml` owns project intent:

- which sources the project needs
- which ref each source should resolve
- which agents should receive which skills
- any project-local agent root overrides

## Why Keep Them Separate

This keeps the project manifest portable. A repository can declare the skills it depends on without assuming that every machine has the same global filesystem layout.

It also lets local users override agent destinations for testing while preserving the project's dependency declaration.
