VERSION ?= dev

BINARY := claude-code-hooks
APP := claude-code-hooks-notify.app
NOTIFIER_DIR := notifier
NOTIFIER_PROJECT := $(NOTIFIER_DIR)/claude-code-hooks-notify.xcodeproj
NOTIFIER_DERIVED := $(NOTIFIER_DIR)/.build/DerivedData
PKG := pkg/claude-code-hooks
GO_LDFLAGS := -s -w -X main.version=$(VERSION)

.PHONY: all icon build-go build-notifier build-universal package test test-go test-notifier clean

all: build-universal

# App icon: draw an iconset and compile it to AppIcon.icns.
icon:
	cd $(NOTIFIER_DIR) && swift generate_icon.swift && iconutil -c icns AppIcon.iconset -o AppIcon.icns

# Universal (arm64 + x86_64) Go binary.
build-go:
	mkdir -p dist
	GOOS=darwin GOARCH=arm64 go build -ldflags "$(GO_LDFLAGS)" -o dist/$(BINARY)_arm64 ./cmd/claude-code-hooks
	GOOS=darwin GOARCH=amd64 go build -ldflags "$(GO_LDFLAGS)" -o dist/$(BINARY)_amd64 ./cmd/claude-code-hooks
	lipo -create -output $(BINARY) dist/$(BINARY)_arm64 dist/$(BINARY)_amd64

# Universal notifier .app via xcodegen + xcodebuild.
build-notifier: icon
	cd $(NOTIFIER_DIR) && xcodegen generate
	xcodebuild -project $(NOTIFIER_PROJECT) -scheme claude-code-hooks-notify \
		-configuration Release -derivedDataPath $(NOTIFIER_DERIVED) \
		ARCHS="arm64 x86_64" ONLY_ACTIVE_ARCH=NO build
	rm -rf $(APP)
	cp -R $(NOTIFIER_DERIVED)/Build/Products/Release/$(APP) .

# Assemble the release tarball payload under pkg/claude-code-hooks/.
build-universal: build-go build-notifier
	rm -rf pkg
	mkdir -p $(PKG)/share
	cp $(BINARY) $(PKG)/
	cp -R $(APP) $(PKG)/
	cp share/hooks.json $(PKG)/share/

test: test-go test-notifier

test-go:
	go test ./...

test-notifier:
	cd $(NOTIFIER_DIR) && swift run claude-code-hooks-notify-tests

clean:
	rm -rf $(BINARY) $(APP) pkg dist $(NOTIFIER_DIR)/.build $(NOTIFIER_DIR)/*.xcodeproj \
		$(NOTIFIER_DIR)/AppIcon.iconset $(NOTIFIER_DIR)/AppIcon.icns
