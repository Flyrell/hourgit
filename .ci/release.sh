#!/usr/bin/env bash
set -euo pipefail

# release.sh — Compute the next semver tag from commit messages since the last tag.
#
# Functions can be sourced by other scripts (e.g. tests).
# When executed directly, outputs VERSION and CHANGELOG for consumption by CI.

get_last_tag() {
  git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0"
}

get_changelog() {
  local last_tag="$1"
  if [ "$last_tag" = "v0.0.0" ] && ! git tag -l | grep -q '^v0\.0\.0$'; then
    git log --oneline --no-merges
  else
    git log "${last_tag}..HEAD" --oneline --no-merges
  fi
}

compute_next_version() {
  local current="$1"
  local changelog="$2"

  # Strip leading 'v'
  local version="${current#v}"

  local major minor patch
  IFS='.' read -r major minor patch <<< "$version"

  local bump="patch"

  while IFS= read -r line; do
    # Extract prefix after the short hash (e.g. "abc1234 feat: ...")
    local msg="${line#* }"
    case "$msg" in
      breaking:*|breaking\(*) bump="major" ;;
      feat:*|feat\(*)
        if [ "$bump" != "major" ]; then
          bump="minor"
        fi
        ;;
    esac
  done <<< "$changelog"

  case "$bump" in
    major) major=$((major + 1)); minor=0; patch=0 ;;
    minor) minor=$((minor + 1)); patch=0 ;;
    patch) patch=$((patch + 1)) ;;
  esac

  echo "v${major}.${minor}.${patch}"
}

# When run directly (not sourced), output VERSION and CHANGELOG.
if [ "${BASH_SOURCE[0]}" = "$0" ]; then
  LAST_TAG="$(get_last_tag)"
  CHANGELOG="$(get_changelog "$LAST_TAG")"
  NEXT_VERSION="$(compute_next_version "$LAST_TAG" "$CHANGELOG")"

  echo "VERSION=${NEXT_VERSION}"
  echo "CHANGELOG<<EOF"
  echo "$CHANGELOG"
  echo "EOF"
fi
