#!/bin/sh

set -eu

fail() {
  printf '%s\n' "error: $*" >&2
  exit 1
}

tag="${1:-}"
changelog_path="${2:-CHANGELOG.md}"

[ -n "$tag" ] || fail "usage: release-notes.sh <vX.Y.Z> [changelog-path]"
printf '%s\n' "$tag" | grep -Eq '^v[0-9]+\.[0-9]+\.[0-9]+$' ||
  fail "release tag must match vX.Y.Z: $tag"
[ -f "$changelog_path" ] || fail "changelog not found: $changelog_path"

version="${tag#v}"
heading_count="$(
  awk -v version="$version" '
    function is_target(line, prefix, date) {
      prefix = "## [" version "] - "
      if (substr(line, 1, length(prefix)) != prefix) {
        return 0
      }
      date = substr(line, length(prefix) + 1)
      return date ~ /^[0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9]$/
    }

    is_target($0) { count++ }
    END { print count + 0 }
  ' "$changelog_path"
)"

[ "$heading_count" -eq 1 ] ||
  fail "expected exactly one dated [$version] section in $changelog_path; found $heading_count"

if ! awk -v version="$version" '
  function is_target(line, prefix, date) {
    prefix = "## [" version "] - "
    if (substr(line, 1, length(prefix)) != prefix) {
      return 0
    }
    date = substr(line, length(prefix) + 1)
    return date ~ /^[0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9]$/
  }

  is_target($0) {
    found = 1
  }

  found && !is_target($0) && /^## \[/ {
    exit
  }

  found {
    print
    if (!is_target($0) && $0 !~ /^[[:space:]]*$/) {
      has_content = 1
    }
  }

  END {
    if (!found || !has_content) {
      exit 1
    }
  }
' "$changelog_path"; then
  fail "release section [$version] has no notes in $changelog_path"
fi
