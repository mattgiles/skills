#!/bin/sh

set -eu

OWNER="mattgiles"
REPO="skills"
BINARY="skills"
VERSION="${VERSION:-}"
INSTALL_DIR="${INSTALL_DIR:-}"

tmpdir=""

cleanup() {
  if [ -n "$tmpdir" ] && [ -d "$tmpdir" ]; then
    rm -rf "$tmpdir"
  fi
}

trap cleanup EXIT INT TERM

fail() {
  printf '%s\n' "error: $*" >&2
  exit 1
}

need_cmd() {
  command -v "$1" >/dev/null 2>&1 || fail "required command not found: $1"
}

detect_os() {
  os="$(uname -s)"
  case "$os" in
    Darwin) printf 'darwin\n' ;;
    *) fail "unsupported operating system: $os (v1 supports macOS only)" ;;
  esac
}

detect_arch() {
  arch="$(uname -m)"
  case "$arch" in
    arm64|aarch64) printf 'arm64\n' ;;
    x86_64) printf 'amd64\n' ;;
    *) fail "unsupported architecture: $arch" ;;
  esac
}

resolve_version() {
  if [ -n "$VERSION" ]; then
    printf '%s\n' "$VERSION"
    return
  fi

  api_url="https://api.github.com/repos/$OWNER/$REPO/releases/latest"
  latest="$(
    curl -fsSL "$api_url" |
      sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' |
      head -n 1
  )"

  [ -n "$latest" ] || fail "could not resolve latest release version"
  printf '%s\n' "$latest"
}

normalize_asset_version() {
  version="$1"
  case "$version" in
    v*) printf '%s\n' "${version#v}" ;;
    *) printf '%s\n' "$version" ;;
  esac
}

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

  fail "could not find a writable install directory"
}

install_binary() {
  src="$1"
  dir="$2"
  mkdir -p "$dir"

  if command -v install >/dev/null 2>&1; then
    install -m 0755 "$src" "$dir/$BINARY"
  else
    cp "$src" "$dir/$BINARY"
    chmod 0755 "$dir/$BINARY"
  fi
}

verify_checksum() {
  archive="$1"
  checksums="$2"
  filename="$(basename "$archive")"

  expected="$(
    awk -v name="$filename" '$2 == name { print $1 }' "$checksums"
  )"
  [ -n "$expected" ] || fail "missing checksum for $filename"

  actual="$(shasum -a 256 "$archive" | awk '{ print $1 }')"
  [ "$expected" = "$actual" ] || fail "checksum mismatch for $filename"
}

print_path_hint() {
  dir="$1"
  if path_contains "$dir"; then
    return
  fi

  printf '\n'
  printf '%s\n' "Add this directory to PATH to use $BINARY from new shells:"
  printf '  export PATH="%s:$PATH"\n' "$dir"
}

main() {
  need_cmd curl
  need_cmd tar
  need_cmd shasum
  need_cmd uname
  need_cmd mktemp

  os="$(detect_os)"
  arch="$(detect_arch)"
  version="$(resolve_version)"
  asset_version="$(normalize_asset_version "$version")"
  install_dir="$(choose_install_dir)"

  asset="${BINARY}_${asset_version}_${os}_${arch}.tar.gz"
  checksums_asset="${BINARY}_checksums.txt"
  release_base="https://github.com/$OWNER/$REPO/releases/download/$version"

  tmpdir="$(mktemp -d)"
  archive_path="$tmpdir/$asset"
  checksums_path="$tmpdir/$checksums_asset"

  printf '%s\n' "Installing $BINARY $version for $os/$arch"
  printf '%s\n' "Download source: $release_base/$asset"

  curl -fsSL "$release_base/$asset" -o "$archive_path"
  curl -fsSL "$release_base/$checksums_asset" -o "$checksums_path"

  verify_checksum "$archive_path" "$checksums_path"

  tar -xzf "$archive_path" -C "$tmpdir"
  [ -f "$tmpdir/$BINARY" ] || fail "archive did not contain $BINARY"

  install_binary "$tmpdir/$BINARY" "$install_dir"

  printf '%s\n' "Installed to $install_dir/$BINARY"
  "$install_dir/$BINARY" version
  print_path_hint "$install_dir"
}

main "$@"
