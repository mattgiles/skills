# Refactor Plan

## Summary

This codebase is functionally healthy today:

- `go test ./...` passes
- `go vet ./...` passes

The main refactor pressure is structural rather than correctness-driven. The largest hotspots are:

- `cmd/skills/root.go`
- `internal/project/project.go`
- `internal/doctor/doctor.go`

Across the audit lenses from `golang-cli`, `golang-code-style`, `golang-context`, and `golang-structs-interfaces`, the recommended direction is consistent:

- reduce command-layer duplication
- propagate a single top-level context through the CLI
- split oversized orchestration packages by concern
- stay concrete-first and avoid speculative interfaces

This document is a planning artifact only. It does not change the CLI surface, manifest schema, config schema, or runtime behavior on its own.

## Current Audit Findings

### `golang-cli`

The CLI already gets several important basics right:

- `SilenceUsage: true` and `SilenceErrors: true` are set on the root command
- command output generally uses `cmd.OutOrStdout()` / `cmd.ErrOrStderr()`
- tests exercise command behavior directly

The main CLI debt is orchestration duplication:

- repo and `--global` handling is repeated across `status`, `sync`, `update`, `doctor`, and parts of source/skill flows
- command handlers resolve working scope, config, summaries, and output rendering inline instead of through shared execution helpers
- `cmd/skills/root.go` has accumulated unrelated command families and helper functions

Recommendations:

- introduce shared command execution helpers that resolve scope once and then dispatch repo/global work through a single path
- move command-family code out of `cmd/skills/root.go` into smaller files grouped by responsibility
- standardize command handler shape so each command only does argument parsing, scope resolution, and one operation call
- add a top-level root context strategy based on `ExecuteContext` and signal-aware startup once the command layer is split cleanly
- document an exit-code policy after structural cleanup; do not block the refactor on that work

### `golang-code-style`

The strongest style issue is not formatting; it is concentration of too many responsibilities in a few files:

- `internal/project/project.go` mixes manifest/state IO, workspace resolution, sync/update/status pipelines, link planning, worktree operations, and gitignore ownership
- `internal/doctor/doctor.go` mixes report modeling, config loading, project inspection, global inspection, and finding synthesis
- `cmd/skills/root.go` still acts as both root command assembly and a catch-all for multiple command families

Recommendations:

- split `internal/project` by behavior, not by arbitrary size targets
- split `internal/doctor` into smaller concern-oriented files
- replace repeated inline branching with named helpers for common repo/global patterns
- prefer smaller focused functions and concrete helper types over broad “manager” or “service” wrappers
- keep behavior-local data transformations near the workflow that owns them instead of accumulating generic utility helpers

### `golang-context`

Context usage is partially good in internal packages, but weak at the CLI boundary:

- command handlers frequently create `context.Background()` inline instead of propagating a single process-level context
- helper paths such as scope resolution and ownership inspection still create their own background contexts internally
- the code is not currently structured around `cmd.Context()`

Recommendations:

- create one top-level context in `main` and execute the root command with `ExecuteContext`
- use `cmd.Context()` in all command handlers
- thread `ctx context.Context` through helpers that currently call `context.Background()` internally
- restrict `context.Background()` to top-level entry points and tests
- do not introduce detached background work during this refactor; the goal is context propagation, not concurrency expansion

### `golang-structs-interfaces`

The repo is already concrete-first, which is the right default for this codebase:

- there are almost no production interfaces
- most state is represented through concrete structs
- constructors and workflow entry points mostly return concrete values

The type-design issue is breadth of state bundles, not lack of interfaces:

- `workspace` and `resolvedSource` currently carry multiple workflow concerns at once
- several internal functions accept broad state objects because package boundaries are too coarse
- large files make it harder to see which structs are true DTOs versus orchestration-only state

Recommendations:

- do not introduce interfaces just to break up large files
- keep manifest/state/report structs concrete unless a real consumer boundary requires abstraction
- narrow workflow-specific structs where a type currently bundles unrelated concerns
- introduce smaller concrete operation inputs when it improves readability of sync/update/status pipelines
- define interfaces only at actual consumer boundaries that need substitution or testing seams

## Public Interfaces And Compatibility

This refactor should preserve external behavior:

- all existing CLI commands and flags remain unchanged
- manifest and config file schemas remain unchanged
- output semantics should remain stable unless a later, explicit CLI UX change is proposed

Likely signature changes are internal-only:

- more helpers will take `ctx context.Context`
- repo/global command execution may move behind shared concrete helper functions
- internal packages may be reorganized into smaller files without changing exported behavior

Explicit non-goals:

- no new cross-package interfaces by default
- no new configuration layering model
- no machine-readable output redesign in this refactor
- no behavioral changes to sync, update, doctor, add, or source discovery semantics

## Phased Roadmap

### Phase 1: CLI Boundary Cleanup

Goal:
Make command handlers thinner, reduce repo/global duplication, and establish one propagated command context.

Changes:

- update `main` to create a signal-aware top-level context and run the root command with `ExecuteContext`
- convert command handlers to use `cmd.Context()` instead of creating `context.Background()`
- extract shared repo/global execution helpers for command families that currently branch inline
- move command-family implementations out of `cmd/skills/root.go` so root construction only wires commands together
- keep rendering helpers close to their command family unless a renderer is genuinely shared

Keep unchanged:

- command names
- flags
- output format
- error messages except where context plumbing requires minor wrapping cleanup

Validation:

- `go test ./...`
- `go vet ./...`
- focused regression on `status`, `sync`, `update`, `doctor`, `source`, `skill`, and `add`

### Phase 2: `internal/project` Decomposition

Goal:
Split the current monolithic project package into smaller files organized by stable behavior boundaries without changing semantics.

Changes:

- isolate manifest, state, and local-config load/save/validation logic
- isolate workspace resolution and path derivation
- isolate sync, update, and status orchestration entry points
- isolate source resolution and worktree preparation logic
- isolate link planning, link application, and stale-link pruning
- isolate project ownership and gitignore management helpers

Recommended file grouping:

- manifest/state/config IO
- workspace resolution
- sync/update/status workflows
- source resolution and commit selection
- link planning and link mutation
- gitignore and ownership checks

Type guidance:

- keep `Manifest`, `State`, `SourceReport`, `LinkReport`, and related wire/data structs concrete
- narrow `workspace` and `resolvedSource` if parts of their state are only relevant to one workflow stage
- prefer explicit operation inputs over adding generic coordinating structs

Keep unchanged:

- package name
- exported function behavior
- persisted file formats

Validation:

- `go test ./...`
- `go vet ./...`
- extra regression focus on sync/update/status parity, state writes, and link creation/pruning

### Phase 3: `internal/doctor` Decomposition

Goal:
Separate report modeling from inspection flows so doctor logic is easier to change without re-reading a thousand-line file.

Changes:

- isolate report model and summary helpers
- isolate config loading and validation
- isolate project workspace inspection
- isolate global workspace inspection
- isolate translation from project status reports into doctor findings
- isolate reusable finding helpers and skipped-section handling

Keep unchanged:

- doctor scope model
- finding codes and general output semantics
- relationship between doctor and `internal/project`

Validation:

- `go test ./...`
- `go vet ./...`
- targeted regression on doctor output for healthy, missing-manifest, malformed-config, and stale-link scenarios

### Phase 4: Follow-On Polish

Goal:
Finish the structural cleanup with small follow-on improvements that become easier once the packages are decomposed.

Changes:

- document an explicit exit-code policy for the CLI
- decide whether command renderers need shared abstractions after the split; default answer should remain “no”
- add package comments or short file-level docs only where the new structure needs orientation help
- consider machine-readable output only as a separate, later feature once command flow is simpler

Keep unchanged:

- functional behavior
- current concrete-first package design

Validation:

- `go test ./...`
- `go vet ./...`

## Implementation Rules

The refactor should follow these guardrails:

- prefer concrete helper functions and concrete types over new interfaces
- define any interface only where a consumer actually needs substitution
- use early-return control flow to reduce orchestration nesting as files are split
- keep `ctx context.Context` as the first parameter on internal operations that participate in command execution
- avoid moving unrelated behavior together merely to hit file-size targets
- preserve stdout versus stderr behavior in all commands

## Acceptance Criteria

The refactor is complete when:

- oversized files have been decomposed into concern-based units
- command handlers consistently propagate a single root context
- repo/global command execution duplication is substantially reduced
- `internal/project` and `internal/doctor` are easier to navigate without introducing abstraction-heavy indirection
- CLI behavior, config behavior, manifest behavior, and tests remain stable

## Validation Checklist

Run after each phase:

- `go test ./...`
- `go vet ./...`

Review explicitly after each phase:

- repo mode versus `--global` parity
- sync, update, status, doctor, source, skill, and add behavior
- manifest, state, and config load/save behavior
- stdout versus stderr discipline
- error wrapping and user-facing message stability

## Assumptions

- this plan intentionally targets maintainability, context correctness, and package structure rather than new product behavior
- the current repo health means structural refactor work can be phased safely behind existing tests
- speculative performance work is out of scope
- interface extraction is out of scope unless a real consumer boundary appears during implementation
