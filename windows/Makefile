BINARY      = rocq-platform-starter.exe
GO          = go
CC_WINDOWS  = x86_64-w64-mingw32-gcc
WINDRES     = x86_64-w64-mingw32-windres
SRC_DIR     = ./cmd/rocq-platform-starter

# Paths to source assets (relative to repo root)
REPO_ROOT   = ..
MANIFEST    = $(REPO_ROOT)/manifest/latest.json
TEMPLATES   = $(REPO_ROOT)/templates

# Windows resource file (embeds the .exe icon)
RC_FILE     = $(SRC_DIR)/app.rc
SYSO_FILE   = $(SRC_DIR)/app_windows_amd64.syso

VERSION     ?= dev
LDFLAGS     = -H windowsgui -s -w -X main.Version=$(VERSION)

.PHONY: all debug clean sync-assets

all: sync-assets $(BINARY)

# Copy latest assets from the repo into the embedded directory
sync-assets:
	cp -f $(MANIFEST) embedded/manifest/latest.json
	cp -f $(TEMPLATES)/test.v embedded/templates/test.v
	cp -f $(TEMPLATES)/main.v embedded/templates/main.v
	cp -f $(TEMPLATES)/_RocqProject embedded/templates/_RocqProject

# Compile Windows resource (.rc -> .syso)
$(SYSO_FILE): $(RC_FILE) embedded/icon/rocq-icon.ico
	$(WINDRES) -i $(RC_FILE) -O coff -o $(SYSO_FILE)

$(BINARY): sync-assets $(SYSO_FILE)
	CGO_ENABLED=1 CC=$(CC_WINDOWS) GOOS=windows GOARCH=amd64 \
		$(GO) build -ldflags="$(LDFLAGS)" -o $(BINARY) $(SRC_DIR)

# Debug build: shows console window for error output
debug: sync-assets $(SYSO_FILE)
	CGO_ENABLED=1 CC=$(CC_WINDOWS) GOOS=windows GOARCH=amd64 \
		$(GO) build -ldflags="-s -w -X main.Version=$(VERSION)" -o rocq-platform-starter-debug.exe $(SRC_DIR)

clean:
	rm -f $(BINARY) rocq-platform-starter-debug.exe $(SYSO_FILE)
