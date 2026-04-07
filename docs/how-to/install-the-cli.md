# Install The CLI

The public install path uses GitHub Releases plus a shell installer.

V1 platform support:

- macOS arm64
- macOS amd64

## Install The Latest Release

```bash
curl -fsSL https://raw.githubusercontent.com/mattgiles/skills/main/scripts/install.sh | sh
```

## Install A Specific Version

```bash
curl -fsSL https://raw.githubusercontent.com/mattgiles/skills/main/scripts/install.sh | VERSION=v0.1.0 sh
```

## What The Installer Does

- detects your macOS architecture
- downloads the matching release archive from GitHub Releases
- downloads and verifies the published checksums file
- installs `skills` into a writable directory already on `PATH` when possible
- falls back to `~/.local/bin` or `~/bin` with a clear `PATH` hint when needed

## Verify The Install

```bash
skills version
skills --help
```

## Upgrade An Existing Install

Upgrade to the latest release:

```bash
skills self update
```

Upgrade to a specific release:

```bash
skills self update --version v0.1.0
```

## Build From Source

Source builds are still available for contributors:

```bash
mkdir -p ./bin
go build -o ./bin/skills ./cmd/skills
./bin/skills version
```

## Check Git Availability

Some workflows require Git access:

```bash
git --version
skills source sync --help
```

For current command coverage, see [CLI Reference](../reference/cli.md).
