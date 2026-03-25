.PHONY: all build clean generate run test subscriber tokengen test-e2e

# Default target
all: generate build

# Generate FlatBuffers Go code
generate:
	@echo "Generating FlatBuffers code..."
	@cd ipc && flatc --go ipc_defs.fbs && cd ..
	@echo "FlatBuffers code generated."

# Build binaries
build: generate
	@echo "Building binaries..."
	@go mod tidy
	@go build -o child child.go child_ipc.go
	@go build -o parent parent.go
	@chmod +x child parent
	@echo "Build complete."

# Build subscriber binary (for e2e tests)
subscriber: generate
	@echo "Building subscriber..."
	@go build -o subscriber subscriber.go
	@chmod +x subscriber

# Build token generator (for e2e tests, separate module — no Agora SDK dependency)
tokengen:
	@echo "Building tokengen..."
	@cd cmd/tokengen && go mod tidy && go build -o ../../tokengen .
	@chmod +x tokengen

# Run e2e integration tests (requires APP_ID and APP_CERT environment variables)
test-e2e: build subscriber tokengen
	@./test_e2e.sh

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -f child parent subscriber tokengen
	@rm -rf ipc/ipcgen/
	@rm -f *.log
	@echo "Clean complete."

# Run with example (requires APP_ID environment variable)
run: build
	@if [ -z "$(APP_ID)" ]; then \
		echo "Error: APP_ID environment variable not set"; \
		echo "Usage: make run APP_ID=your_agora_app_id"; \
		exit 1; \
	fi
	./parent -appID "$(APP_ID)" -channelName "test-channel" -userID "test-user-123"

# Run tests (requires flatc for code generation, does NOT require Agora SDK native libs)
test: generate
	@echo "Running tests..."
	@go test -v -count=1 -tags test ./...
	@echo "Tests complete."

# Help
help:
	@echo "Available targets:"
	@echo "  make          - Generate FlatBuffers and build binaries"
	@echo "  make generate - Generate FlatBuffers Go code"
	@echo "  make build    - Build child and parent binaries"
	@echo "  make clean    - Remove build artifacts"
	@echo "  make run      - Run the demo (requires APP_ID=your_app_id)"
	@echo "  make test     - Run Go tests"
	@echo "  make subscriber - Build subscriber binary (for e2e tests)"
	@echo "  make tokengen - Build token generator (for e2e tests)"
	@echo "  make test-e2e - Run e2e tests (requires APP_ID and APP_CERT)"
	@echo "  make help     - Show this help message"
