set shell := ["zsh", "-lc"]

default:
  @just --list

fmt:
  find cmd internal -type f -name '*.go' -print0 | xargs -0 gofmt -w

fmt-check:
  test -z "$(find cmd internal -type f -name '*.go' -print0 | xargs -0 gofmt -l)"

tidy:
  go mod tidy

test:
  go test ./...

snapshot:
  go test ./cmd/skills -run TestMarkdownSnapshots

snapshot-live:
  RUN_LIVE_SNAPSHOT_TESTS=1 go test ./cmd/skills -run TestMarkdownSnapshots

check: fmt-check test

build:
  go build ./cmd/skills

run *args:
  go run ./cmd/skills {{args}}
