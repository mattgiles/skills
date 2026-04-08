set shell := ["zsh", "-lc"]

[private]
default:
    @just --list

# Format Go source files in cmd/ and internal/.
fmt:
    find cmd internal -type f -name '*.go' -print0 | xargs -0 gofmt -w

# Check whether Go source files are already formatted.
fmt-check:
    test -z "$(find cmd internal -type f -name '*.go' -print0 | xargs -0 gofmt -l)"

# Run go mod tidy.
tidy:
    go mod tidy

# Run the full Go test suite.
test:
    go test ./...

# Update local markdown snapshots for CLI output tests.
snapshot:
    go test ./cmd/skills -run TestMarkdownSnapshots

# Run live markdown snapshot tests against real repositories.
snapshot-live:
    RUN_LIVE_SNAPSHOT_TESTS=1 go test ./cmd/skills -run TestMarkdownSnapshots

# Run formatting and tests.
check: fmt-check test

# Build the skills CLI.
build:
    go build ./cmd/skills

# Create and push the next release tag; default bump is minor.
release bump='minor':
    {{ justfile_directory() }}/scripts/release.sh {{ bump }}

# Build and install the current HEAD binary.
install-head:
    {{ justfile_directory() }}/scripts/install-head.sh

# Run the CLI from source with arbitrary arguments.
run *args:
    go run ./cmd/skills {{ args }}
