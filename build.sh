#!/bin/bash
set -euo pipefail
cd "$(dirname "$0")"

APP="dist/TokenTray.app"

echo "[build] compiling Go binary..."
CGO_ENABLED=1 go build -ldflags="-s -w" -o TokenTray .

echo "[build] assembling .app bundle..."
rm -rf "$APP"
mkdir -p "$APP/Contents/MacOS"
mkdir -p "$APP/Contents/Resources"

cp TokenTray "$APP/Contents/MacOS/TokenTray"

if [ -f icon.icns ]; then
    cp icon.icns "$APP/Contents/Resources/icon.icns"
    echo "[build] icon embedded"
fi

cat > "$APP/Contents/Info.plist" <<'PLIST'
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>CFBundleName</key><string>TokenTray</string>
    <key>CFBundleDisplayName</key><string>TokenTray</string>
    <key>CFBundleIdentifier</key><string>com.zrcder.tokentray</string>
    <key>CFBundleVersion</key><string>0.1.0</string>
    <key>CFBundleShortVersionString</key><string>0.1.0</string>
    <key>CFBundleExecutable</key><string>TokenTray</string>
    <key>CFBundlePackageType</key><string>APPL</string>
    <key>CFBundleIconFile</key><string>icon</string>
    <key>LSUIElement</key><true/>
    <key>LSMinimumSystemVersion</key><string>13.0</string>
    <key>NSHighResolutionCapable</key><true/>
    <key>NSAppTransportSecurity</key>
    <dict>
        <key>NSAllowsArbitraryLoads</key><true/>
    </dict>
</dict>
</plist>
PLIST

echo "[build] code signing..."
codesign --force --deep --sign - "$APP" 2>&1 || true

echo "[build] registering with LaunchServices..."
/System/Library/Frameworks/CoreServices.framework/Frameworks/LaunchServices.framework/Support/lsregister \
    "$APP" 2>/dev/null || true

echo ""
echo "✅ Build complete."
du -sh "$APP"
echo "   Launch: open $APP"
