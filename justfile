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

install-head:
  #!/bin/sh
  set -eu
  BINARY="skills"
  INSTALL_DIR="${INSTALL_DIR:-}"
  INSTALL_PATH="${INSTALL_PATH:-}"

  path_contains() {
    target="$1"
    old_ifs="${IFS:- }"
    IFS=:
    for entry in $PATH; do
      if [ "$entry" = "$target" ]; then
        IFS="$old_ifs"
        return 0
      fi
    done
    IFS="$old_ifs"
    return 1
  }

  dir_writable_or_creatable() {
    dir="$1"
    if [ -d "$dir" ]; then
      [ -w "$dir" ]
      return
    fi

    parent="$(dirname "$dir")"
    while [ ! -d "$parent" ]; do
      next_parent="$(dirname "$parent")"
      [ "$next_parent" != "$parent" ] || return 1
      parent="$next_parent"
    done

    [ -w "$parent" ]
  }

  choose_install_dir() {
    if [ -n "$INSTALL_DIR" ]; then
      printf '%s\n' "$INSTALL_DIR"
      return
    fi

    for dir in /opt/homebrew/bin /usr/local/bin; do
      if path_contains "$dir" && dir_writable_or_creatable "$dir"; then
        printf '%s\n' "$dir"
        return
      fi
    done

    old_ifs="${IFS:- }"
    IFS=:
    for dir in $PATH; do
      [ -n "$dir" ] || continue
      if dir_writable_or_creatable "$dir"; then
        IFS="$old_ifs"
        printf '%s\n' "$dir"
        return
      fi
    done
    IFS="$old_ifs"

    for dir in "$HOME/.local/bin" "$HOME/bin"; do
      if dir_writable_or_creatable "$dir"; then
        printf '%s\n' "$dir"
        return
      fi
    done

    printf '%s\n' "error: could not find a writable install directory" >&2
    exit 1
  }

  if [ -n "$INSTALL_PATH" ]; then
    target="$INSTALL_PATH"
  elif command -v "$BINARY" >/dev/null 2>&1; then
    target="$(command -v "$BINARY")"
  else
    install_dir="$(choose_install_dir)"
    target="$install_dir/$BINARY"
  fi

  tmpdir="$(mktemp -d)"
  trap 'rm -rf "$tmpdir"' EXIT INT TERM
  bin_path="$tmpdir/$BINARY"

  go build -o "$bin_path" ./cmd/skills
  mkdir -p "$(dirname "$target")"
  if command -v install >/dev/null 2>&1; then
    install -m 0755 "$bin_path" "$target"
  else
    cp "$bin_path" "$target"
    chmod 0755 "$target"
  fi

  printf '%s\n' "installed HEAD to $target"
  "$target" version

run *args:
  go run ./cmd/skills {{args}}
