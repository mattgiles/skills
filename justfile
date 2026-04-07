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

check: fmt-check test

build:
  go build ./cmd/skills

run *args:
  go run ./cmd/skills {{args}}
