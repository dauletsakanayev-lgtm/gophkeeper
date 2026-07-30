# GophKeeper — Makefile
#
# Основные цели:
#   make build         — собрать сервер и клиент под текущую платформу в ./bin
#   make build-all     — собрать клиент под Linux, macOS, Windows (amd64) в ./dist
#   make test          — прогнать все тесты
#   make cover         — тесты с покрытием, вывести total
#   make db-up/db-down — поднять/остановить PostgreSQL через docker-compose
#   make lint          — go vet
#   make fmt           — gofmt -s -w

MODULE  := github.com/dauletsakanayev-lgtm/gophkeeper
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE    ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

LDFLAGS := -X '$(MODULE)/internal/version.Version=$(VERSION)' \
           -X '$(MODULE)/internal/version.Commit=$(COMMIT)' \
           -X '$(MODULE)/internal/version.Date=$(DATE)'

BIN_DIR  := bin
DIST_DIR := dist

.PHONY: help build build-server build-client build-all test cover lint fmt db-up db-down clean

help:
	@echo "GophKeeper build targets:"
	@echo "  make build       - build server and client for current OS/arch"
	@echo "  make build-all   - build client for linux/darwin/windows amd64"
	@echo "  make test        - run all tests"
	@echo "  make cover       - run tests with coverage report"
	@echo "  make lint        - go vet ./..."
	@echo "  make fmt         - gofmt -s -w ."
	@echo "  make db-up       - start local PostgreSQL via docker-compose"
	@echo "  make db-down     - stop local PostgreSQL"
	@echo "  make clean       - remove ./bin and ./dist"

build: build-server build-client

build-server:
	@mkdir -p $(BIN_DIR)
	go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/gophkeeper-server ./cmd/server

build-client:
	@mkdir -p $(BIN_DIR)
	go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/gophkeeper ./cmd/client

# Кросс-компиляция клиента под три платформы.
# Именование: gophkeeper_<os>_<arch>[.exe]
build-all:
	@mkdir -p $(DIST_DIR)
	GOOS=linux   GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(DIST_DIR)/gophkeeper_linux_amd64        ./cmd/client
	GOOS=darwin  GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(DIST_DIR)/gophkeeper_darwin_amd64       ./cmd/client
	GOOS=darwin  GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o $(DIST_DIR)/gophkeeper_darwin_arm64       ./cmd/client
	GOOS=windows GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(DIST_DIR)/gophkeeper_windows_amd64.exe  ./cmd/client

test:
	go test ./... -count=1

cover:
	go test ./... -count=1 -coverprofile=coverage.out
	go tool cover -func=coverage.out | tail -1

lint:
	go vet ./...

fmt:
	gofmt -s -w .

db-up:
	docker compose up -d postgres

db-down:
	docker compose down

clean:
	rm -rf $(BIN_DIR) $(DIST_DIR) coverage.out
