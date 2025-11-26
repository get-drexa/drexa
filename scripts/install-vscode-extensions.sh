#!/bin/bash

# Install VS Code extensions from devcontainer.json
# This script reads extensions from the single source of truth

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DEVCONTAINER_JSON="$SCRIPT_DIR/../.devcontainer/devcontainer.json"

echo "Installing VS Code extensions from devcontainer.json..."

if [ ! -f "$DEVCONTAINER_JSON" ]; then
    echo "Error: devcontainer.json not found at $DEVCONTAINER_JSON"
    exit 1
fi

# Use Bun to parse JSON and extract extensions
EXTENSIONS=$(bun -e "
const config = await Bun.file('$DEVCONTAINER_JSON').json();
const extensions = config?.customizations?.vscode?.extensions ?? [];
console.log(extensions.join('\n'));
")

if [ -z "$EXTENSIONS" ]; then
    echo "No extensions found in devcontainer.json"
    exit 0
fi

# Determine which CLI to use (code-server, cursor, or code)
if command -v code-server &> /dev/null; then
    CLI="code-server"
elif command -v cursor &> /dev/null; then
    CLI="cursor"
elif command -v code &> /dev/null; then
    CLI="code"
else
    echo "Warning: No VS Code CLI found (code-server, cursor, or code)"
    echo "Extensions to install:"
    echo "$EXTENSIONS"
    exit 0
fi

echo "Using $CLI to install extensions..."

# Install each extension
echo "$EXTENSIONS" | while read -r ext; do
    if [ -n "$ext" ]; then
        echo "Installing: $ext"
        $CLI --install-extension "$ext" 2>/dev/null || true
    fi
done

echo "Extension installation complete!"
