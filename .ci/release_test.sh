#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/release.sh"

PASS=0
FAIL=0
TEMP_DIRS=()

assert_eq() {
  local test_name="$1" expected="$2" actual="$3"
  if [ "$expected" = "$actual" ]; then
    echo "  PASS: $test_name"
    PASS=$((PASS + 1))
  else
    echo "  FAIL: $test_name — expected '$expected', got '$actual'"
    FAIL=$((FAIL + 1))
  fi
}

# Creates a temp git repo and cd's into it. Sets REPO to the path.
# Must NOT be called in a subshell — the cd needs to affect the caller.
setup_temp_repo() {
  REPO="$(mktemp -d)"
  TEMP_DIRS+=("$REPO")
  cd "$REPO" || { echo "FATAL: failed to cd into temp repo $REPO"; exit 1; }
  # Safety: abort if we're not actually in the temp dir (prevents polluting the real repo)
  if [ "$PWD" != "$REPO" ]; then
    echo "FATAL: expected to be in $REPO but got $PWD"
    exit 1
  fi
  git init -q
  git config user.email "test@test.com"
  git config user.name "Test"
}

cleanup_all() {
  for dir in "${TEMP_DIRS[@]}"; do
    rm -rf "$dir"
  done
}
trap cleanup_all EXIT

# --- Tests ---

echo "=== Test: no tags starts from v0.0.0 ==="
setup_temp_repo
echo "init" > file.txt && git add . && git commit -q -m "chore: initial"
TAG="$(get_last_tag)"
assert_eq "get_last_tag returns v0.0.0" "v0.0.0" "$TAG"
CHANGELOG="$(get_changelog "$TAG")"
NEXT="$(compute_next_version "$TAG" "$CHANGELOG")"
assert_eq "first release is v0.0.1" "v0.0.1" "$NEXT"

echo "=== Test: patch bump (fix/refactor/chore commits) ==="
setup_temp_repo
echo "a" > file.txt && git add . && git commit -q -m "chore: initial"
git tag v1.2.3
echo "b" > file.txt && git add . && git commit -q -m "fix: a bug"
echo "c" > file.txt && git add . && git commit -q -m "refactor: cleanup"
CHANGELOG="$(get_changelog "v1.2.3")"
NEXT="$(compute_next_version "v1.2.3" "$CHANGELOG")"
assert_eq "patch bump" "v1.2.4" "$NEXT"

echo "=== Test: minor bump (has feat: commit) ==="
setup_temp_repo
echo "a" > file.txt && git add . && git commit -q -m "chore: initial"
git tag v1.2.3
echo "b" > file.txt && git add . && git commit -q -m "fix: a bug"
echo "c" > file.txt && git add . && git commit -q -m "feat: new feature"
CHANGELOG="$(get_changelog "v1.2.3")"
NEXT="$(compute_next_version "v1.2.3" "$CHANGELOG")"
assert_eq "minor bump" "v1.3.0" "$NEXT"

echo "=== Test: major bump (has breaking: commit) ==="
setup_temp_repo
echo "a" > file.txt && git add . && git commit -q -m "chore: initial"
git tag v1.2.3
echo "b" > file.txt && git add . && git commit -q -m "breaking: remove API"
CHANGELOG="$(get_changelog "v1.2.3")"
NEXT="$(compute_next_version "v1.2.3" "$CHANGELOG")"
assert_eq "major bump" "v2.0.0" "$NEXT"

echo "=== Test: major takes precedence over minor ==="
setup_temp_repo
echo "a" > file.txt && git add . && git commit -q -m "chore: initial"
git tag v1.2.3
echo "b" > file.txt && git add . && git commit -q -m "feat: new feature"
echo "c" > file.txt && git add . && git commit -q -m "breaking: big change"
echo "d" > file.txt && git add . && git commit -q -m "feat: another feature"
CHANGELOG="$(get_changelog "v1.2.3")"
NEXT="$(compute_next_version "v1.2.3" "$CHANGELOG")"
assert_eq "major over minor" "v2.0.0" "$NEXT"

echo "=== Test: version string parsing and reconstruction ==="
NEXT="$(compute_next_version "v0.0.0" "abc1234 chore: something")"
assert_eq "v0.0.0 patch" "v0.0.1" "$NEXT"
NEXT="$(compute_next_version "v0.0.0" "abc1234 feat: something")"
assert_eq "v0.0.0 minor" "v0.1.0" "$NEXT"
NEXT="$(compute_next_version "v10.20.30" "abc1234 fix: something")"
assert_eq "large version patch" "v10.20.31" "$NEXT"
NEXT="$(compute_next_version "v10.20.30" "abc1234 breaking: something")"
assert_eq "large version major" "v11.0.0" "$NEXT"

echo ""
echo "=== Results: $PASS passed, $FAIL failed ==="

if [ "$FAIL" -gt 0 ]; then
  exit 1
fi
