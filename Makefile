# The pult command for this Machine. `make install` puts it in ~/.local/bin,
# which must be on the PATH of a non-login `$SHELL -c`: the app runs pult
# over SSH in such a shell (README.md).
PREFIX ?= $(HOME)/.local
BINDIR ?= $(PREFIX)/bin

# The Android toolchain pins, and the Core's Android library for the phone
# (arm64) and the emulator (amd64), bound through the Bridge (mobile/).
include toolchain.env
ANDROID_HOME ?= $(HOME)/Android/Sdk
export ANDROID_HOME ANDROID_CMDLINE_TOOLS ANDROID_PLATFORM ANDROID_NDK ANDROID_MIN_API
CORE_AAR := build/core.aar
CORE_TARGETS := android/arm64,android/amd64
CORE_VERSION ?= $(shell git describe --always --dirty 2>/dev/null || echo dev)

.PHONY: build install test toolchain-check bind

build:
	go build -trimpath -o bin/pult ./cmd/pult

# install replaces the file instead of writing into it, so a running pult
# never sees a half-written binary.
install: build
	install -d $(DESTDIR)$(BINDIR)
	install -m 0755 bin/pult $(DESTDIR)$(BINDIR)/pult

test:
	go vet ./...
	go test ./...

toolchain-check:
	sh scripts/toolchain-check.sh

# gomobile takes the newest NDK under $ANDROID_HOME/ndk and the first gobind
# on the PATH, so bind names the pinned NDK and puts a gobind built from
# go.mod first.
bind: toolchain-check
	go build -o bin/gobind golang.org/x/mobile/cmd/gobind
	install -d $(dir $(CORE_AAR))
	PATH="$(CURDIR)/bin:$$PATH" ANDROID_NDK_HOME="$(ANDROID_HOME)/ndk/$(ANDROID_NDK)" \
		go tool gomobile bind -target=$(CORE_TARGETS) -androidapi $(ANDROID_MIN_API) -trimpath \
		-ldflags "-X github.com/stevetosak/pult/core.version=$(CORE_VERSION)" \
		-o $(CORE_AAR) ./mobile
