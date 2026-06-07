BINARY=taism
BIN_DIR=bin
CMD=./cmd/taism

.PHONY: build install clean

build:
	go build -o $(BIN_DIR)/$(BINARY) $(CMD)

install: build
	cp $(BIN_DIR)/$(BINARY) /usr/local/bin/$(BINARY)

clean:
	rm -rf $(BIN_DIR)
