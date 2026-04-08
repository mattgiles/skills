#!/bin/sh

set -eu

bump="${1:-minor}"

fail() {
  printf '%s\n' "error: $*" >&2
  exit 1
}

validate_bump() {
  case "$1" in
    minor|major)
      ;;
    *)
      fail "unsupported bump \"$1\"; use 'minor' or 'major'"
      ;;
  esac
}

require_git_repo() {
  git rev-parse --is-inside-work-tree >/dev/null 2>&1 || fail "not inside a Git repository"
}

require_origin_remote() {
  git remote get-url origin >/dev/null 2>&1 || fail "remote \"origin\" is not configured"
}

require_main_branch() {
  branch="$(git branch --show-current)"
  [ "$branch" = "main" ] || fail "release must run from branch \"main\"; current branch is \"$branch\""
}

require_origin_main_upstream() {
  upstream="$(git rev-parse --abbrev-ref --symbolic-full-name @{u} 2>/dev/null || true)"
  [ "$upstream" = "origin/main" ] || fail "release requires upstream \"origin/main\"; current upstream is \"${upstream:-<none>}\""
}

require_clean_worktree() {
  status="$(git status --porcelain)"
  if [ -n "$status" ]; then
    printf '%s\n' "error: release requires a clean worktree" >&2
    git status --short >&2
    exit 1
  fi
}

fetch_release_refs() {
  git fetch --quiet --tags origin main
}

require_head_matches_origin_main() {
  local_head="$(git rev-parse HEAD)"
  remote_head="$(git rev-parse origin/main)"
  [ "$local_head" = "$remote_head" ] || {
    printf '%s\n' "error: local main must match origin/main before releasing" >&2
    printf 'local:  %s\n' "$local_head" >&2
    printf 'origin: %s\n' "$remote_head" >&2
    exit 1
  }
}

latest_release_tag() {
  for tag in $(git tag --list --sort=-v:refname); do
    if printf '%s\n' "$tag" | grep -Eq '^v[0-9]+\.[0-9]+\.[0-9]+$'; then
      printf '%s\n' "$tag"
      return
    fi
  done
}

next_release_tag() {
  bump_type="$1"
  latest_tag="$2"

  if [ -z "$latest_tag" ]; then
    case "$bump_type" in
      minor) printf 'v0.1.0\n' ;;
      major) printf 'v1.0.0\n' ;;
    esac
    return
  fi

  version="${latest_tag#v}"
  old_ifs="${IFS}"
  IFS=.
  set -- $version
  IFS="${old_ifs}"
  major="$1"
  minor="$2"

  case "$bump_type" in
    minor) printf 'v%s.%s.0\n' "$major" "$((minor + 1))" ;;
    major) printf 'v%s.0.0\n' "$((major + 1))" ;;
  esac
}

require_missing_tag() {
  tag="$1"
  if git show-ref --verify --quiet "refs/tags/$tag"; then
    fail "tag \"$tag\" already exists"
  fi
}

create_annotated_tag() {
  tag="$1"
  git tag -a "$tag" -m "$tag"
}

push_tag() {
  tag="$1"
  if git push origin "$tag"; then
    printf 'pushed tag %s to origin\n' "$tag"
  else
    fail "created local tag \"$tag\" but failed to push it to origin"
  fi
}

main() {
  validate_bump "$bump"
  require_git_repo
  require_origin_remote
  require_main_branch
  require_origin_main_upstream
  require_clean_worktree
  fetch_release_refs
  require_head_matches_origin_main

  latest_tag="$(latest_release_tag || true)"
  next_tag="$(next_release_tag "$bump" "$latest_tag")"
  require_missing_tag "$next_tag"

  printf 'latest tag: %s\n' "${latest_tag:-<none>}"
  printf 'bump: %s\n' "$bump"
  printf 'next tag: %s\n' "$next_tag"

  create_annotated_tag "$next_tag"
  push_tag "$next_tag"
}

main "$@"
