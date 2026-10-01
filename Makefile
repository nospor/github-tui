BIN := github-tui
CMD := ./cmd/github-tui
PREFIX ?= /usr/local
BINDIR ?= $(PREFIX)/bin
INSTALL_DIR := $(DESTDIR)$(BINDIR)

.PHONY: build install run clean test

build:
	go build -o $(BIN) $(CMD)

install: build
	mkdir -p $(INSTALL_DIR)
	cp $(BIN) $(INSTALL_DIR)/$(BIN)
	@echo "Installed to $(INSTALL_DIR)/$(BIN)"

run:
	go run $(CMD)

test:
	go test ./...

clean:
	rm -f $(BIN)
