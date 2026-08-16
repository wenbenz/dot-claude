BIN := bin/automake-db
SRC := $(wildcard cli/automake-db/*.go)

GOBIN := $(shell go env GOBIN)
ifeq ($(strip $(GOBIN)),)
GOBIN := $(shell go env GOPATH)/bin
endif
ifeq ($(strip $(GOBIN)),)
GOBIN := $(HOME)/go/bin
endif

.PHONY: all install clean

all: install

# Rebuilds only when a source file, go.mod, or go.sum is newer than the
# existing binary -- plain mtime comparison, no go invocation at all when
# nothing changed.
$(BIN): $(SRC) cli/automake-db/go.mod cli/automake-db/go.sum
	mkdir -p $(dir $(BIN))
	go build -C cli/automake-db -o $(CURDIR)/$(BIN) .

# Always re-links, even when $(BIN) was already up to date, so the symlink
# tracks whichever plugin version CLAUDE_PLUGIN_ROOT resolves to on this run.
install: $(BIN)
	mkdir -p $(GOBIN)
	ln -sf $(CURDIR)/$(BIN) $(GOBIN)/automake-db

clean:
	rm -f $(BIN)
