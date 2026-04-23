BINARY      = rocq-platform-starter
GO          = go
SRC_DIR     = ./cmd/rocq-platform-starter

# Paths to source assets (relative to repo root)
REPO_ROOT   = ..
MANIFEST    = $(REPO_ROOT)/manifest/latest.json
TEMPLATES   = $(REPO_ROOT)/templates

# Install paths (user-local by default)
PREFIX      ?= $(HOME)/.local
BIN_DIR     = $(PREFIX)/bin
ICON_DIR    = $(PREFIX)/share/icons/hicolor/256x256/apps
DESKTOP_DIR = $(PREFIX)/share/applications

VERSION     ?= dev
LDFLAGS     = -s -w -X main.Version=$(VERSION)

.PHONY: all clean sync-assets install uninstall

all: sync-assets $(BINARY)

# Copy latest assets from the repo into the embedded directory
sync-assets:
	cp -f $(MANIFEST) embedded/manifest/latest.json
	cp -f $(TEMPLATES)/test.v embedded/templates/test.v
	cp -f $(TEMPLATES)/main.v embedded/templates/main.v
	cp -f $(TEMPLATES)/_RocqProject embedded/templates/_RocqProject

$(BINARY): sync-assets
	CGO_ENABLED=1 GOOS=linux GOARCH=amd64 \
		$(GO) build -ldflags="$(LDFLAGS)" -o $(BINARY) $(SRC_DIR)

install: $(BINARY)
	install -d $(BIN_DIR) $(ICON_DIR) $(DESKTOP_DIR)
	install -m 755 $(BINARY) $(BIN_DIR)/$(BINARY)
	install -m 644 embedded/icon/rocq-icon.png $(ICON_DIR)/rocq-platform-starter.png
	install -m 644 rocq-platform-starter.desktop $(DESKTOP_DIR)/rocq-platform-starter.desktop
	@echo "Installed to $(PREFIX). Make sure $(BIN_DIR) is in your PATH."
	@# Update desktop database if available
	@command -v update-desktop-database >/dev/null 2>&1 && update-desktop-database $(DESKTOP_DIR) 2>/dev/null || true
	@command -v gtk-update-icon-cache >/dev/null 2>&1 && gtk-update-icon-cache -f -t $(PREFIX)/share/icons/hicolor 2>/dev/null || true

uninstall:
	rm -f $(BIN_DIR)/$(BINARY)
	rm -f $(ICON_DIR)/rocq-platform-starter.png
	rm -f $(DESKTOP_DIR)/rocq-platform-starter.desktop
	@command -v update-desktop-database >/dev/null 2>&1 && update-desktop-database $(DESKTOP_DIR) 2>/dev/null || true
	@echo "Uninstalled from $(PREFIX)."

clean:
	rm -f $(BINARY)
