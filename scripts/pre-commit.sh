#!/usr/bin/env bash
set -euo pipefail

REPO_ROOT="$(git rev-parse --show-toplevel)"
CONFIG="$REPO_ROOT/.pre-commit.yaml"
LOCAL_CONFIG="$REPO_ROOT/.pre-commit.local.yaml"

# ── Parse YAML config (pure bash, no external deps) ──

is_hook_enabled() {
  local hook="$1"
  local enabled="true"

  # Check local override first
  if [[ -f "$LOCAL_CONFIG" ]]; then
    local val
    val=$(grep -E "^\s+${hook}:" "$LOCAL_CONFIG" 2>/dev/null | awk '{print $2}' | tr -d '[:space:]')
    if [[ -n "$val" ]]; then
      enabled="$val"
    fi
  fi

  # Fall back to base config
  if [[ "$enabled" == "true" ]] && [[ -f "$CONFIG" ]]; then
    local val
    val=$(grep -E "^\s+${hook}:" "$CONFIG" 2>/dev/null | awk '{print $2}' | tr -d '[:space:]')
    if [[ -n "$val" ]]; then
      enabled="$val"
    fi
  fi

  [[ "$enabled" == "true" ]]
}

FAILED=0

run_hook() {
  local name="$1"
  shift
  if is_hook_enabled "$name"; then
    printf "  %-25s" "$name"
    if output=$("$@" 2>&1); then
      echo "✓"
    else
      echo "✗"
      echo "$output" | head -20
      FAILED=1
    fi
  fi
}

echo "── pre-commit hooks ──"

# Get changed Go files and packages
CHANGED_GO_FILES=$(git diff --cached --name-only --diff-filter=ACMR -- '*.go' || true)
CHANGED_PKGS=""
if [[ -n "$CHANGED_GO_FILES" ]]; then
  CHANGED_PKGS=$(echo "$CHANGED_GO_FILES" | xargs -I{} dirname {} | sort -u | sed 's|^|./|')
fi

# Get changed SQL files
CHANGED_SQL=$(git diff --cached --name-only --diff-filter=ACMR -- 'migrations/*.sql' || true)

# ── Hook: gofmt ──
if [[ -n "$CHANGED_GO_FILES" ]]; then
  run_hook "gofmt" bash -c "
    unformatted=\$(echo '$CHANGED_GO_FILES' | xargs gofmt -l 2>/dev/null)
    if [[ -n \"\$unformatted\" ]]; then
      echo 'Files need formatting:'; echo \"\$unformatted\"; exit 1
    fi
  "
fi

# ── Hook: govet ──
if [[ -n "$CHANGED_PKGS" ]]; then
  run_hook "govet" bash -c "echo '$CHANGED_PKGS' | xargs go vet"
fi

# ── Hook: lint ──
if [[ -n "$CHANGED_PKGS" ]]; then
  run_hook "lint" bash -c "echo '$CHANGED_PKGS' | xargs golangci-lint run"
fi

# ── Hook: test_changed ──
if [[ -n "$CHANGED_PKGS" ]]; then
  run_hook "test_changed" bash -c "echo '$CHANGED_PKGS' | xargs go test -short -count=1"
fi

if [[ -n "$CHANGED_SQL" ]]; then
  run_hook "sql_validate" bash -c '
    for f in $0; do
      base=$(basename "$f")
      if ! echo "$base" | grep -qE '"'"'^[0-9]{14}_[a-z_]+\.(up|down)\.sql$'"'"'; then
        echo "Invalid migration name: $base"
        echo "Expected: YYYYMMDDHHMMSS_name.up.sql"
        exit 1
      fi
      if [[ ! -s "$f" ]]; then
        echo "Empty migration file: $f"
        exit 1
      fi
    done
  ' $CHANGED_SQL
fi

# ── Hook: secret_scan ──
if [[ -n "$CHANGED_GO_FILES" ]]; then
  run_hook "secret_scan" bash -c "
    # Detect common secret patterns in staged Go files
    patterns='(password|secret|token|api_key|apikey|private_key)\s*[:=]\s*\"[^\"]{8,}'
    matches=\$(echo '$CHANGED_GO_FILES' | xargs grep -inE \"\$patterns\" 2>/dev/null \
      | grep -v '//nolint:secret_scan' || true)
    if [[ -n \"\$matches\" ]]; then
      echo 'Potential hardcoded secrets detected:'; echo \"\$matches\"; exit 1
    fi
  "
fi

# ── Hook: vulncheck ──
run_hook "vulncheck" govulncheck ./...

if [[ "$FAILED" -ne 0 ]]; then
  echo ""
  echo "Pre-commit checks failed. Fix issues or use --no-verify to skip."
  exit 1
fi
echo "── all hooks passed ──"
