# ==============================================================================
# Termino Makefile
# High-performance, low-overhead browser terminal workstation
# Targets: amd64, x86 (386), arm64, arm32 (armv7)
# ==============================================================================

SHELL        := /bin/bash
GO           ?= go
APP_NAME     := termino
BIN_DIR      := bin
CMD_DIR      := ./cmd/termino
VERSION      ?= 1.0.0
COMMIT       ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
BUILD_DATE   ?= $(shell date -u +'%Y-%m-%dT%H:%M:%SZ')

# Go build flags for minimal binary size, zero debug bloat, and trimmed paths
LDFLAGS      := -s -w -X main.Version=$(VERSION) -X main.GitCommit=$(COMMIT) -X main.BuildDate=$(BUILD_DATE)
BUILD_FLAGS  := -trimpath -ldflags="$(LDFLAGS)"

.PHONY: all build build-all build-amd64 build-x86 build-386 build-arm64 build-arm32 build-arm clean test check run checksums vendor help

# Default target: build for current host architecture
all: build

build:
	@mkdir -p $(BIN_DIR)
	@echo "==> Building $(APP_NAME) (host: $$(go env GOOS)/$$(go env GOARCH))..."
	CGO_ENABLED=0 $(GO) build $(BUILD_FLAGS) -o $(BIN_DIR)/$(APP_NAME) $(CMD_DIR)
	@echo "    Built: $(BIN_DIR)/$(APP_NAME)"

# Build for all supported architectures
build-all: build-amd64 build-x86 build-arm64 build-arm32 checksums
	@echo "==> All target binaries compiled successfully in $(BIN_DIR)/:"
	@ls -lh $(BIN_DIR)

# 64-bit x86 (AMD64 / Intel 64)
build-amd64:
	@mkdir -p $(BIN_DIR)
	@echo "==> Building Linux AMD64 (x86_64)..."
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GO) build $(BUILD_FLAGS) -o $(BIN_DIR)/$(APP_NAME)-linux-amd64 $(CMD_DIR)
	@echo "    Built: $(BIN_DIR)/$(APP_NAME)-linux-amd64"

# 32-bit x86 (i386 / IA-32)
build-x86: build-386

build-386:
	@mkdir -p $(BIN_DIR)
	@echo "==> Building Linux x86 (386)..."
	CGO_ENABLED=0 GOOS=linux GOARCH=386 $(GO) build $(BUILD_FLAGS) -o $(BIN_DIR)/$(APP_NAME)-linux-x86 $(CMD_DIR)
	@ln -sf $(APP_NAME)-linux-x86 $(BIN_DIR)/$(APP_NAME)-linux-386 2>/dev/null || cp $(BIN_DIR)/$(APP_NAME)-linux-x86 $(BIN_DIR)/$(APP_NAME)-linux-386
	@echo "    Built: $(BIN_DIR)/$(APP_NAME)-linux-x86 (and $(BIN_DIR)/$(APP_NAME)-linux-386)"

# 64-bit ARM (AArch64)
build-arm64:
	@mkdir -p $(BIN_DIR)
	@echo "==> Building Linux ARM64 (aarch64)..."
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 $(GO) build $(BUILD_FLAGS) -o $(BIN_DIR)/$(APP_NAME)-linux-arm64 $(CMD_DIR)
	@echo "    Built: $(BIN_DIR)/$(APP_NAME)-linux-arm64"

# 32-bit ARM (ARMv7 / Raspberry Pi 2/3/4 32-bit)
build-arm32: build-arm

build-arm:
	@mkdir -p $(BIN_DIR)
	@echo "==> Building Linux ARM32 (armv7)..."
	CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=7 $(GO) build $(BUILD_FLAGS) -o $(BIN_DIR)/$(APP_NAME)-linux-arm32 $(CMD_DIR)
	@ln -sf $(APP_NAME)-linux-arm32 $(BIN_DIR)/$(APP_NAME)-linux-armv7 2>/dev/null || cp $(BIN_DIR)/$(APP_NAME)-linux-arm32 $(BIN_DIR)/$(APP_NAME)-linux-armv7
	@echo "    Built: $(BIN_DIR)/$(APP_NAME)-linux-arm32 (and $(BIN_DIR)/$(APP_NAME)-linux-armv7)"

# Generate SHA256 checksums
checksums:
	@echo "==> Generating SHA256 checksums..."
	@cd $(BIN_DIR) && sha256sum $(APP_NAME)-linux-* > sha256sums.txt 2>/dev/null || true
	@echo "    Generated: $(BIN_DIR)/sha256sums.txt"

test:
	@echo "==> Running tests..."
	$(GO) test -v ./...

check: test

run:
	@echo "==> Running $(APP_NAME)..."
	$(GO) run $(CMD_DIR)

vendor:
	@bash ./scripts/pull_vendor.sh

clean:
	@echo "==> Cleaning build artifacts..."
	rm -rf $(BIN_DIR) $(APP_NAME) retroterm
	@echo "    Clean complete."

help:
	@echo "Termino Build System"
	@echo ""
	@echo "Usage:"
	@echo "  make <target>"
	@echo ""
	@echo "Build Targets:"
	@echo "  build        Build binary for current host into bin/$(APP_NAME)"
	@echo "  build-all    Cross-compile binaries for all 4 architectures (amd64, x86, arm64, arm32)"
	@echo "  build-amd64  Build Linux AMD64 (x86_64) binary"
	@echo "  build-x86    Build Linux x86 (32-bit 386) binary"
	@echo "  build-arm64  Build Linux ARM64 (aarch64) binary"
	@echo "  build-arm32  Build Linux ARM32 (armv7) binary"
	@echo "  checksums    Compute SHA256 checksums for all binaries in bin/"
	@echo ""
	@echo "Development Targets:"
	@echo "  run          Run application directly with go run"
	@echo "  vendor       Pull latest vendor files (htmx, xterm.js, css, addon-fit)"
	@echo "  test         Run test suite"
	@echo "  clean        Remove compiled binaries and bin/ directory"
	@echo "  help         Display this help message"
