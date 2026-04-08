# Current Scope And Roadmap

The codebase already supports the main v1 workflow:

- initialize global config
- register and sync sources
- add a skill directly with one command
- discover skills
- create a project manifest
- sync project or shared-home skills into canonical install roots
- record and update resolved commits
- preview updates and syncs with dry-run mode
- run doctor checks across config, workspace, sources, and managed links
- generate shell completion scripts

## What Is Current Behavior

The reference and how-to sections in this documentation describe the current implementation and are derived from code and tests.

## What Is Still Roadmap Material

There is no separate checked-in roadmap document in this repository today.

Treat these as likely future-direction topics rather than current promises:

- additional machine-readable output modes
- more installer and platform coverage beyond the current public macOS release flow
- higher-level ergonomics on top of the current source, manifest, and sync model

If design notes, issue discussions, or old planning text disagree with the code, the code and tests win for user-facing docs.
