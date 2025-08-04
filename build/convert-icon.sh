#!/bin/bash
# Usage: ./convert-icon.sh path/to/appicon.png
# This script creates a macOS .icns icon from a PNG

set -e

ICON_PNG="$1"
ICONSET_DIR="appicon.iconset"
ICNS_OUT="appicon.icns"

if [ -z "$ICON_PNG" ]; then
  echo "Usage: $0 path/to/appicon.png"
  exit 1
fi

mkdir -p "$ICONSET_DIR"

# Generate all required icon sizes
sips -z 16 16     "$ICON_PNG" --out "$ICONSET_DIR/icon_16x16.png"
sips -z 32 32     "$ICON_PNG" --out "$ICONSET_DIR/icon_16x16@2x.png"
sips -z 32 32     "$ICON_PNG" --out "$ICONSET_DIR/icon_32x32.png"
sips -z 64 64     "$ICON_PNG" --out "$ICONSET_DIR/icon_32x32@2x.png"
sips -z 128 128   "$ICON_PNG" --out "$ICONSET_DIR/icon_128x128.png"
sips -z 256 256   "$ICON_PNG" --out "$ICONSET_DIR/icon_128x128@2x.png"
sips -z 256 256   "$ICON_PNG" --out "$ICONSET_DIR/icon_256x256.png"
sips -z 512 512   "$ICON_PNG" --out "$ICONSET_DIR/icon_256x256@2x.png"
sips -z 512 512   "$ICON_PNG" --out "$ICONSET_DIR/icon_512x512.png"
sips -z 1024 1024 "$ICON_PNG" --out "$ICONSET_DIR/icon_512x512@2x.png"

# Convert to .icns
iconutil -c icns "$ICONSET_DIR" -o "$ICNS_OUT"
echo "Created $ICNS_OUT from $ICON_PNG"
