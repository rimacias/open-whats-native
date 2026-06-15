.PHONY: all run build clean build-cross check-tools

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
	go build -o $(OUTPUT_NAME) $(TARGET_DIR)main.go

# Install CLI tools required for packaging
check-tools:
	@command -v fyne >/dev/null 2>&1 || (echo "Installing fyne..." && go install fyne.io/fyne/v2/cmd/fyne@latest)
	@command -v fyne-cross >/dev/null 2>&1 || (echo "Installing fyne-cross..." && go install github.com/fyne-io/fyne-cross@latest)

# Cross-compile for Windows, Mac, and Linux using Docker
build-cross: check-tools
	@echo "Cross-compiling with fyne-cross (requires Docker)..."
	fyne-cross windows -arch=amd64 -app-id=$(APP_ID) -dir=$(TARGET_DIR)
	fyne-cross darwin -arch=amd64,arm64 -app-id=$(APP_ID) -dir=$(TARGET_DIR)
	fyne-cross linux -arch=amd64 -app-id=$(APP_ID) -dir=$(TARGET_DIR)
	@echo "Check fyne-cross/bin/ for the builds."

# Clean compiled binaries and cache
clean:
	@echo "Cleaning build artifacts..."
	rm -f $(OUTPUT_NAME)
	rm -rf fyne-cross/
	rm -rf db/
