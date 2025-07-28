#!/bin/bash

# Vendor library health check script
# Usage: ./scripts/check-vendor.sh

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
VENDOR_DIR="$PROJECT_ROOT/static/vendor"

echo "🔍 Checking vendor library health..."

cd "$VENDOR_DIR"

# Check if all required files exist
REQUIRED_FILES=("marked.min.js" "highlight.min.js" "github.min.css" "versions.txt")
MISSING_FILES=()

for file in "${REQUIRED_FILES[@]}"; do
    if [ ! -f "$file" ]; then
        MISSING_FILES+=("$file")
    fi
done

if [ ${#MISSING_FILES[@]} -ne 0 ]; then
    echo "❌ Missing required vendor files:"
    printf '   - %s\n' "${MISSING_FILES[@]}"
    echo ""
    echo "Run './scripts/update-vendor.sh' to download missing files."
    exit 1
fi

echo "✅ All required files present"

# Validate JavaScript files
echo "🔍 Validating JavaScript files..."

if ! node -c marked.min.js; then
    echo "❌ marked.min.js is not valid JavaScript"
    exit 1
fi
echo "✅ marked.min.js is valid"

if ! node -c highlight.min.js; then
    echo "❌ highlight.min.js is not valid JavaScript"
    exit 1
fi
echo "✅ highlight.min.js is valid"

# Check file sizes (basic sanity check)
MARKED_SIZE=$(stat -f%z marked.min.js 2>/dev/null || stat -c%s marked.min.js 2>/dev/null)
HIGHLIGHT_SIZE=$(stat -f%z highlight.min.js 2>/dev/null || stat -c%s highlight.min.js 2>/dev/null)
CSS_SIZE=$(stat -f%z github.min.css 2>/dev/null || stat -c%s github.min.css 2>/dev/null)

echo "📊 File sizes:"
echo "   marked.min.js: ${MARKED_SIZE} bytes"
echo "   highlight.min.js: ${HIGHLIGHT_SIZE} bytes"
echo "   github.min.css: ${CSS_SIZE} bytes"

# Sanity check - files should be reasonably sized
if [ "$MARKED_SIZE" -lt 10000 ]; then
    echo "❌ WARNING: marked.min.js seems unusually small (${MARKED_SIZE} bytes)"
fi

if [ "$HIGHLIGHT_SIZE" -lt 50000 ]; then
    echo "❌ WARNING: highlight.min.js seems unusually small (${HIGHLIGHT_SIZE} bytes)"
fi

if [ "$CSS_SIZE" -lt 500 ]; then
    echo "❌ WARNING: github.min.css seems unusually small (${CSS_SIZE} bytes)"
fi

# Show current versions
echo ""
echo "📋 Current versions:"
grep -E "(marked\.js|highlight\.js)=" versions.txt | sed 's/^/   /'

echo ""
echo "✅ Vendor library health check passed!"
echo "🌐 Application can run completely offline"
