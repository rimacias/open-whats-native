# Building Open-Whats Native

Open-Whats Native uses [Fyne](https://fyne.io/) for its graphical user interface. To build production-ready applications across different platforms (Windows, macOS, Linux), we use `fyne-cross`, which leverages Docker containers to cross-compile the CGO dependencies required by Fyne (like GLFW, OpenGL).

## Prerequisites

1. **Go (1.21+)**: Ensure you have Go installed.
2. **Docker**: You must have Docker Desktop or the Docker Engine installed and running. `fyne-cross` relies on it heavily to build for platforms other than your host OS.

## Building Automatically

We provide convenient scripts to automate the cross-compilation process:

**For Mac/Linux Users:**
```bash
chmod +x scripts/prod-build.sh
./scripts/prod-build.sh
```

**For Windows Users:**
```cmd
scripts\prod-build.bat
```

**Using Make:**
If you have `make` installed, you can simply run:
```bash
make build-cross
```

These scripts will automatically:
1. Install `fyne` and `fyne-cross` if they are missing.
2. Check if Docker is running.
3. Build a Windows `.exe`.
4. Build macOS `.app` bundles for both Intel and Apple Silicon (ARM64).
5. Build a Linux tarball.

All compiled binaries will be output into the `fyne-cross/bin/` folder at the root of the project.

## Development Makefile Commands

We provide a `Makefile` in the root folder with several helpful shortcuts for development:

- `make run`: Runs the app locally without compiling a permanent binary.
- `make build`: Compiles a standard binary for your current operating system.
- `make build-cross`: Compiles release binaries for Windows, macOS, and Linux using Docker.
- `make clean`: Cleans up local build artifacts and cache folders.

## Building Manually (Native Only)

If you don't have Docker installed and only want to build an executable for your *current* operating system, you can use the standard Fyne CLI:

```bash
go install fyne.io/fyne/v2/cmd/fyne@latest

# Run this inside the project root:
fyne package -os darwin -appID com.openwhats.native -src ./cmd/open-whats/
# (Replace 'darwin' with 'windows' or 'linux' depending on your host OS)
```

## Data Storage Notes

When compiled for production using `fyne package` or `fyne-cross`, the application runs in a protected environment. Instead of using a local `db/` folder relative to the binary, Open-Whats automatically resolves the host's standard application data directory using `os.UserConfigDir()`. 

- **macOS**: `~/Library/Application Support/open-whats/store.db`
- **Windows**: `C:\Users\Username\AppData\Roaming\open-whats\store.db`
- **Linux**: `~/.config/open-whats/store.db`

This ensures that the app conforms to the standard conventions of desktop apps on all operating systems.
