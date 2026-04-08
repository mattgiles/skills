# Release A Version

This project uses a tag-driven GitHub Releases workflow.

Merging to `main` does not publish a release by itself. A release is created when you push a semver tag such as `v0.1.0`.

## Prerequisites

- your release commit is already on `main`
- CI is green on that commit
- you have permission to push tags to the repository

## 1. Update `main`

```bash
git checkout main
git pull
```

Make sure `HEAD` is the exact commit you want to release.

## 2. Run The Release Recipe

```bash
just release
```

By default, `just release` creates the next minor version tag and pushes only that tag to `origin`.

Examples:

- `v0.1.3` becomes `v0.2.0`
- if no release tags exist yet, the first minor release becomes `v0.1.0`

For a major bump:

```bash
just release major
```

Examples:

- `v0.1.3` becomes `v1.0.0`
- if no release tags exist yet, the first major release becomes `v1.0.0`

The release workflow only triggers on pushed tags matching `v*`.

## 3. Understand The Safety Checks

`just release` refuses to continue unless all of these are true:

- the current branch is `main`
- the upstream branch is `origin/main`
- the worktree is clean
- local `HEAD` matches `origin/main`
- the computed release tag does not already exist

The recipe fetches `origin/main` and tags before it computes the next version.

It creates an annotated tag using the tag name as the annotation message, then runs:

```bash
git push origin <tag>
```

If the push fails after tag creation, the tag remains present locally and is not deleted automatically.

## 4. Wait For The Release Workflow

GitHub Actions runs the release workflow in `.github/workflows/release.yml`.

That workflow uses GoReleaser and `.goreleaser.yaml` to:

- build macOS binaries for `darwin/amd64` and `darwin/arm64`
- package release archives
- generate `skills_checksums.txt`
- create the GitHub Release
- upload the artifacts

## 5. Verify The GitHub Release

After the workflow finishes, verify the release contains:

- `skills_<version>_darwin_amd64.tar.gz`
- `skills_<version>_darwin_arm64.tar.gz`
- `skills_checksums.txt`

## 6. Verify The Public Installer

Install the new version directly from the public installer:

```bash
curl -fsSL https://raw.githubusercontent.com/mattgiles/skills/main/scripts/install.sh | VERSION=v0.1.0 sh
```

Then verify the installed binary:

```bash
skills version
```

## Notes

- pushing a feature branch does not create a release
- merging to `main` does not create a release
- pushing a `v...` tag is what publishes a release
- `just release` pushes the tag only; it does not push `main`
