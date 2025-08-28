.PHONY: all build clean generate run

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
	@go build -o child child.go
	@go build -o parent parent.go
	@chmod +x child parent
	@echo "Build complete."

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -f child parent
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

# Help
help:
	@echo "Available targets:"
	@echo "  make          - Generate FlatBuffers and build binaries"
	@echo "  make generate - Generate FlatBuffers Go code"
	@echo "  make build    - Build child and parent binaries"
	@echo "  make clean    - Remove build artifacts"
	@echo "  make run      - Run the demo (requires APP_ID=your_app_id)"
	@echo "  make help     - Show this help message"
