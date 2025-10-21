.PHONY: build deps lint test serve eval-ib

BINARY_NAME=cataloger
LLM_PROVIDER=openai

deps:
	go mod tidy

build: deps
	go build -o $(BINARY_NAME)

lint:
	go fmt ./...
	golangci-lint run

	@if command -v json5 > /dev/null 2>&1; then \
		echo "Running json5 validation on renovate.json5"; \
		json5 --validate renovate.json5 > /dev/null; \
	else \
		echo "json5 not found, skipping renovate validation"; \
	fi

test: build
	go test -v -race ./...

serve: build
	./cataloger serve

eval-ib: build
	./cataloger eval ib --sample 2 --verbose --provider $(LLM_PROVIDER)
