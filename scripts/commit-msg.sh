#!/usr/bin/env bash
set -euo pipefail

REPO_ROOT="$(git rev-parse --show-toplevel)"
CONFIG="$REPO_ROOT/.pre-commit.yaml"
LOCAL_CONFIG="$REPO_ROOT/.pre-commit.local.yaml"

# Check if conventional_commit hook is enabled
is_enabled() {
  local enabled="true"
  if [[ -f "$LOCAL_CONFIG" ]]; then
    val=$(grep -E "^\s+conventional_commit:" "$LOCAL_CONFIG" 2>/dev/null | awk '{print $2}' | tr -d '[:space:]')
    [[ -n "$val" ]] && enabled="$val"
  fi
  if [[ "$enabled" == "true" ]] && [[ -f "$CONFIG" ]]; then
    val=$(grep -E "^\s+conventional_commit:" "$CONFIG" 2>/dev/null | awk '{print $2}' | tr -d '[:space:]')
    [[ -n "$val" ]] && enabled="$val"
  fi
  [[ "$enabled" == "true" ]]
}

if ! is_enabled; then
  exit 0
fi

COMMIT_MSG_FILE="$1"
COMMIT_MSG=$(head -1 "$COMMIT_MSG_FILE")

# Conventional commit regex
# type(scope): description
# type: description
# Allowed types: feat fix refactor docs test chore style perf ci build revert
PATTERN='^(feat|fix|refactor|docs|test|chore|style|perf|ci|build|revert)(\([a-z0-9_/-]+\))?!?: .{1,100}$'

if ! echo "$COMMIT_MSG" | grep -qE "$PATTERN"; then
  echo "ERROR: Commit message does not follow Conventional Commits format or above 100 characters."
  echo ""
  echo "Expected: <type>(<scope>): <description>"
  echo ""
  echo "Types: feat fix refactor docs test chore style perf ci build revert"
  echo "Example: feat(auth): add password reset endpoint"
  echo ""
  echo "Your message: $COMMIT_MSG"
  exit 1
fi
