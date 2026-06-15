.PHONY: all run build clean build-cross check-tools test setup

APP_ID = com.openwhats.native
TARGET_DIR = ./cmd/open-whats/
OUTPUT_NAME = open-whats

all: build

# Run the app locally without compiling a binary
run:
	@echo "Running Open-Whats Native..."
	go run $(TARGET_DIR)main.go

# Build a native binary for the host OS
build:
	@echo "Building local native executable..."
	go env -w CGO_ENABLED=1
	go build -o $(OUTPUT_NAME) $(TARGET_DIR)main.go

# Install CLI tools required for packaging
check-tools:
	@echo "Checking fyne tools..."
	go install fyne.io/fyne/v2/cmd/fyne@latest
	go install github.com/fyne-io/fyne-cross@latest

# Cross-compile for Windows, Mac, and Linux using Docker
build-cross: check-tools
	@echo "Cross-compiling with fyne-cross (requires Docker)..."
	fyne-cross windows -arch=amd64 -app-id=$(APP_ID) $(TARGET_DIR)
	fyne-cross darwin -arch=amd64,arm64 -app-id=$(APP_ID) $(TARGET_DIR)
	fyne-cross linux -arch=amd64 -app-id=$(APP_ID) $(TARGET_DIR)
	@echo "Check fyne-cross/bin/ for the builds."

# Clean compiled binaries and cache
clean:
	@echo "Cleaning build artifacts..."
	rm -f $(OUTPUT_NAME)
	rm -rf fyne-cross/
	rm -rf db/

# Run tests
test:
	@echo "Running tests..."
	go test -v ./...

# Setup local environment
setup:
	@echo "Setting up dependencies..."
	go mod tidy
	@gcc --version >nul 2>&1 || echo "WARNING: gcc is not installed. You may need to install MinGW-w64 for CGO."

