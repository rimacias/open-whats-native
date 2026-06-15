package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"open-whats/internal/store"
	"open-whats/internal/ui"
	"open-whats/internal/whatsapp"
)

func main() {
	ctx := context.Background()

	configDir, err := os.UserConfigDir()
	if err != nil {
		configDir = "."
	}
	appDir := filepath.Join(configDir, "open-whats")
	if err := os.MkdirAll(appDir, 0755); err != nil {
		fmt.Printf("Fatal error creating config directory: %v\n", err)
		os.Exit(1)
	}
	dbPath := filepath.Join(appDir, "store.db")

	// 1. Initialize the SQLite store
	localStore, err := store.InitDatabase(ctx, dbPath)
	if err != nil {
		fmt.Printf("Fatal error initializing database: %v\n", err)
		os.Exit(1)
	}

	// 2. Initialize the WhatsApp Client Adapter using our store
	waClient := whatsapp.NewClient(localStore.GetDevice(), localStore)
	
	// 3. Initialize the Native Fyne UI (registers callbacks)
	appUI := ui.NewAppUI(waClient, localStore)

	// Connect to WhatsApp (now non-blocking if QR code needs to be scanned)
	if err := waClient.Connect(ctx); err != nil {
		fmt.Printf("Fatal error connecting to WhatsApp: %v\n", err)
		os.Exit(1)
	}
	
	defer waClient.Disconnect()

	fmt.Println("Launching Native Fyne Application...")
	// Start() blocks until the GUI window is closed by the user
	appUI.Start()

	fmt.Println("\nShutting down open-whats...")
}
