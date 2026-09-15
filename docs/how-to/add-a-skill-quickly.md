# Add A Skill Quickly

Use `skills add` when you already know the source alias and skill name you want. It updates the manifest and immediately syncs the result.

## Add A Skill From An Existing Source

```bash
skills add repo-one analytics
```

This:

- adds the `(repo-one, analytics)` skill declaration if it is not already present
- runs sync for the current scope
- creates or updates the canonical and Claude links if the source resolves successfully

## Add A Skill And Create The Source At The Same Time

```bash
skills add repo-one analytics --url git@github.com:example/repo-one.git --ref main
```

Use this when the source alias is not yet declared.

If you omit `--ref`, `skills` infers the remote default branch:

```bash
skills add repo-one analytics --url git@github.com:example/repo-one.git
```

## Add A Skill In Shared Home Scope

```bash
skills add --global repo-one analytics --url git@github.com:example/repo-one.git --ref main
```

This writes to the shared home manifest and installs into the shared home directories.

## Disambiguate Duplicate Skill Names

If a source contains two directories with the same name that both hold a
`SKILL.md`, `skills add repo-one bedrock` fails with `ambiguous-skill` and
lists the candidates:

```text
repo-one/bedrock: ambiguous-skill (multiple skills share this directory name; set path: to one of: plugins/x/bedrock, skills/bedrock)
```

First see every copy (the `Path` column is what you will select on):

```bash
skills skill list --source repo-one --all
```

Then pick one of two fixes.

**Scope the source** so only one copy is discovered. This fixes every duplicate
in the source at once:

```bash
skills source add repo-one <url> --exclude plugins
skills add repo-one bedrock
```

**Pin this skill to a path.** The skill name stays the link directory:

```bash
skills add repo-one bedrock --path skills/bedrock
```

This writes `path: skills/bedrock` on the entry. Running it again with the same
`--path` is a no-op; running it with a different `--path` updates the entry in
place and re-syncs.

### Install both copies under distinct names

Because `--path` selects the directory and the skill name is only the link
label, you can install both copies side by side:

```bash
skills add repo-one bedrock --path skills/bedrock
skills add repo-one bedrock-plugin --path plugins/x/bedrock
```

The same trick renames a single skill: `skills add repo-one my-bedrock --path
skills/bedrock` links it at `.agents/skills/my-bedrock`.

## Understand No-Op Behavior

If the same `(source, skill)` pair is already declared, the command prints a message and exits successfully:

```text
skill "analytics" from source "repo-one" is already declared
```

The same applies when `--path` is passed and matches the entry's existing `path`.

## Verify The Result

Run:

```bash
skills status
```

Or in home scope:

```bash
skills status --global
```

For exact flags and command semantics, see [CLI Reference](../reference/cli.md).
