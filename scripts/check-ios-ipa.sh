#!/usr/bin/env bash
# Validate a signed WeekTodo .ipa before uploading it to App Store Connect.
set -euo pipefail

IPA="${1:-mobile/ios/bin/weektodo.ipa}"
EXPECTED_BUNDLE_ID="${EXPECTED_BUNDLE_ID:-com.uiautodev.weektodo}"

if [[ ! -f "$IPA" ]]; then
	echo "IPA not found: $IPA" >&2
	exit 1
fi

TMP_DIR="$(mktemp -d)"
trap 'rm -rf "$TMP_DIR"' EXIT

unzip -q "$IPA" -d "$TMP_DIR"
APP="$(find "$TMP_DIR/Payload" -maxdepth 1 -name '*.app' -print -quit)"
if [[ -z "$APP" || ! -d "$APP" ]]; then
	echo "No .app found inside $IPA" >&2
	exit 1
fi

echo "App: $APP"
echo "Size: $(du -sh "$APP" | cut -f1)"

fail=0
require_plist() {
	local key="$1"
	local value
	value="$(/usr/libexec/PlistBuddy -c "Print :$key" "$APP/Info.plist" 2>/dev/null || true)"
	if [[ -z "$value" ]]; then
		echo "MISSING Info.plist key: $key"
		fail=1
	else
		echo "OK  $key = $value"
	fi
}

require_plist CFBundleIdentifier
require_plist CFBundleShortVersionString
require_plist CFBundleVersion
require_plist CFBundleIconName
require_plist UILaunchStoryboardName
require_plist NSLocalNetworkUsageDescription

bundle_id="$(/usr/libexec/PlistBuddy -c 'Print :CFBundleIdentifier' "$APP/Info.plist")"
version="$(/usr/libexec/PlistBuddy -c 'Print :CFBundleShortVersionString' "$APP/Info.plist")"
build="$(/usr/libexec/PlistBuddy -c 'Print :CFBundleVersion' "$APP/Info.plist")"

if [[ "$bundle_id" != "$EXPECTED_BUNDLE_ID" ]]; then
	echo "Bundle ID mismatch: $bundle_id (expected $EXPECTED_BUNDLE_ID)"
	fail=1
fi
if [[ ! "$build" =~ ^[0-9]+([.][0-9]+){0,2}$ ]]; then
	echo "CFBundleVersion is not a valid build number: $build"
	fail=1
fi

encryption="$(/usr/libexec/PlistBuddy -c 'Print :ITSAppUsesNonExemptEncryption' "$APP/Info.plist" 2>/dev/null || true)"
if [[ "$encryption" != "false" ]]; then
	echo "ITSAppUsesNonExemptEncryption should be false, got: $encryption"
	fail=1
else
	echo "OK  ITSAppUsesNonExemptEncryption = false"
fi

for file in Assets.car PrivacyInfo.xcprivacy embedded.mobileprovision "LaunchScreen.storyboardc"; do
	if [[ -e "$APP/$file" ]]; then
		echo "OK  $file"
	else
		echo "MISSING $file"
		fail=1
	fi
done

if ! codesign --verify --deep --strict "$APP" 2>&1; then
	echo "codesign verification failed"
	fail=1
else
	echo "OK  codesign verification"
fi

entitlements="$(codesign -d --entitlements :- "$APP" 2>/dev/null || true)"
if [[ -z "$entitlements" ]]; then
	echo "MISSING code signing entitlements"
	fail=1
else
	ent_file="$TMP_DIR/entitlements.plist"
	printf '%s' "$entitlements" > "$ent_file"
	if plutil -p "$ent_file" | grep -q 'get-task-allow.*=> 1\|get-task-allow.*=> true'; then
		echo "get-task-allow is true; this is a development build, not an App Store build"
		fail=1
	else
		echo "OK  get-task-allow is false"
	fi
fi

profile="$APP/embedded.mobileprovision"
if [[ -f "$profile" ]]; then
	profile_plist="$TMP_DIR/profile.plist"
	security cms -D -i "$profile" > "$profile_plist" 2>/dev/null || true
	profile_id="$(/usr/libexec/PlistBuddy -c 'Print :Entitlements:application-identifier' "$profile_plist" 2>/dev/null || true)"
	if [[ "$profile_id" != *"$EXPECTED_BUNDLE_ID" ]]; then
		echo "Provisioning profile does not match $EXPECTED_BUNDLE_ID (got $profile_id)"
		fail=1
	else
		echo "OK  provisioning profile matches $EXPECTED_BUNDLE_ID"
	fi
fi

echo
echo "Version: $version ($build)"
if [[ "$fail" -ne 0 ]]; then
	echo "IPA validation FAILED" >&2
	exit 1
fi
echo "IPA validation passed"
