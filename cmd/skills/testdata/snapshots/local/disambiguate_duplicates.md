# Local Snapshot Suites

## Duplicate Skill Names Fail Without A Selector

`repo/repo-one/plugins/x/bedrock/SKILL.md`:
```md
# bedrock (plugin copy)
```

`repo/repo-one/skills/bedrock/SKILL.md`:
```md
# bedrock
```

```repo repo-one
commit "initial"
```

`project/.agents/manifest.yaml`:
```yaml
sources:
  repo-one:
    url: {{repo:repo-one}}
    ref: main
skills:
  - source: repo-one
    name: bedrock
```

<!-- exit: 1 -->
```command
skills sync --verbose
```

```stdout-assert
```

```stderr
```

```command
skills status --verbose
```

```stdout-assert
[Skills]
repo-one bedrock ambiguous-skill <project>/.agents/skills/bedrock - multiple skills share this directory name; set path: to one of: plugins/x/bedrock, skills/bedrock
```

```stderr
```

## Exclude Scope Syncs One Copy

`repo/repo-one/plugins/x/bedrock/SKILL.md`:
```md
# bedrock (plugin copy)
```

`repo/repo-one/skills/bedrock/SKILL.md`:
```md
# bedrock
```

```repo repo-one
commit "initial"
```

`project/.agents/manifest.yaml`:
```yaml
sources:
  repo-one:
    url: {{repo:repo-one}}
    ref: main
    exclude: [plugins]
skills:
  - source: repo-one
    name: bedrock
```

```command
skills sync --verbose
```

```stdout-assert
[Sources]
repo-one resolved main <sha> - <project>/.agents/cache/repos/repo-one-<sha> <project>/.agents/cache/worktrees/project-<sha>/repo-one/<sha> -

[Skills]
repo-one bedrock created <project>/.agents/skills/bedrock <project>/.agents/cache/worktrees/project-<sha>/repo-one/<sha>/skills/bedrock -

[Claude]
repo-one bedrock created <project>/.claude/skills/bedrock <project>/.agents/skills/bedrock -
```

```stderr
```

## Path Selectors Sync Both Copies Under Distinct Names

`repo/repo-one/plugins/x/bedrock/SKILL.md`:
```md
# bedrock (plugin copy)
```

`repo/repo-one/skills/bedrock/SKILL.md`:
```md
# bedrock
```

```repo repo-one
commit "initial"
```

`project/.agents/manifest.yaml`:
```yaml
sources:
  repo-one:
    url: {{repo:repo-one}}
    ref: main
skills:
  - source: repo-one
    name: bedrock
    path: skills/bedrock
  - source: repo-one
    name: bedrock-plugin
    path: plugins/x/bedrock
```

```command
skills sync --verbose
```

```stdout-assert
[Sources]
repo-one resolved main <sha> - <project>/.agents/cache/repos/repo-one-<sha> <project>/.agents/cache/worktrees/project-<sha>/repo-one/<sha> -

[Skills]
repo-one bedrock created <project>/.agents/skills/bedrock <project>/.agents/cache/worktrees/project-<sha>/repo-one/<sha>/skills/bedrock -
repo-one bedrock-plugin created <project>/.agents/skills/bedrock-plugin <project>/.agents/cache/worktrees/project-<sha>/repo-one/<sha>/plugins/x/bedrock -

[Claude]
repo-one bedrock created <project>/.claude/skills/bedrock <project>/.agents/skills/bedrock -
repo-one bedrock-plugin created <project>/.claude/skills/bedrock-plugin <project>/.agents/skills/bedrock-plugin -
```

```stderr
```

```command
skills status --verbose
```

```stdout-assert
[Skills]
repo-one bedrock linked <project>/.agents/skills/bedrock <project>/.agents/cache/worktrees/project-<sha>/repo-one/<sha>/skills/bedrock -
repo-one bedrock-plugin linked <project>/.agents/skills/bedrock-plugin <project>/.agents/cache/worktrees/project-<sha>/repo-one/<sha>/plugins/x/bedrock -
```

```stderr
```

## Skill List --all Shows The Scope Column

`repo/repo-one/plugins/x/bedrock/SKILL.md`:
```md
# bedrock (plugin copy)
```

`repo/repo-one/skills/bedrock/SKILL.md`:
```md
# bedrock
```

`repo/repo-one/skills/lambda/SKILL.md`:
```md
# lambda
```

```repo repo-one
commit "initial"
```

`project/.agents/manifest.yaml`:
```yaml
sources:
  repo-one:
    url: {{repo:repo-one}}
    ref: main
    exclude: [plugins]
skills: []
```

```command
skills source sync
```

```stdout-assert
[Source Sync]
cloned repo-one main@<sha> main@<sha>
```

```stderr
```

```command
skills skill list
```

```stdout-assert
[Skills]
Source Name Path
repo-one bedrock skills/bedrock
repo-one lambda skills/lambda
```

```stderr
```

```command
skills skill list --all
```

```stdout-assert
[Skills]
Source Name Path Scope
repo-one bedrock plugins/x/bedrock excluded
repo-one bedrock skills/bedrock included
repo-one lambda skills/lambda included
```

```stderr
```
