.PHONY: all clean tests lint build format benchmarks coverage

all: build tests lint

tests:
	@echo "Running tests..."
	go test -race ./...

benchmarks:
	@echo "Running benchmarks..."
	go test -race -run=^$$ -bench=. -benchmem ./...

build:
	@echo "Building package..."
	go build ./...

format:
	@echo "Formatting code..."
	gofmt -s -w .
	@if command -v golines >/dev/null 2>&1; then \
		golines -w .; \
	else \
		echo "golines not installed, skipping line wrapping"; \
	fi

lint:
	@echo "Linting code..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not installed, skipping lint"; \
	fi

coverage:
	@echo "Running coverage..."
	go test -race -coverprofile=coverage.out ./...

clean:
	@echo "Cleaning build artifacts..."
	@find . -type f -name "*.test" -delete
	@find . -type f -name "coverage.out" -delete
	@find . -type f -name "coverage.html" -delete
	@rm -rf bin/
