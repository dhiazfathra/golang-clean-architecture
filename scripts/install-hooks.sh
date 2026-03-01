#!/usr/bin/env bash
set -euo pipefail

REPO_ROOT="$(git rev-parse --show-toplevel)"
HOOKS_DIR="$REPO_ROOT/.git/hooks"

echo "Installing git hooks..."

# Pre-commit hook
cat > "$HOOKS_DIR/pre-commit" << 'HOOK'
#!/usr/bin/env bash
exec "$(git rev-parse --show-toplevel)/scripts/pre-commit.sh"
HOOK
chmod +x "$HOOKS_DIR/pre-commit"

# Commit-msg hook
cat > "$HOOKS_DIR/commit-msg" << 'HOOK'
#!/usr/bin/env bash
exec "$(git rev-parse --show-toplevel)/scripts/commit-msg.sh" "$1"
HOOK
chmod +x "$HOOKS_DIR/commit-msg"

echo "Git hooks installed successfully."
echo "Configure hooks in .pre-commit.yaml or override in .pre-commit.local.yaml"
