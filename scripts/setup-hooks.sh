#!/bin/bash
# Install git hooks for this repository
# Usage: ./scripts/setup-hooks.sh
#    or: make dev-setup

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
HOOKS_DIR="$SCRIPT_DIR/hooks"
GIT_HOOKS_DIR="$(git rev-parse --git-dir)/hooks"

if [ ! -d "$HOOKS_DIR" ]; then
    echo "Error: hooks directory not found"
    exit 1
fi

for hook in "$HOOKS_DIR"/*; do
    if [ -f "$hook" ]; then
        hookname=$(basename "$hook")
        # Skip .sh files (those are speckit hooks, not git hooks)
        case "$hookname" in
            *.sh) continue ;;
        esac
        cp "$hook" "$GIT_HOOKS_DIR/$hookname"
        chmod +x "$GIT_HOOKS_DIR/$hookname"
        echo "✓ Installed $hookname"
    fi
done

# Configure merge driver for .gitattributes merge=ours strategy
# This keeps specs/, .dev/, .claude/commands/ deleted on main when merging from dev
git config merge.ours.driver true
echo "✓ Configured merge.ours driver"

echo "Done! Git hooks installed."
