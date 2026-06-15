# Open-Whats Native

Open-Whats is a blazingly fast, lightweight, and native desktop WhatsApp client built with Go and [Fyne](https://fyne.io/). It interfaces directly with the WhatsApp Multi-Device protocol via the fantastic [whatsmeow](https://github.com/tulir/whatsmeow) library.

Unlike official desktop clients which are heavy Electron or web-wrapped applications, Open-Whats compiles to a native binary, utilizing hardware-accelerated rendering and a local SQLite database for instant boot-ups.

## Features

- **Blazing Fast**: Native Go & Fyne architecture means instant boot and low memory footprint.
- **Offline First**: All messages and chats are cached instantly to a local SQLite database. No more waiting for WhatsApp servers to sync before reading old messages!
- **Multi-Device Support**: Scan once, stay logged in forever. Doesn't rely on your phone being online.
- **Native GUI**: Custom-built chat bubbles, dynamic text scaling, and clean sidebar navigation.
- **Sticker Support**: Parses and natively renders static `.webp` WhatsApp stickers.
- **Smart Contact Resolution**: Automatically maps raw phone numbers to WhatsApp Push Names when they aren't in your address book.

## Installation

### Prerequisites
- Go 1.20 or higher
- C compiler (GCC/Clang) for SQLite CGO bindings

### Build from Source
```bash
# Clone the repository
git clone https://github.com/yourusername/open-whats.git
cd open-whats

# Install dependencies
go mod tidy

# Build the native binary
go build -o open-whats ./cmd/open-whats/main.go

# Run
./open-whats
```

## Usage
1. Launch the application.
2. A QR Code will pop up natively in the center of the screen.
3. Open WhatsApp on your phone -> Linked Devices -> Link a Device.
4. Scan the QR code.
5. Watch the top left corner as it syncs your history! 

## Architecture
This project strictly follows SOLID principles and Domain-Driven Design:
- **Domain Layer**: Core business models (`Message`, `ChatPreview`, `Contact`).
- **Data Layer (Store)**: Handles localized SQLite caching and transparently fetches older data.
- **Infrastructure (WhatsApp)**: Encapsulates all `whatsmeow` complexities, protobuf decoding, and history syncing.
- **Presentation (UI)**: Pure Fyne rendering using the Strategy Pattern for scalable component types (Text, Stickers, Images, etc.).

## Roadmap
See the planned features in our issue tracker, including media attachments, profile viewing, and status support.

## Disclaimer
This is an unofficial project. It is not affiliated, associated, authorized, endorsed by, or in any way officially connected with WhatsApp Inc., or any of its subsidiaries or its affiliates. The official WhatsApp website can be found at https://www.whatsapp.com.
