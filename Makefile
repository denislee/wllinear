.PHONY: build run test tidy clean fmt vet help

BINARY=wllinear

all: build

build:
	@echo "Building $(BINARY)..."
	go build -o $(BINARY) .

run: build
	@echo "Running $(BINARY)..."
	./$(BINARY)

test:
	go test ./...

tidy:
	go mod tidy

fmt:
	go fmt ./...

vet:
	go vet ./...

clean:
	rm -f $(BINARY)
	go clean

help:
	@echo "Targets:"
	@echo "  make build  - Build the binary"
	@echo "  make run    - Build and run the application"
	@echo "  make test   - Run tests"
	@echo "  make tidy   - go mod tidy"
	@echo "  make fmt    - go fmt"
	@echo "  make vet    - go vet"
	@echo "  make clean  - Remove binary and clean build cache"
