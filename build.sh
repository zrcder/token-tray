#!/bin/bash
# Build TokenTray.app — native Go binary wrapped in macOS app bundle.
# Output: dist/TokenTray.app (~5MB, no runtime deps)
set -euo pipefail
cd "$(dirname "$0")"

APP="dist/TokenTray.app"

echo "[build] compiling Go binary..."
CGO_ENABLED=1 go build -ldflags="-s -w" -o TokenTray .

echo "[build] cleaning previous bundle..."
rm -rf "$APP"

echo "[build] assembling .app bundle..."
mkdir -p "$APP/Contents/MacOS"
mkdir -p "$APP/Contents/Resources"

cp TokenTray "$APP/Contents/MacOS/TokenTray"

echo "[build] ad-hoc signing (required by macOS 15 for status item)..."
codesign --force --deep --sign - "$APP" 2>&1 || echo "  (codesign failed, continuing)"

cat > "$APP/Contents/Info.plist" <<'PLIST'
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>CFBundleName</key>
    <string>TokenTray</string>
    <key>CFBundleDisplayName</key>
    <string>TokenTray</string>
    <key>CFBundleIdentifier</key>
    <string>com.zja.tokentray</string>
    <key>CFBundleVersion</key>
    <string>0.1.0</string>
    <key>CFBundleShortVersionString</key>
    <string>0.1.0</string>
    <key>CFBundleExecutable</key>
    <string>TokenTray</string>
    <key>CFBundlePackageType</key>
    <string>APPL</string>
    <key>LSUIElement</key>
    <true/>
    <key>LSMinimumSystemVersion</key>
    <string>13.0</string>
    <key>NSHighResolutionCapable</key>
    <true/>
    <key>NSAppTransportSecurity</key>
    <dict>
        <key>NSAllowsArbitraryLoads</key>
        <true/>
    </dict>
</dict>
</plist>
PLIST

echo "[build] registering with LaunchServices..."
/System/Library/Frameworks/CoreServices.framework/Frameworks/LaunchServices.framework/Support/lsregister \
    "$APP" 2>/dev/null || true

echo ""
echo "✅ Build complete."
echo "   Bundle: $APP"
du -sh "$APP"
echo ""
echo "   Launch: open $APP"
