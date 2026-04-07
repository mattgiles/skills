# Use Local Agent Directories

Project-level agent overrides are useful for testing because they keep symlinks inside the project instead of writing to real agent homes.

## Set A Project-Local Skills Directory

In `.skills.yaml`:

```yaml
agents:
  codex:
    skills_dir: ./agent-skills
```

Relative paths are resolved from the project directory.

## Sync The Project

```bash
skills project sync
```

The symlink path will be:

```text
./agent-skills/<skill-name>
```

## Inspect The Links

```bash
ls -l ./agent-skills
```

This is the recommended setup for local testing and tutorials because it avoids changing `~/.codex/skills` or `~/.claude/skills`.

For how project overrides interact with global config, see [Config Vs Project](../explanation/config-vs-project.md).
