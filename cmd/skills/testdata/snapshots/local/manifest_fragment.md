# Local Snapshot Suites

## Project Merges A Manifest Fragment

`repo/repo-one/analytics/SKILL.md`:
```md
# analytics
```

`repo/repo-one/reporting/SKILL.md`:
```md
# reporting
```

```repo repo-one
commit "initial"
```

`repo/repo-two/lint/SKILL.md`:
```md
# lint
```

```repo repo-two
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

`project/.agents/manifest.d/perk.yaml`:
```yaml
sources:
  repo-two:
    url: {{repo:repo-two}}
    ref: main
skills:
  - source: repo-one
    name: reporting
  - source: repo-two
    name: lint
```

```command
skills sync --verbose
```

```stdout-assert
[Sources]
repo-one resolved main <sha> - <project>/.agents/cache/repos/repo-one-<sha> <project>/.agents/cache/worktrees/project-<sha>/repo-one/<sha> -
repo-two resolved main <sha> - <project>/.agents/cache/repos/repo-two-<sha> <project>/.agents/cache/worktrees/project-<sha>/repo-two/<sha> -

[Skills]
repo-one analytics created <project>/.agents/skills/analytics <project>/.agents/cache/worktrees/project-<sha>/repo-one/<sha>/analytics -
repo-one reporting created <project>/.agents/skills/reporting <project>/.agents/cache/worktrees/project-<sha>/repo-one/<sha>/reporting -
repo-two lint created <project>/.agents/skills/lint <project>/.agents/cache/worktrees/project-<sha>/repo-two/<sha>/lint -

[Claude]
repo-one analytics created <project>/.claude/skills/analytics <project>/.agents/skills/analytics -
repo-one reporting created <project>/.claude/skills/reporting <project>/.agents/skills/reporting -
repo-two lint created <project>/.claude/skills/lint <project>/.agents/skills/lint -
```

```stderr
```

```command
skills status --verbose
```

```stdout-assert
[Sources]
repo-one up-to-date main <sha> <sha> <project>/.agents/cache/repos/repo-one-<sha> <project>/.agents/cache/worktrees/project-<sha>/repo-one/<sha> -
repo-two up-to-date main <sha> <sha> <project>/.agents/cache/repos/repo-two-<sha> <project>/.agents/cache/worktrees/project-<sha>/repo-two/<sha> -

[Skills]
repo-one analytics linked <project>/.agents/skills/analytics <project>/.agents/cache/worktrees/project-<sha>/repo-one/<sha>/analytics -
repo-one reporting linked <project>/.agents/skills/reporting <project>/.agents/cache/worktrees/project-<sha>/repo-one/<sha>/reporting -
repo-two lint linked <project>/.agents/skills/lint <project>/.agents/cache/worktrees/project-<sha>/repo-two/<sha>/lint -

[Claude]
repo-one analytics linked <project>/.claude/skills/analytics <project>/.agents/skills/analytics -
repo-one reporting linked <project>/.claude/skills/reporting <project>/.agents/skills/reporting -
repo-two lint linked <project>/.claude/skills/lint <project>/.agents/skills/lint -
```

```stderr
```
