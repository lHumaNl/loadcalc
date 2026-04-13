.PHONY: build run test test-coverage lint clean fmt vet mod-tidy all help

# Version from git tags
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-s -w -X main.Version=$(VERSION)"

# Binary and directories
BINARY_NAME := loadcalc
BIN_DIR := bin
CMD_DIR := cmd/loadcalc

## build: Build the binary
build:
	@echo "Building $(BINARY_NAME)..."
	go build $(LDFLAGS) -o $(BIN_DIR)/$(BINARY_NAME) ./$(CMD_DIR)

## run: Run the application locally
run:
	go run ./$(CMD_DIR)

## test: Run all tests with race detection
test:
	go test -v -race -coverprofile=coverage.out ./...

## test-coverage: Generate HTML coverage report
test-coverage:
	go tool cover -html=coverage.out

## lint: Run golangci-lint
lint:
	golangci-lint run

## clean: Remove build artifacts
clean:
	rm -rf $(BIN_DIR)/
	rm -f coverage.out coverage.html

## fmt: Format code
fmt:
	go fmt ./...

## vet: Run go vet
vet:
	go vet ./...

## mod-tidy: Tidy dependencies
mod-tidy:
	go mod tidy

## all: Run all quality checks
all: fmt vet lint test

## help: Show this help message
help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/## /  /' | column -t -s ':'
