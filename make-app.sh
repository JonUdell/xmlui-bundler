#!/bin/bash
set -euo pipefail

APP_NAME="XMLUILauncher"
BINARY_NAME="xmlui-launcher"
VERSION="1.0"
ICON_SVG="xmlui-logo.svg"
ICONSET_DIR="icon.iconset"
ICNS_FILE="${APP_NAME}.icns"

APP_DIR="${APP_NAME}.app"
CONTENTS_DIR="${APP_DIR}/Contents"
MACOS_DIR="${CONTENTS_DIR}/MacOS"
RESOURCES_DIR="${CONTENTS_DIR}/Resources"
PLIST_PATH="${CONTENTS_DIR}/Info.plist"

# Ensure the SVG exists
if [ ! -f "$ICON_SVG" ]; then
  echo "❌ Missing: $ICON_SVG"
  exit 1
fi

# Clean up old output
rm -rf "$APP_DIR" "$ICONSET_DIR" "$ICNS_FILE"
mkdir -p "$ICONSET_DIR"

# Create required iconset files
declare -a sizes=(16 32 128 256 512)
for size in "${sizes[@]}"; do
  double=$((size * 2))
  base="icon_${size}x${size}"
  magick "$ICON_SVG" -background none -resize ${size}x${size} -gravity center -extent ${size}x${size} "$ICONSET_DIR/${base}.png"
  magick "$ICON_SVG" -background none -resize ${double}x${double} -gravity center -extent ${double}x${double} "$ICONSET_DIR/${base}@2x.png"
done

# Validate the iconset before building .icns
echo "📦 Building ICNS..."
iconutil -c icns "$ICONSET_DIR" -o "$ICNS_FILE" || {
  echo "❌ iconutil failed."
  exit 1
}

# Build .app structure
mkdir -p "$MACOS_DIR" "$RESOURCES_DIR"
cp "$BINARY_NAME" "$MACOS_DIR/"
cp "$ICNS_FILE" "$RESOURCES_DIR/"

# Generate Info.plist
cat > "$PLIST_PATH" <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN"
  "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>CFBundleName</key>
  <string>${APP_NAME}</string>
  <key>CFBundleIdentifier</key>
  <string>com.jonudell.xmluilauncher</string>
  <key>CFBundleVersion</key>
  <string>${VERSION}</string>
  <key>CFBundlePackageType</key>
  <string>APPL</string>
  <key>CFBundleExecutable</key>
  <string>${BINARY_NAME}</string>
  <key>CFBundleIconFile</key>
  <string>${ICNS_FILE}</string>
</dict>
</plist>
EOF

echo "✅ ${APP_DIR} created with icon: ${ICNS_FILE}"
