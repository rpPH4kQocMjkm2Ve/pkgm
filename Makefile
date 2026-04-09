.PHONY: build install uninstall test clean man

PREFIX   = /usr
DESTDIR  =
pkgname  = pkgm

BINDIR       = $(PREFIX)/bin
LICENSEDIR   = $(PREFIX)/share/licenses/$(pkgname)
MANDIR       = $(PREFIX)/share/man
ZSH_COMPDIR  = $(PREFIX)/share/zsh/site-functions
BASH_COMPDIR = $(PREFIX)/share/bash-completion/completions

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)

BINARY = pkgm

MANPAGES = man/pkgm.8

build:
	CGO_ENABLED=0 go build -trimpath -buildmode=pie -ldflags "-X main.version=$(VERSION)" -o $(BINARY) ./cmd/pkgm/

test:
	go test ./...

man: $(MANPAGES)

man/%.8: man/%.8.md
	pandoc -s -t man -o $@ $<

clean:
	rm -f $(BINARY) $(MANPAGES)

install:
	install -Dm755 $(BINARY)          $(DESTDIR)$(BINDIR)/$(BINARY)
	install -Dm644 LICENSE            $(DESTDIR)$(LICENSEDIR)/LICENSE
	install -Dm644 completions/_pkgm  $(DESTDIR)$(ZSH_COMPDIR)/_pkgm
	install -Dm644 completions/pkgm.bash $(DESTDIR)$(BASH_COMPDIR)/pkgm
	install -Dm644 man/pkgm.8         $(DESTDIR)$(MANDIR)/man8/pkgm.8

uninstall:
	rm -f  $(DESTDIR)$(BINDIR)/$(BINARY)
	rm -rf $(DESTDIR)$(LICENSEDIR)/
	rm -f  $(DESTDIR)$(ZSH_COMPDIR)/_pkgm
	rm -f  $(DESTDIR)$(BASH_COMPDIR)/pkgm
	rm -f  $(DESTDIR)$(MANDIR)/man8/pkgm.8
	@echo "Note: state files in ~/.local/state/pkgm/ preserved. Remove manually if needed."
