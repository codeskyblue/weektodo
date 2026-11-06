#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DESKTOP="${ROOT}/desktop"
VERSION="${VERSION:-$(cd "${ROOT}" && git describe --tags --always --dirty 2>/dev/null | sed 's/^v//')}"
VERSION="${VERSION:-dev}"
VERSION="${VERSION#v}"
COMMIT="${COMMIT:-$(cd "${ROOT}" && git rev-parse --short HEAD 2>/dev/null || echo unknown)}"
OUT="${OUT_DIR:-${DESKTOP}/build/bin/windows}"
ARCHS="${ARCHS:-amd64 arm64}"
WIN_VERSION="${VERSION%%-*}"
if [[ "${WIN_VERSION}" =~ ^[0-9]+(\.[0-9]+)*$ ]]; then
	IFS=. read -r -a parts <<< "${WIN_VERSION}"
	while [[ ${#parts[@]} -lt 4 ]]; do parts+=("0"); done
	WIN_VERSION="${parts[0]}.${parts[1]}.${parts[2]}.${parts[3]}"
else
	WIN_VERSION="0.0.0.0"
fi

if [[ ! -f "${ROOT}/frontend/build/index.html" ]]; then
	echo "Frontend is not built. Run: make frontend" >&2
	exit 1
fi
if [[ ! -f "${DESKTOP}/build/windows/icon.ico" || ! -f "${DESKTOP}/build/windows/wails.exe.manifest" ]]; then
	echo "Windows build assets are missing. Run: make icons" >&2
	exit 1
fi

WORK="$(mktemp -d)"
cleanup() {
	rm -rf "${WORK}"
	rm -f "${DESKTOP}"/rsrc_windows_*.syso
}
trap cleanup EXIT

cat > "${WORK}/info.json" <<EOF
{
	"fixed": {
		"file_version": "${WIN_VERSION}",
		"product_version": "${WIN_VERSION}"
	},
	"info": {
		"0409": {
			"ProductVersion": "${VERSION}",
			"FileVersion": "${WIN_VERSION}",
			"CompanyName": "WeekTodo",
			"FileDescription": "WeekTodo weekly task planner",
			"LegalCopyright": "Copyright WeekTodo",
			"ProductName": "WeekTodo",
			"OriginalFilename": "WeekTodo.exe"
		}
	}
}
EOF

mkdir -p "${OUT}"
for arch in ${ARCHS}; do
	echo "Building Windows ${arch}..."
	(cd "${ROOT}" && go run ./cmd/windows-syso \
		-arch "${arch}" \
		-icon "${DESKTOP}/build/windows/icon.ico" \
		-manifest "${DESKTOP}/build/windows/wails.exe.manifest" \
		-info "${WORK}/info.json" \
		-out "${DESKTOP}/rsrc_windows_${arch}.syso")

	stage="${WORK}/stage-${arch}"
	mkdir -p "${stage}"
	(cd "${DESKTOP}" && GOTOOLCHAIN=auto GOOS=windows GOARCH="${arch}" CGO_ENABLED=0 \
		go build -trimpath \
		-ldflags "-s -w -H=windowsgui -X main.version=${VERSION} -X main.commit=${COMMIT}" \
		-o "${stage}/WeekTodo.exe" .)
	cat > "${stage}/README.txt" <<'EOF'
WeekTodo for Windows
====================

1. Run WeekTodo.exe.
2. Microsoft Edge WebView2 Runtime (Evergreen) is required. Windows 10/11 normally has it.
3. Data is stored in %AppData%\weektodo.

No installer is required; keep the executable in any writable folder.
EOF
	rm -f "${DESKTOP}/rsrc_windows_${arch}.syso"
	(cd "${stage}" && zip -q -r "${OUT}/WeekTodo-windows-${arch}-${VERSION}.zip" .)
	echo "Built ${OUT}/WeekTodo-windows-${arch}-${VERSION}.zip"
done
