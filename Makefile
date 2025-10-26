# ABOUTME: Makefile for building and installing kizai static site generator
# ABOUTME: Provides minimal targets for build, install, clean, and help operations

BINARY_NAME=kizai

.PHONY: help build install clean

help:
	@echo "Available targets:"
	@echo "  build   - Build the binary"
	@echo "  install - Install the binary"
	@echo "  clean   - Remove build artifacts"
	@echo "  help    - Show this help message"

build:
	go build -o $(BINARY_NAME) .

install:
	go install .

clean:
	go clean
	rm -f $(BINARY_NAME)
