#!/bin/bash
# Собирает Fokus-Mac.pkg (запускается на macOS в GitHub Actions).
# Нужны: dist/fokus-mac-arm64, dist/fokus-mac-amd64, assets/icon.png. Аргумент — версия.
set -euo pipefail
V="$1"
W=$(mktemp -d)
APP="$W/root/Applications/Фокус.app"
mkdir -p "$APP/Contents/MacOS" "$APP/Contents/Resources" "$W/scripts"
lipo -create dist/fokus-mac-arm64 dist/fokus-mac-amd64 -output "$APP/Contents/Resources/fokus"
chmod 755 "$APP/Contents/Resources/fokus"
cat > "$APP/Contents/MacOS/Fokus" <<'S'
#!/bin/sh
exec "$(dirname "$0")/../Resources/fokus" open
S
chmod 755 "$APP/Contents/MacOS/Fokus"
cat > "$APP/Contents/Info.plist" <<P
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict>
<key>CFBundleName</key><string>Фокус</string>
<key>CFBundleDisplayName</key><string>Фокус</string>
<key>CFBundleIdentifier</key><string>com.kolya.fokus.app</string>
<key>CFBundleExecutable</key><string>Fokus</string>
<key>CFBundleIconFile</key><string>icon</string>
<key>CFBundlePackageType</key><string>APPL</string>
<key>CFBundleShortVersionString</key><string>$V</string>
<key>CFBundleVersion</key><string>$V</string>
<key>LSMinimumSystemVersion</key><string>11.0</string>
</dict></plist>
P
IS="$W/icon.iconset"; mkdir -p "$IS"
for s in 16 32 128 256 512; do
  sips -z $s $s assets/icon.png --out "$IS/icon_${s}x${s}.png" >/dev/null
  d=$((s*2)); sips -z $d $d assets/icon.png --out "$IS/icon_${s}x${s}@2x.png" >/dev/null
done
iconutil -c icns "$IS" -o "$APP/Contents/Resources/icon.icns"

cat > "$W/scripts/postinstall" <<'S'
#!/bin/bash
"/Applications/Фокус.app/Contents/Resources/fokus" install /quiet
U=$(stat -f%Su /dev/console)
if [ -n "$U" ] && [ "$U" != "root" ]; then
  for i in $(seq 1 30); do nc -z 127.0.0.1 47890 2>/dev/null && break; sleep 0.5; done
  launchctl asuser "$(id -u "$U")" sudo -u "$U" open "http://127.0.0.1:47890/" || true
fi
exit 0
S
chmod 755 "$W/scripts/postinstall"

pkgbuild --analyze --root "$W/root" "$W/comp.plist"
plutil -replace 0.BundleIsRelocatable -bool NO "$W/comp.plist"
pkgbuild --root "$W/root" --component-plist "$W/comp.plist" --scripts "$W/scripts" \
  --identifier com.kolya.fokus.pkg --version "$V" --install-location / "$W/core.pkg"

mkdir -p "$W/res"
cat > "$W/res/welcome.html" <<'H'
<html><body style="font-family:-apple-system;font-size:13px">
<h2>«Фокус»</h2>
<p>Блокирует видео во всех браузерах и программах, включает ночной режим и правила для сайтов и программ.</p>
<p>Ослабить блокировки можно только одноразовым кодом из Google Authenticator того, у кого ключ.</p>
<p>Старая программа VideoBlock, если она есть, удалится сама.</p>
</body></html>
H
cat > "$W/dist.xml" <<X
<?xml version="1.0" encoding="utf-8"?>
<installer-gui-script minSpecVersion="2">
  <title>Фокус</title>
  <welcome file="welcome.html"/>
  <options customize="never" require-scripts="true" hostArchitectures="arm64,x86_64"/>
  <domains enable_localSystem="true"/>
  <choices-outline><line choice="c"/></choices-outline>
  <choice id="c" title="Фокус"><pkg-ref id="com.kolya.fokus.pkg"/></choice>
  <pkg-ref id="com.kolya.fokus.pkg" version="$V" onConclusion="none">core.pkg</pkg-ref>
</installer-gui-script>
X
productbuild --distribution "$W/dist.xml" --resources "$W/res" --package-path "$W" dist/Fokus-Mac.pkg
ls -la dist/Fokus-Mac.pkg
