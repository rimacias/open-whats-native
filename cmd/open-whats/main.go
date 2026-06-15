package main

import (
	"context"
	"fmt"
	"os"

	"open-whats/internal/store"
	"open-whats/internal/ui"
	"open-whats/internal/whatsapp"
)

func main() {
	ctx := context.Background()

	// 1. Initialize the SQLite store (now includes our local messages table)
	localStore, err := store.InitDatabase(ctx, "store.db")
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
