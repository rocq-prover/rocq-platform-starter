BINARY      = rocq-platform-starter
GO          = go
SRC_DIR     = ./cmd/rocq-platform-starter

# Paths to source assets (relative to repo root)
REPO_ROOT   = ..
MANIFEST    = $(REPO_ROOT)/manifest/latest.json
TEMPLATES   = $(REPO_ROOT)/templates

# App bundle configuration
APP_NAME    = Rocq-Platform-Starter.app
BUNDLE_ID   = org.rocq-prover.rocq-platform-starter
DMG_NAME    = rocq-platform-starter.dmg

VERSION     ?= dev
LDFLAGS     = -s -w -X main.Version=$(VERSION)

.PHONY: all debug clean sync-assets app-bundle dmg

all: sync-assets app-bundle dmg

# Copy latest assets from the repo into the embedded directory
sync-assets:
	cp -f $(MANIFEST) embedded/manifest/latest.json
	cp -f $(TEMPLATES)/test.v embedded/templates/test.v
	cp -f $(TEMPLATES)/main.v embedded/templates/main.v
	cp -f $(TEMPLATES)/_RocqProject embedded/templates/_RocqProject

$(BINARY): sync-assets
	CGO_ENABLED=1 GOOS=darwin GOARCH=arm64 \
		$(GO) build -ldflags="$(LDFLAGS)" -o $(BINARY) $(SRC_DIR)

# Debug build: keeps symbols for debugging
debug: sync-assets
	CGO_ENABLED=1 GOOS=darwin GOARCH=arm64 \
		$(GO) build -ldflags="-X main.Version=$(VERSION)" -o $(BINARY)-debug $(SRC_DIR)

# Create macOS .app bundle
app-bundle: $(BINARY)
	@echo "==> Creating app bundle: $(APP_NAME)"
	rm -rf "$(APP_NAME)"
	mkdir -p "$(APP_NAME)/Contents/MacOS"
	mkdir -p "$(APP_NAME)/Contents/Resources"
	cp $(BINARY) "$(APP_NAME)/Contents/MacOS/$(BINARY)"
	cp Info.plist "$(APP_NAME)/Contents/Info.plist"
	@if [ "$(VERSION)" != "dev" ]; then \
		sed -i.bak 's|<string>1.0.0</string>|<string>$(VERSION)</string>|' "$(APP_NAME)/Contents/Info.plist"; \
		rm -f "$(APP_NAME)/Contents/Info.plist.bak"; \
	fi
	@# Convert PNG to icns if sips is available, otherwise use PNG directly
	@if command -v sips >/dev/null 2>&1 && command -v iconutil >/dev/null 2>&1; then \
		echo "  Creating .icns from PNG..."; \
		mkdir -p /tmp/rocq-icon.iconset; \
		sips -z 16 16     embedded/icon/rocq-icon.png --out /tmp/rocq-icon.iconset/icon_16x16.png >/dev/null 2>&1; \
		sips -z 32 32     embedded/icon/rocq-icon.png --out /tmp/rocq-icon.iconset/icon_16x16@2x.png >/dev/null 2>&1; \
		sips -z 32 32     embedded/icon/rocq-icon.png --out /tmp/rocq-icon.iconset/icon_32x32.png >/dev/null 2>&1; \
		sips -z 64 64     embedded/icon/rocq-icon.png --out /tmp/rocq-icon.iconset/icon_32x32@2x.png >/dev/null 2>&1; \
		sips -z 128 128   embedded/icon/rocq-icon.png --out /tmp/rocq-icon.iconset/icon_128x128.png >/dev/null 2>&1; \
		sips -z 256 256   embedded/icon/rocq-icon.png --out /tmp/rocq-icon.iconset/icon_128x128@2x.png >/dev/null 2>&1; \
		sips -z 256 256   embedded/icon/rocq-icon.png --out /tmp/rocq-icon.iconset/icon_256x256.png >/dev/null 2>&1; \
		sips -z 512 512   embedded/icon/rocq-icon.png --out /tmp/rocq-icon.iconset/icon_256x256@2x.png >/dev/null 2>&1; \
		sips -z 512 512   embedded/icon/rocq-icon.png --out /tmp/rocq-icon.iconset/icon_512x512.png >/dev/null 2>&1; \
		sips -z 1024 1024 embedded/icon/rocq-icon.png --out /tmp/rocq-icon.iconset/icon_512x512@2x.png >/dev/null 2>&1; \
		iconutil -c icns /tmp/rocq-icon.iconset -o "$(APP_NAME)/Contents/Resources/AppIcon.icns"; \
		rm -rf /tmp/rocq-icon.iconset; \
	else \
		echo "  sips/iconutil not available, copying PNG as icon"; \
		cp embedded/icon/rocq-icon.png "$(APP_NAME)/Contents/Resources/AppIcon.png"; \
	fi
	@echo "  App bundle created: $(APP_NAME)"

# Create DMG disk image
dmg: app-bundle
	@echo "==> Creating DMG: $(DMG_NAME)"
	rm -f $(DMG_NAME)
	hdiutil create -volname "Rocq-Platform-Starter" \
		-srcfolder "$(APP_NAME)" \
		-ov -format UDZO \
		$(DMG_NAME)
	@echo "  DMG created: $(DMG_NAME)"

# Sign the app bundle (requires Apple Developer certificate)
sign: app-bundle
	codesign --deep --force --options runtime \
		--entitlements entitlements.plist \
		--sign "Developer ID Application" \
		"$(APP_NAME)"

clean:
	rm -f $(BINARY) $(BINARY)-debug $(DMG_NAME)
	rm -rf "$(APP_NAME)"
