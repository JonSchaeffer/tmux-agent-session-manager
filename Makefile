BINARY=tasm
BIN_DIR=bin
CMD=./cmd/tasm
MODULE=github.com/jonschaeffer/tmux-agent-session-manager
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS=-s -w -X $(MODULE)/internal/cli.version=$(VERSION)

.PHONY: build install clean

build:
	go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY) $(CMD)

install: build
	cp $(BIN_DIR)/$(BINARY) /usr/local/bin/$(BINARY)

clean:
	rm -rf $(BIN_DIR)
