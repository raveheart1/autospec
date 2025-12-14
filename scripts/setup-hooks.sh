#!/bin/sh
# Install git hooks for development
# Usage: ./scripts/setup-hooks.sh
#    or: make dev-setup

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
HOOKS_DIR="$(git rev-parse --git-dir)/hooks"

echo "Installing git hooks..."

cp "$SCRIPT_DIR/hooks/pre-merge-commit" "$HOOKS_DIR/pre-merge-commit"
chmod +x "$HOOKS_DIR/pre-merge-commit"
echo "✓ Installed pre-merge-commit hook"

cp "$SCRIPT_DIR/hooks/post-merge" "$HOOKS_DIR/post-merge"
chmod +x "$HOOKS_DIR/post-merge"
echo "✓ Installed post-merge hook"

echo ""
echo "Done! Hooks installed to $HOOKS_DIR"
