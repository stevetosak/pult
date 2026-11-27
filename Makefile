# The pult command for this Machine. `make install` puts it in ~/.local/bin,
# which must be on the PATH of a non-login `$SHELL -c`: the app runs pult
# over SSH in such a shell (README.md).
PREFIX ?= $(HOME)/.local
BINDIR ?= $(PREFIX)/bin

.PHONY: build install test

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
