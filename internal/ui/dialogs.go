package ui

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"github.com/skip2/go-qrcode"

	"open-whats/internal/domain"
)

func (ui *AppUI) showNewChatDialog() {
	var validContacts []domain.Contact
	for _, c := range ui.allContacts {
		if c.Name != "" || c.PushName != "" {
			validContacts = append(validContacts, c)
		}
	}

	filtered := validContacts

	searchEntry := widget.NewEntry()
	searchEntry.SetPlaceHolder("Search contacts...")

	list := widget.NewList(
		func() int { return len(filtered) },
		func() fyne.CanvasObject {
			return widget.NewLabel("Contact Name")
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			c := filtered[i]
			name := c.Name
			if name == "" {
				name = c.PushName
			}
			o.(*widget.Label).SetText(name)
		},
	)

	var d dialog.Dialog

	searchEntry.OnChanged = func(s string) {
		s = strings.ToLower(s)
		var f []domain.Contact
		for _, c := range validContacts {
			name := c.Name
			if name == "" { name = c.PushName }
			if strings.Contains(strings.ToLower(name), s) {
				f = append(f, c)
			}
		}
		filtered = f
		list.Refresh()
	}

	list.OnSelected = func(id widget.ListItemID) {
		c := filtered[id]
		name := c.Name
		if name == "" { name = c.PushName }

		ui.updateChatPreview(c.JID, "", time.Now())
		ui.activeJID = c.JID
		ui.chatTitle.SetText(fmt.Sprintf("Chat: %s", name))
		ui.loadChatHistory(c.JID)
		
		d.Hide()
	}

	content := container.NewBorder(searchEntry, nil, nil, nil, list)
	content.Resize(fyne.NewSize(300, 400))

	d = dialog.NewCustom("New Chat", "Cancel", content, ui.mainWindow)
	d.Resize(fyne.NewSize(350, 500))
	d.Show()
}

func (ui *AppUI) logout() {
	dialog.ShowConfirm("Logout", "Are you sure you want to log out? This will delete your local session data.", func(b bool) {
		if b {
			err := ui.client.Logout(context.Background())
			if err != nil {
				dialog.ShowError(err, ui.mainWindow)
				return
			}
			
			// Clear UI state immediately
			ui.allChats = nil
			ui.filteredCh = nil
			ui.allMsgs = nil
			ui.filteredMsg = nil
			ui.chatList.Refresh()
			ui.msgVBox.Objects = nil
			ui.msgVBox.Refresh()
			ui.chatTitle.SetText("Select a chat to start messaging")
			
			dialog.ShowInformation("Logged Out", "You have been logged out. Please restart the application to link a new device.", ui.mainWindow)
			// Close the app after a short delay so they can read the dialog
			go func() {
				time.Sleep(3 * time.Second)
				fyne.Do(func() {
					ui.fyneApp.Quit()
				})
			}()
		}
	}, ui.mainWindow)
}

func (ui *AppUI) showQRCode(code string) {
	png, err := qrcode.Encode(code, qrcode.Medium, 256)
	if err == nil {
		img := canvas.NewImageFromReader(bytes.NewReader(png), "qr.png")
		img.FillMode = canvas.ImageFillContain
		img.SetMinSize(fyne.NewSize(300, 300))

		if ui.qrDialog != nil {
			ui.qrDialog.Hide()
		}
		
		content := container.NewVBox(
			widget.NewLabelWithStyle("Scan this QR code with WhatsApp", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
			img,
		)
		ui.qrDialog = dialog.NewCustom("Login Required", "Close", content, ui.mainWindow)
		ui.qrDialog.Show()
	}
}

func (ui *AppUI) showAttachDialog() {
	if ui.activeJID == "" {
		dialog.ShowInformation("No Chat", "Please select a chat before attaching a file.", ui.mainWindow)
		return
	}

	d := dialog.NewFileOpen(func(uc fyne.URIReadCloser, err error) {
		if err != nil || uc == nil {
			return
		}
		defer uc.Close()

		// Read the file data
		data, err := io.ReadAll(uc)
		if err != nil {
			dialog.ShowError(err, ui.mainWindow)
			return
		}

		// Basic MIME type detection
		mimeType := "image/jpeg"
		ext := strings.ToLower(uc.URI().Extension())
		if ext == ".png" {
			mimeType = "image/png"
		} else if ext == ".webp" {
			mimeType = "image/webp"
		}

		go func() {
			err := ui.client.SendImage(context.Background(), ui.activeJID, data, mimeType)
			if err != nil {
				fmt.Println("Error sending image:", err)
			} else {
				// We reload history to update UI, or we can just fetch the new message.
				// For simplicity, reload history
				ui.loadChatHistory(ui.activeJID)
				ui.updateChatPreview(ui.activeJID, "📷 Photo", time.Now())
			}
		}()
	}, ui.mainWindow)

	// Set file filter if needed
	d.Show()
}
