#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

APP="$ROOT/dist/Carpets Ameer.app"
STAGING="$ROOT/dist/DMG Contents"
SDK="$(xcrun --sdk macosx --show-sdk-path)"

rm -rf "$ROOT/dist"
mkdir -p "$APP/Contents/MacOS" "$APP/Contents/Resources" "$STAGING"

# Compile natively on Apple macOS with the installed Xcode SDK,
# for both Apple Silicon and Intel; this is not a Linux cross-build.
for ARCH in arm64 x86_64; do
  echo "Compiling macOS $ARCH ..."
  xcrun swiftc -O -parse-as-library \
    -target "$ARCH-apple-macos11.0" -sdk "$SDK" \
    -framework AppKit -framework WebKit \
    "$ROOT/mac-app/CarpetsAmeerApp.swift" \
    -o "$ROOT/dist/CarpetsAmeer-$ARCH"
done

xcrun lipo -create \
  "$ROOT/dist/CarpetsAmeer-arm64" \
  "$ROOT/dist/CarpetsAmeer-x86_64" \
  -output "$APP/Contents/MacOS/CarpetsAmeer"
chmod 755 "$APP/Contents/MacOS/CarpetsAmeer"

cp "$ROOT/index.html" "$APP/Contents/Resources/index.html"
cat > "$APP/Contents/Info.plist" <<'PLIST'
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN"
 "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict>
<key>CFBundleIdentifier</key><string>ps.carpetsameer.sellingdesk</string>
<key>CFBundleName</key><string>Carpets Ameer</string>
<key>CFBundleDisplayName</key><string>Carpets Ameer</string>
<key>CFBundleExecutable</key><string>CarpetsAmeer</string>
<key>CFBundlePackageType</key><string>APPL</string>
<key>CFBundleShortVersionString</key><string>3.0.0</string>
<key>CFBundleVersion</key><string>3</string>
<key>LSMinimumSystemVersion</key><string>11.0</string>
<key>NSHighResolutionCapable</key><true/>
<key>LSApplicationCategoryType</key><string>public.app-category.business</string>
<key>NSDocumentsFolderUsageDescription</key>
<string>Save your carpet sales Excel workbook to a folder you choose.</string>
</dict></plist>
PLIST
echo "APPL????" > "$APP/Contents/PkgInfo"

# An ad-hoc signature makes the app structurally signed, without pretending
# it is Apple Developer ID signed or notarized.
codesign --force --deep --sign - "$APP"

echo "Verifying native package ..."
plutil -lint "$APP/Contents/Info.plist"
codesign --verify --deep --strict --verbose=2 "$APP"
xcrun lipo -info "$APP/Contents/MacOS/CarpetsAmeer"
file "$APP/Contents/MacOS/CarpetsAmeer"
"$APP/Contents/MacOS/CarpetsAmeer" --self-test

ln -s /Applications "$STAGING/Applications"
ditto "$APP" "$STAGING/Carpets Ameer.app"
hdiutil create -volname "Carpets Ameer" -srcfolder "$STAGING" \
  -ov -format UDZO "$ROOT/dist/Carpets_Ameer_macOS_Universal.dmg"
hdiutil verify "$ROOT/dist/Carpets_Ameer_macOS_Universal.dmg"

ditto -c -k --sequesterRsrc --keepParent "$APP" \
  "$ROOT/dist/Carpets_Ameer_macOS_Universal_App.zip"

echo "Build outputs:"
ls -lh "$ROOT/dist"/*.dmg "$ROOT/dist"/*.zip
