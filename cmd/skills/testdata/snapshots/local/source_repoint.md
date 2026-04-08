# Local Source Repoint Snapshot Suites

## Add Repoints A Source To A New Repository Identity

`repo/vercel-legacy/skills/find-skills/SKILL.md`:
```md
# find-skills
```

```repo vercel-legacy
commit "initial"
```

`repo/vercel-agent-skills/skills/react-best-practices/SKILL.md`:
```md
# react-best-practices
```

```repo vercel-agent-skills
commit "initial"
```

`project/.agents/manifest.yaml`:
```yaml
sources:
  vercel:
    url: {{repo:vercel-legacy}}
    ref: main
skills: []
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
vercel resolved main <sha> - <project>/.agents/cache/repos/vercel-legacy-<sha> <project>/.agents/cache/worktrees/project-<sha>/vercel/<sha> -
```

```stderr
```

`project/.agents/manifest.yaml`:
```yaml
sources:
  vercel:
    url: {{repo:vercel-agent-skills}}
    ref: main
skills: []
```

```command
skills add vercel react-best-practices
```

```stdout-assert
[Workspace]
Scope repo
Root <project>
Installs <project>/.agents/skills
Cache local
Worktrees <project>/.agents/cache/worktrees

[Sources]
vercel up-to-date main <sha> - 

[Skills]
vercel react-best-practices created <project>/.agents/skills/react-best-practices -

[Claude]
vercel react-best-practices created <project>/.claude/skills/react-best-practices -
```

```stderr
```
