VERSION ?= $(shell v=$$(git describe --tags --always --dirty 2>/dev/null | sed 's/^v//'); echo "$${v:-dev}")
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
GOTOOLCHAIN ?= auto
export GOTOOLCHAIN
BIN := bin/weektodo
LDFLAGS := -s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT)

.PHONY: all frontend build dev run test test-webdav check desktop windows ios ios-package ios-ipa icons clean

all: build

frontend:
	cd frontend && npm install
	cd frontend && npm run build

build: frontend
	go build -trimpath -ldflags "$(LDFLAGS)" -o $(BIN) ./cmd/weektodo

dev:
	./scripts/dev.sh

run: build
	./$(BIN)

test: frontend
	go test ./...

test-webdav:
	go test ./internal/webdav/... -v
	go test ./internal/todo -run 'Merge' -v
	go test ./internal/server -run 'WebDAV' -v

check:
	cd frontend && npm run check

desktop: frontend
	VERSION=$(VERSION) COMMIT=$(COMMIT) ./scripts/build-macos-app.sh

windows: frontend
	VERSION=$(VERSION) COMMIT=$(COMMIT) ./scripts/build-windows-app.sh

ios:
	cd mobile/ios && wails3 task ios:run

ios-package:
	cd mobile/ios && wails3 task ios:package

IOS_VERSION ?= 0.3.0
IOS_BUILD ?= 1
IOS_SIGN_IDENTITY ?=
IOS_PROVISIONING_PROFILE ?=

ios-ipa:
	@test -n "$(IOS_SIGN_IDENTITY)" || (echo "IOS_SIGN_IDENTITY is required, e.g. Apple Distribution: Name (TEAMID)"; exit 1)
	@test -n "$(IOS_PROVISIONING_PROFILE)" || (echo "IOS_PROVISIONING_PROFILE is required, path to the App Store .mobileprovision"; exit 1)
	cd mobile/ios && wails3 task ios:package:ipa \
		IOS_PLATFORM=device \
		CODESIGN_IDENTITY="$(IOS_SIGN_IDENTITY)" \
		PROVISIONING_PROFILE="$(IOS_PROVISIONING_PROFILE)" \
		APP_VERSION="$(IOS_VERSION)" \
		APP_BUILD="$(IOS_BUILD)"

icons:
	./scripts/make-icons.sh

clean:
	rm -rf bin frontend/.svelte-kit frontend/build desktop/build/bin
	rm -f desktop/build/windows/*.syso
