VERSION ?= $(shell v=$$(git describe --tags --always --dirty 2>/dev/null | sed 's/^v//'); echo "$${v:-dev}")
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
GOTOOLCHAIN ?= auto
export GOTOOLCHAIN
BIN := bin/weektodo
LDFLAGS := -s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT)

.PHONY: all frontend build dev run test test-webdav check desktop icons clean

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

icons:
	./scripts/make-icons.sh

clean:
	rm -rf bin frontend/.svelte-kit frontend/build desktop/build/bin desktop/build/windows
