# Install The CLI

`skills` is currently built from source.

## Prerequisites

- Go
- Git

## Build A Local Binary

From the repository root:

```bash
mkdir -p ./bin
go build -o ./bin/skills ./cmd/skills
```

Verify the binary:

```bash
./bin/skills --help
```

## Run Without Installing

If you do not want a binary yet:

```bash
go run ./cmd/skills --help
```

## Check Git Availability

Some workflows require Git access. A simple way to confirm the environment is:

```bash
git --version
./bin/skills source sync --help
```

For current command coverage, see [CLI Reference](../reference/cli.md).
