# =============================================================================
# NRServer — Build System
# =============================================================================

MODULE       := $(shell go list -m 2>/dev/null || echo nrserver)
BINARY_NAME  := nrserver
CMD_PATH     := ./cmd/nrserver
BUILD_DIR    := bin

VERSION      := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT       := $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
BUILD_TIME   := $(shell date -u +'%Y-%m-%dT%H:%M:%SZ')

# Injeta metadados de versão no binário sem depender de paths locais.
# Ajuste "internal/version" para o pacote real onde essas vars existem.
LDFLAGS := -w -s \
	-X '$(MODULE)/internal/version.Version=$(VERSION)' \
	-X '$(MODULE)/internal/version.Commit=$(COMMIT)' \
	-X '$(MODULE)/internal/version.BuildTime=$(BUILD_TIME)'

# Flags comuns a todos os builds:
#   -trimpath      remove paths absolutos do filesystem (ex: /media/all/.../nostr-relay-server)
#                  tanto de informações de pacote quanto de stack traces em panics
#   -buildvcs=false evita embutir metadados de VCS (que podem incluir paths/remote URLs)
#   CGO_ENABLED=0  garante binário estático, sem dependências de libc do host
COMMON_FLAGS := -a -trimpath -buildvcs=false -installsuffix cgo -ldflags="$(LDFLAGS)"

.DEFAULT_GOAL := help

# -----------------------------------------------------------------------------
# Help
# -----------------------------------------------------------------------------
.PHONY: help
help: ## Mostra esta ajuda
	@echo "NRServer — Build Instructions"
	@echo ""
	@echo "Uso: make <target>"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-16s\033[0m %s\n", $$1, $$2}'

# -----------------------------------------------------------------------------
# Builds por plataforma
# -----------------------------------------------------------------------------
.PHONY: linux-pc
linux-pc: ## Build para Linux PC (x86-64)
	@echo "==> Building $(BINARY_NAME) $(VERSION) for linux/amd64"
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 GOAMD64=v2 CGO_ENABLED=0 \
		go build $(COMMON_FLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 $(CMD_PATH)

.PHONY: linux-rpi
linux-rpi: ## Build para Linux Raspberry Pi (ARM64)
	@echo "==> Building $(BINARY_NAME) $(VERSION) for linux/arm64"
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 \
		go build $(COMMON_FLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 $(CMD_PATH)

.PHONY: windows
windows: ## Build para Windows (x86-64)
	@echo "==> Building $(BINARY_NAME) $(VERSION) for windows/amd64"
	@mkdir -p $(BUILD_DIR)
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 \
		go build $(COMMON_FLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe $(CMD_PATH)

.PHONY: windows32
windows32: ## Build para Windows (x86-32)
	@echo "==> Building $(BINARY_NAME) $(VERSION) for windows/386"
	@mkdir -p $(BUILD_DIR)
	GOOS=windows GOARCH=386 CGO_ENABLED=0 \
		go build $(COMMON_FLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-386.exe $(CMD_PATH)

.PHONY: darwin
darwin: ## Build para macOS (arm64, Apple Silicon)
	@echo "==> Building $(BINARY_NAME) $(VERSION) for darwin/arm64"
	@mkdir -p $(BUILD_DIR)
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 \
		go build $(COMMON_FLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 $(CMD_PATH)

# -----------------------------------------------------------------------------
# Release: builda tudo e gera checksums
# -----------------------------------------------------------------------------
.PHONY: release
release: clean linux-pc linux-rpi windows windows32 darwin ## Builda todas as plataformas e gera checksums
	@echo "==> Generating checksums"
	@cd $(BUILD_DIR) && sha256sum * > checksums.txt
	@echo "==> Release $(VERSION) pronto em ./$(BUILD_DIR)"

# -----------------------------------------------------------------------------
# Docker
# -----------------------------------------------------------------------------
.PHONY: docker
docker: ## Builda a imagem Docker
	@echo "==> Building Docker image gmouradev96/nrserver:$(VERSION)"
	docker build \
		-t gmouradev96/nrserver:$(VERSION) \
		-t gmouradev96/nrserver:latest \
		--build-arg VERSION=$(VERSION) \
		--build-arg BUILD_DATE=$(BUILD_TIME) \
		--build-arg COMMIT=$(COMMIT) \
		.

# -----------------------------------------------------------------------------
# Utilitários
# -----------------------------------------------------------------------------
.PHONY: verify
verify: ## Confirma que nenhum path local vazou para o(s) binário(s)
	@echo "==> Checking for leaked local paths in $(BUILD_DIR)/"
	@! grep -laR "$$(pwd)" $(BUILD_DIR) 2>/dev/null || (echo "FAIL: local path found in binary" && exit 1)
	@echo "OK: nenhum path local encontrado"

.PHONY: clean
clean: ## Remove artefatos de build
	@echo "==> Cleaning $(BUILD_DIR)"
	@rm -rf $(BUILD_DIR)

.PHONY: version
version: ## Mostra as informações de versão que serão injetadas
	@echo "Version:    $(VERSION)"
	@echo "Commit:     $(COMMIT)"
	@echo "Build time: $(BUILD_TIME)"

.PHONY: all
all: help