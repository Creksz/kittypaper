.PHONY: build install uninstall clean fmt vet lint

GOTOOLCHAIN ?= local
GO ?= go
APP_VERSION ?= 0.1.0
GIT_COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo dev)
LDFLAGS := -ldflags "-s -w -X kittypaper/internal/version.Version=$(APP_VERSION) -X kittypaper/internal/version.Commit=$(GIT_COMMIT)"

BIN_DIR := bin
PREFIX ?= $(HOME)/.local
BINDIR ?= $(PREFIX)/bin

build: build-cli build-tui build-gui

build-cli:
	$(GO) build $(LDFLAGS) -o $(BIN_DIR)/kittypaper ./cmd/kittypaper

build-tui:
	$(GO) build $(LDFLAGS) -o $(BIN_DIR)/kittypaper-tui ./cmd/kittypaper-tui

build-gui:
	$(GO) build $(LDFLAGS) -o $(BIN_DIR)/kittypaper-gui ./cmd/kittypaper-gui

fmt:
	GOTOOLCHAIN=$(GOTOOLCHAIN) $(GO) fmt ./...

vet:
	GOTOOLCHAIN=$(GOTOOLCHAIN) $(GO) vet ./...

lint: fmt vet

# Install the main CLI to PATH (~/.local/bin by default).
# After this: kittypaper gui | kittypaper tui | kittypaper set ...
install: build-cli
	mkdir -p "$(BINDIR)"
	install -m 755 "$(BIN_DIR)/kittypaper" "$(BINDIR)/kittypaper"
	@echo "Installed: $(BINDIR)/kittypaper"
	@echo "Make sure $(BINDIR) is on your PATH, then run: kittypaper gui"

uninstall:
	rm -f "$(BINDIR)/kittypaper"

clean:
	rm -rf $(BIN_DIR)
