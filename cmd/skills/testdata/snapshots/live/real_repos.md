# Live Real-World Repos

## Syncs Dagster Expert From Dagster Skills

`project/.agents/manifest.yaml`:
```yaml
sources:
  dagster:
    url: https://github.com/dagster-io/skills
    ref: master
skills:
  - source: dagster
    name: dagster-expert
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

[Sources]
dagster resolved master <sha> - <project>/.agents/cache/repos/dagster <project>/.agents/cache/worktrees/project-<sha>/dagster/<sha> -

[Skills]
dagster dagster-expert created <project>/.agents/skills/dagster-expert <project>/.agents/cache/worktrees/project-<sha>/dagster/<sha>/skills/dagster-expert/skills/dagster-expert -

[Claude]
dagster dagster-expert created <project>/.claude/skills/dagster-expert <project>/.agents/skills/dagster-expert -
```

```stderr
```

```command
skills status --verbose
```

```stdout-assert
[Workspace]
Scope repo
Root <project>
Installs <project>/.agents/skills
Cache local
Worktrees <project>/.agents/cache/worktrees
Repos <project>/.agents/cache/repos

[Sources]
dagster up-to-date master <sha> <sha> <project>/.agents/cache/repos/dagster <project>/.agents/cache/worktrees/project-<sha>/dagster/<sha> -

[Skills]
dagster dagster-expert linked <project>/.agents/skills/dagster-expert <project>/.agents/cache/worktrees/project-<sha>/dagster/<sha>/skills/dagster-expert/skills/dagster-expert -

[Claude]
dagster dagster-expert linked <project>/.claude/skills/dagster-expert <project>/.agents/skills/dagster-expert -
```

```stderr
```

## Syncs Find Skills From Vercel Skills

`project/.agents/manifest.yaml`:
```yaml
sources:
  vercel:
    url: https://github.com/vercel-labs/skills
    ref: main
skills:
  - source: vercel
    name: find-skills
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

[Sources]
vercel resolved main <sha> - <project>/.agents/cache/repos/vercel <project>/.agents/cache/worktrees/project-<sha>/vercel/<sha> -

[Skills]
vercel find-skills created <project>/.agents/skills/find-skills <project>/.agents/cache/worktrees/project-<sha>/vercel/<sha>/skills/find-skills -

[Claude]
vercel find-skills created <project>/.claude/skills/find-skills <project>/.agents/skills/find-skills -
```

```stderr
```

## Syncs Root Skill From Terraform Skill

`project/.agents/manifest.yaml`:
```yaml
sources:
  terraform-skill:
    url: https://github.com/antonbabenko/terraform-skill
    ref: master
skills:
  - source: terraform-skill
    name: terraform-skill
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

[Sources]
terraform-skill resolved master <sha> - <project>/.agents/cache/repos/terraform-skill <project>/.agents/cache/worktrees/project-<sha>/terraform-skill/<sha> -

[Skills]
terraform-skill terraform-skill created <project>/.agents/skills/terraform-skill <project>/.agents/cache/worktrees/project-<sha>/terraform-skill/<sha> -

[Claude]
terraform-skill terraform-skill created <project>/.claude/skills/terraform-skill <project>/.agents/skills/terraform-skill -
```

```stderr
```

```command
skills status --verbose
```

```stdout-assert
[Workspace]
Scope repo
Root <project>
Installs <project>/.agents/skills
Cache local
Worktrees <project>/.agents/cache/worktrees
Repos <project>/.agents/cache/repos

[Sources]
terraform-skill up-to-date master <sha> <sha> <project>/.agents/cache/repos/terraform-skill <project>/.agents/cache/worktrees/project-<sha>/terraform-skill/<sha> -

[Skills]
terraform-skill terraform-skill linked <project>/.agents/skills/terraform-skill <project>/.agents/cache/worktrees/project-<sha>/terraform-skill/<sha> -

[Claude]
terraform-skill terraform-skill linked <project>/.claude/skills/terraform-skill <project>/.agents/skills/terraform-skill -
```

```stderr
```
