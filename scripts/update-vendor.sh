#!/bin/bash

# Manual vendor library update script
# Usage: ./scripts/update-vendor.sh

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
VENDOR_DIR="$PROJECT_ROOT/static/vendor"

echo "🔍 Checking for vendor library updates..."

# Read current versions
MARKED_VERSION=$(grep "marked.js=" "$VENDOR_DIR/versions.txt" | cut -d'=' -f2)
HIGHLIGHT_VERSION=$(grep "highlight.js=" "$VENDOR_DIR/versions.txt" | cut -d'=' -f2)

echo "Current versions:"
echo "  marked.js: $MARKED_VERSION"
echo "  highlight.js: $HIGHLIGHT_VERSION"

# Check latest versions
echo ""
echo "🌐 Fetching latest versions from GitHub..."

MARKED_LATEST=$(curl -s https://api.github.com/repos/markedjs/marked/releases/latest | jq -r .tag_name | sed 's/^v//')
HIGHLIGHT_LATEST=$(curl -s https://api.github.com/repos/highlightjs/highlight.js/releases/latest | jq -r .tag_name | sed 's/^v//')

echo "Latest versions:"
echo "  marked.js: $MARKED_LATEST"
echo "  highlight.js: $HIGHLIGHT_LATEST"

# Check if updates are needed
UPDATES_NEEDED=false

if [ "$MARKED_VERSION" != "$MARKED_LATEST" ]; then
    echo "✨ marked.js update available: $MARKED_VERSION → $MARKED_LATEST"
    UPDATES_NEEDED=true
fi

if [ "$HIGHLIGHT_VERSION" != "$HIGHLIGHT_LATEST" ]; then
    echo "✨ highlight.js update available: $HIGHLIGHT_VERSION → $HIGHLIGHT_LATEST"
    UPDATES_NEEDED=true
fi

if [ "$UPDATES_NEEDED" = false ]; then
    echo "✅ All vendor libraries are up to date!"
    exit 0
fi

echo ""
read -p "Do you want to update the libraries? (y/N): " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "Update cancelled."
    exit 0
fi

cd "$VENDOR_DIR"

# Backup current files
echo "📦 Creating backups..."
cp marked.min.js marked.min.js.backup
cp highlight.min.js highlight.min.js.backup
cp github.min.css github.min.css.backup

# Download updates
if [ "$MARKED_VERSION" != "$MARKED_LATEST" ]; then
    echo "⬇️  Downloading marked.js $MARKED_LATEST..."
    curl -f -o marked.min.js "https://cdn.jsdelivr.net/npm/marked@$MARKED_LATEST/marked.min.js"
fi

if [ "$HIGHLIGHT_VERSION" != "$HIGHLIGHT_LATEST" ]; then
    echo "⬇️  Downloading highlight.js $HIGHLIGHT_LATEST..."
    curl -f -o highlight.min.js "https://cdnjs.cloudflare.com/ajax/libs/highlight.js/$HIGHLIGHT_LATEST/highlight.min.js"
    curl -f -o github.min.css "https://cdnjs.cloudflare.com/ajax/libs/highlight.js/$HIGHLIGHT_LATEST/styles/github.min.css"
fi

# Validate downloads
echo "🔍 Validating downloaded files..."

if [ "$MARKED_VERSION" != "$MARKED_LATEST" ]; then
    if ! node -c marked.min.js; then
        echo "❌ ERROR: Downloaded marked.js is not valid JavaScript"
        mv marked.min.js.backup marked.min.js
        exit 1
    fi
    echo "✅ marked.js validation passed"
fi

if [ "$HIGHLIGHT_VERSION" != "$HIGHLIGHT_LATEST" ]; then
    if ! node -c highlight.min.js; then
        echo "❌ ERROR: Downloaded highlight.js is not valid JavaScript"
        mv highlight.min.js.backup highlight.min.js
        mv github.min.css.backup github.min.css
        exit 1
    fi
    echo "✅ highlight.js validation passed"
fi

# Update versions file
echo "📝 Updating versions file..."
if [ "$MARKED_VERSION" != "$MARKED_LATEST" ]; then
    sed -i.bak "s/marked.js=.*/marked.js=$MARKED_LATEST/" versions.txt
fi

if [ "$HIGHLIGHT_VERSION" != "$HIGHLIGHT_LATEST" ]; then
    sed -i.bak "s/highlight.js=.*/highlight.js=$HIGHLIGHT_LATEST/" versions.txt
fi

# Update last checked date
sed -i.bak "s/last_checked=.*/last_checked=$(date +%Y-%m-%d)/" versions.txt
rm -f versions.txt.bak

# Clean up backups
rm -f *.backup

echo ""
echo "✅ Vendor libraries updated successfully!"
echo "📋 Updated versions saved to static/vendor/versions.txt"
echo ""
echo "🧪 Please test the application to ensure everything works correctly:"
echo "   cd $PROJECT_ROOT && go run main.go"
