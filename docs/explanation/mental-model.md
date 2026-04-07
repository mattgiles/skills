# Mental Model

The project works best when you keep four ideas separate:

- a Git repository is a source of skills
- a directory containing `SKILL.md` is a skill
- a project manifest declares which skills a project wants
- sync installs those skills into agent directories by symlink

That separation matters because one repo can contain many skills, and multiple projects can depend on the same repo at different refs.

## Why The Model Is Structured This Way

The CLI is intentionally built around `(source, skill)` pairs rather than around copying files into dot-directories.

That gives the project:

- provenance: each installed skill still comes from a known repo and ref
- reuse: one canonical source clone can serve many projects
- reproducibility: project state records the resolved commit
- low churn: installs are symlinks, not duplicated copies

This is the core lens to keep in mind while reading the rest of the docs.
