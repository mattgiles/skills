#!/bin/sh

set -eu

fail() {
  printf '%s\n' "error: $*" >&2
  exit 1
}

script_dir="$(
  CDPATH=''
  cd -- "$(dirname "$0")"
  pwd
)"
repo_root="$(
  CDPATH=''
  cd -- "$script_dir/.."
  pwd
)"
extractor="$script_dir/release-notes.sh"
test_tmpdir="$(mktemp -d "${TMPDIR:-/tmp}/skills-release-notes.XXXXXX")"

cleanup() {
  if [ -n "$test_tmpdir" ] && [ -d "$test_tmpdir" ]; then
    rm -rf "$test_tmpdir"
  fi
}

trap cleanup EXIT INT TERM

fixture="$test_tmpdir/CHANGELOG.md"
{
  printf '%s\n' '# Changelog'
  printf '\n'
  printf '%s\n' '## [Unreleased]'
  printf '\n'
  printf '%s\n' '### Changed'
  printf '\n'
  printf '%s\n' '- Future work.'
  printf '\n'
  printf '%s\n' '## [0.6.0] - 2026-08-12'
  printf '\n'
  printf '%s\n' '### Added'
  printf '\n'
  printf '%s\n' '- Manifest fragments.'
  printf '\n'
  printf '%s\n' '### Fixed'
  printf '\n'
  printf '%s\n' '- Source repoints.'
  printf '\n'
  printf '%s\n' '## [0.5.0] - 2026-04-08'
  printf '\n'
  printf '%s\n' '- Previous release.'
} >"$fixture"

actual="$(sh "$extractor" v0.6.0 "$fixture")"
expected="$(
  printf '%s\n' '## [0.6.0] - 2026-08-12'
  printf '\n'
  printf '%s\n' '### Added'
  printf '\n'
  printf '%s\n' '- Manifest fragments.'
  printf '\n'
  printf '%s\n' '### Fixed'
  printf '\n'
  printf '%s\n' '- Source repoints.'
)"
[ "$actual" = "$expected" ] || fail "extractor returned the wrong section"

if sh "$extractor" v9.9.9 "$fixture" >/dev/null 2>&1; then
  fail "extractor accepted a missing version"
fi

printf '\n%s\n' '## [0.6.0] - 2026-08-13' >>"$fixture"
if sh "$extractor" v0.6.0 "$fixture" >/dev/null 2>&1; then
  fail "extractor accepted duplicate version sections"
fi

empty_fixture="$test_tmpdir/empty.md"
printf '%s\n' '## [0.7.0] - 2026-08-12' >"$empty_fixture"
if sh "$extractor" v0.7.0 "$empty_fixture" >/dev/null 2>&1; then
  fail "extractor accepted an empty version section"
fi

malformed_fixture="$test_tmpdir/malformed.md"
{
  printf '%s\n' '## [0.8.0] - August 12, 2026'
  printf '\n'
  printf '%s\n' '- Invalid date format.'
} >"$malformed_fixture"
if sh "$extractor" v0.8.0 "$malformed_fixture" >/dev/null 2>&1; then
  fail "extractor accepted a malformed version heading"
fi

if sh "$extractor" 0.6.0 "$fixture" >/dev/null 2>&1; then
  fail "extractor accepted a tag without the v prefix"
fi

sh "$extractor" v0.6.0 "$repo_root/CHANGELOG.md" >/dev/null
