#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ICONS="${ROOT}/desktop/build/icons"

command -v sips >/dev/null || { echo "sips is required (macOS)" >&2; exit 1; }
command -v wails3 >/dev/null || { echo "wails3 is required: go install github.com/wailsapp/wails/v3/cmd/wails3@latest" >&2; exit 1; }

sips -s format png "${ICONS}/trayicon.svg" --out "${ROOT}/desktop/build/trayicon.png" >/dev/null
sips -s format png "${ICONS}/appicon.svg" --out "${ICONS}/appicon.png" >/dev/null

mkdir -p "${ROOT}/desktop/build/darwin" "${ROOT}/desktop/build/windows"
(cd "${ROOT}/desktop" && wails3 generate icons \
	-input "build/icons/appicon.png" \
	-macfilename "build/darwin/AppIcon.icns" \
	-windowsfilename "build/windows/icon.ico")

echo "Generated tray and app icons"
