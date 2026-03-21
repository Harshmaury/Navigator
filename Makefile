# Makefile — Navigator
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -X main.navigatorVersion=$(VERSION)
BINDIR  := $(HOME)/bin

.PHONY: all build clean

all: build

build:
	@echo "  → navigator $(VERSION)"
	@CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o $(BINDIR)/navigator ./cmd/navigator/

clean:
	@rm -f $(BINDIR)/navigator
