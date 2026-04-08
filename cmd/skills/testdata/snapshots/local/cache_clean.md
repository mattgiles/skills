# Local Cache Clean Snapshot Suites

## Cleans Local Cache Roots

`repo/repo-one/analytics/SKILL.md`:
```md
# analytics
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
    name: analytics
```

```command
skills sync --verbose
```

```stdout-assert
[Workspace]
Scope repo
Root <project>
Installs <project>/.agents/skills
Cache local
Worktrees <project>/.agents/cache/worktrees
Repos <project>/.agents/cache/repos
```

```stderr
```

```command
skills cache clean
```

```stdout-assert
[Workspace]
Scope repo
Root <project>
Installs <project>/.agents/skills
Cache local
Worktrees <project>/.agents/cache/worktrees

[Cache Clean]
Repos <project>/.agents/cache/repos
Worktrees <project>/.agents/cache/worktrees
```

```stderr
```
