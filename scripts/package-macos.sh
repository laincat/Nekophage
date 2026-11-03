#!/bin/sh
# Builds Nekophage.app for macOS from this checkout.
#
#   sh scripts/package-macos.sh
#   sh scripts/package-macos.sh --no-sign
#
# The bundle is ad-hoc signed: that is what makes a Go binary launchable on Apple
# Silicon, and it is *not* a Developer ID signature or a notarised app. Gatekeeper
# on another machine will still refuse it until it is signed with a real identity,
# which needs credentials this repository does not have.
set -eu

cd "$(dirname "$0")/.."

sign=1
for arg in "$@"; do
	case "$arg" in
	--no-sign) sign=0 ;;
	*) printf 'package-macos: unknown argument %s\n' "$arg" >&2; exit 2 ;;
	esac
done

if [ "$(uname -s)" != "Darwin" ]; then
	printf 'package-macos: this builds a macOS bundle and runs on macOS only\n' >&2
	exit 2
fi

# Reject stale go.work/corestamp.go before replacing a previously built app.
sh scripts/check-core-pin.sh

out=build/macos
app=$out/Nekophage.app
rm -rf "$app"
mkdir -p "$app/Contents/MacOS" "$app/Contents/Resources"

# -trimpath keeps this machine's paths out of the binary; the version and commit
# come from the same code the Windows build uses, so the two agree.
go build -trimpath -ldflags "-s -w" -o "$app/Contents/MacOS/Nekophage" .

info=$("$app/Contents/MacOS/Nekophage" -version)
if [ -z "$info" ]; then
	printf 'package-macos: the built binary printed no version\n' >&2
	exit 1
fi
version=$(printf '%s\n' "$info" | head -1 | awk '{print $2}')

cat > "$app/Contents/Info.plist" <<PLIST
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>CFBundleName</key><string>Nekophage</string>
	<key>CFBundleDisplayName</key><string>Nekophage</string>
	<key>CFBundleIdentifier</key><string>com.laincat.nekophage</string>
	<key>CFBundleExecutable</key><string>Nekophage</string>
	<key>CFBundlePackageType</key><string>APPL</string>
	<key>CFBundleShortVersionString</key><string>${version}</string>
	<key>CFBundleVersion</key><string>${version}</string>
	<key>LSMinimumSystemVersion</key><string>12.0</string>
	<key>NSHighResolutionCapable</key><true/>
</dict>
</plist>
PLIST

cp LICENSE "$app/Contents/Resources/LICENSE"
plutil -lint "$app/Contents/Info.plist" >/dev/null

if [ "$sign" = 1 ]; then
	codesign --force --sign - "$app" >/dev/null 2>&1 || {
		printf 'package-macos: ad-hoc signing failed\n' >&2
		exit 1
	}
fi

printf 'PASS  macOS bundle %s (%s)\n' "$app" "$(printf '%s' "$version")"
printf '%s\n' "$info"
if [ "$sign" = 1 ]; then
	printf 'signature: %s\n' "$(codesign --verify --verbose=2 "$app" 2>&1 | tail -1)"
else
	printf 'signature: none (--no-sign)\n'
fi
