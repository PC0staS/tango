.PHONY: build build-all test run validate lint clean help

BINARY    ?= tango
BUILD_DIR ?= build
GO        ?= go
GOFLAGS   ?= -v
VERSION   ?= dev

build:
	$(GO) build $(GOFLAGS) -ldflags "-X main.Version=$(VERSION)" -o $(BINARY) .

build-all:
	@mkdir -p $(BUILD_DIR)
	@echo "==> linux/amd64"
	GOOS=linux   GOARCH=amd64   $(GO) build $(GOFLAGS) -ldflags "-X main.Version=$(VERSION)" -o $(BUILD_DIR)/tango-linux-amd64 .
	@echo "==> linux/arm64"
	GOOS=linux   GOARCH=arm64   $(GO) build $(GOFLAGS) -ldflags "-X main.Version=$(VERSION)" -o $(BUILD_DIR)/tango-linux-arm64 .
	@echo "==> darwin/amd64"
	GOOS=darwin  GOARCH=amd64   $(GO) build $(GOFLAGS) -ldflags "-X main.Version=$(VERSION)" -o $(BUILD_DIR)/tango-darwin-amd64 .
	@echo "==> darwin/arm64"
	GOOS=darwin  GOARCH=arm64   $(GO) build $(GOFLAGS) -ldflags "-X main.Version=$(VERSION)" -o $(BUILD_DIR)/tango-darwin-arm64 .
	@echo "==> windows/amd64"
	GOOS=windows GOARCH=amd64   $(GO) build $(GOFLAGS) -ldflags "-X main.Version=$(VERSION)" -o $(BUILD_DIR)/tango-windows-amd64.exe .
	@ls -lh $(BUILD_DIR)/

test:
	$(GO) test $(GOFLAGS) ./internal/...

run: build
	./$(BINARY) test examples/health_check.yaml

validate: build
	@for f in examples/*.yaml; do \
		echo "--- $$f ---"; \
		./$(BINARY) validate $$f || exit 1; \
	done

lint:
	golangci-lint run ./... 2>/dev/null || $(GO) vet ./...

clean:
	rm -f $(BINARY)
	rm -rf $(BUILD_DIR)

help:
	@echo "make build         Build local binary into ./$(BINARY)"
	@echo "make build-all     Cross-compile for all platforms into ./$(BUILD_DIR)"
	@echo "make test          Run all tests"
	@echo "make run           Build + run health_check example"
	@echo "make validate      Validate all example YAMLs"
	@echo "make lint          Run linter"
	@echo "make clean         Remove binary and build dir"
