.PHONY: build test clean run lint

BINARY=tango
GO=$(shell which go)
GOFLAGS=-v

build:
	$(GO) build $(GOFLAGS) -o $(BINARY) .

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
