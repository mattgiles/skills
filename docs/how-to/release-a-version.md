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

## 2. Create And Push A Version Tag

```bash
git tag v0.1.0
git push origin v0.1.0
```

The release workflow only triggers on tag pushes matching `v*`.

## 3. Wait For The Release Workflow

GitHub Actions runs the release workflow in `.github/workflows/release.yml`.

That workflow uses GoReleaser and `.goreleaser.yaml` to:

- build macOS binaries for `darwin/amd64` and `darwin/arm64`
- package release archives
- generate `skills_checksums.txt`
- create the GitHub Release
- upload the artifacts

## 4. Verify The GitHub Release

After the workflow finishes, verify the release contains:

- `skills_<version>_darwin_amd64.tar.gz`
- `skills_<version>_darwin_arm64.tar.gz`
- `skills_checksums.txt`

## 5. Verify The Public Installer

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
