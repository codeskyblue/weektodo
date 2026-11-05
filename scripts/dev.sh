#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BACKEND_PORT="${DEV_BACKEND_PORT:-8787}"
FRONTEND_PORT="${DEV_FRONTEND_PORT:-5173}"
BACKEND_URL="http://127.0.0.1:${BACKEND_PORT}"
DEV_BIN="${ROOT}/bin/weektodo-dev"
BACKEND_LOG="${TMPDIR:-/tmp}/weektodo-dev-backend.log"
BACKEND_PID=""
WATCH_PID=""
DATA_DIR="${WEEKTODO_DATA_DIR:-${XDG_CONFIG_HOME:-$HOME/.config}/weektodo}"

cleanup() {
	if [[ -n "${WATCH_PID}" ]] && kill -0 "${WATCH_PID}" 2>/dev/null; then
		kill "${WATCH_PID}" 2>/dev/null || true
		wait "${WATCH_PID}" 2>/dev/null || true
	fi
	stop_backend
	rm -f "${DEV_BIN}.tmp"
}
trap cleanup EXIT INT TERM

stop_backend() {
	if [[ -n "${BACKEND_PID}" ]] && kill -0 "${BACKEND_PID}" 2>/dev/null; then
		kill "${BACKEND_PID}" 2>/dev/null || true
		wait "${BACKEND_PID}" 2>/dev/null || true
	fi
	BACKEND_PID=""
}

backend_hash() {
	case "$(uname -s)" in
	Darwin | FreeBSD)
		{
			find "${ROOT}/cmd" "${ROOT}/internal" -type f -name '*.go' -exec stat -f '%m %z %N' {} +
			find "${ROOT}/frontend" -maxdepth 2 -type f -name '*.go' -exec stat -f '%m %z %N' {} +
			stat -f '%m %z %N' "${ROOT}/go.mod"
			if [[ -f "${ROOT}/go.sum" ]]; then stat -f '%m %z %N' "${ROOT}/go.sum"; fi
		} | sort | cksum
		;;
	*)
		{
			find "${ROOT}/cmd" "${ROOT}/internal" -type f -name '*.go' -exec stat -c '%Y %s %n' {} +
			find "${ROOT}/frontend" -maxdepth 2 -type f -name '*.go' -exec stat -c '%Y %s %n' {} +
			stat -c '%Y %s %n' "${ROOT}/go.mod"
			if [[ -f "${ROOT}/go.sum" ]]; then stat -c '%Y %s %n' "${ROOT}/go.sum"; fi
		} | sort | cksum
		;;
	esac
}

build_backend() {
	rm -f "${DEV_BIN}.tmp"
	if ! go build -trimpath -ldflags "-X main.version=dev" -o "${DEV_BIN}.tmp" ./cmd/weektodo; then
		rm -f "${DEV_BIN}.tmp"
		return 1
	fi
	mv "${DEV_BIN}.tmp" "${DEV_BIN}"
}

start_backend() {
	if [[ -n "${WEEKTODO_DATA_DIR:-}" ]]; then
		WEEKTODO_DATA_DIR="${WEEKTODO_DATA_DIR}" \
			"${DEV_BIN}" -listen "127.0.0.1:${BACKEND_PORT}" >>"${BACKEND_LOG}" 2>&1 &
	else
		"${DEV_BIN}" -listen "127.0.0.1:${BACKEND_PORT}" >>"${BACKEND_LOG}" 2>&1 &
	fi
	BACKEND_PID=$!
	for _ in $(seq 1 40); do
		if curl -fsS "${BACKEND_URL}/api/health" >/dev/null 2>&1; then
			return 0
		fi
		if ! kill -0 "${BACKEND_PID}" 2>/dev/null; then
			echo "Backend failed to start; see ${BACKEND_LOG}" >&2
			return 1
		fi
		sleep 0.15
	done
}

watch_backend() {
	local last current
	last="$(backend_hash)"
	while true; do
		sleep 1
		current="$(backend_hash)"
		[[ "${current}" == "${last}" ]] && continue
		last="${current}"
		echo "[dev] backend changed, rebuilding..."
		if ! build_backend; then
			echo "[dev] backend build failed; keeping the running process" >&2
			continue
		fi
		stop_backend
		if ! start_backend; then
			return 1
		fi
		echo "[dev] backend restarted"
	done
}

cd "${ROOT}/frontend"
if [[ ! -d node_modules ]]; then
	echo "Installing frontend dependencies..."
	npm install
fi

# go:embed needs a populated frontend/build even in dev mode.
if [[ ! -f build/index.html ]]; then
	echo "Preparing embedded frontend assets..."
	npm run build
fi

cd "${ROOT}"
mkdir -p bin
echo "Building backend..."
build_backend

echo ""
echo "WeekTodo dev"
echo "  Backend:  ${BACKEND_URL}  (watching Go files)"
echo "  Frontend: http://localhost:${FRONTEND_PORT}"
echo "  Data dir: ${DATA_DIR}"
echo "  Log:      ${BACKEND_LOG}"
echo ""

start_backend
watch_backend &
WATCH_PID=$!

cd "${ROOT}/frontend"
WEEKTODO_BACKEND_URL="${BACKEND_URL}" npm run dev -- --host 127.0.0.1 --port "${FRONTEND_PORT}" --strictPort
