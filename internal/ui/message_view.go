package ui

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"open-whats/internal/domain"
)

func (ui *AppUI) loadChatHistory(jid string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	msgs, err := ui.msgStore.GetMessages(ctx, jid, 100)
	if err != nil {
		fmt.Println("Error loading history:", err)
		return
	}

	ui.allMsgs = msgs
	ui.filterMessages(ui.searchMsg.Text) // Apply active search if any
}

func (ui *AppUI) filterMessages(query string) {
	if query == "" {
		ui.filteredMsg = ui.allMsgs
	} else {
		query = strings.ToLower(query)
		var filtered []domain.Message
		for _, m := range ui.allMsgs {
			if strings.Contains(strings.ToLower(m.Text), query) {
				filtered = append(filtered, m)
			}
		}
		ui.filteredMsg = filtered
	}
	ui.refreshMessagesList()
}

func (ui *AppUI) refreshMessagesList() {
	ui.msgVBox.Objects = nil
	var prevSenderJID string
	for _, msg := range ui.filteredMsg {
		// Prepare sender name
		senderName := msg.SenderJID
		if msg.IsFromMe {
			senderName = "Me"
		} else if name, ok := ui.contactMap[msg.SenderJID]; ok && name != "" {
			senderName = name
		} else if msg.SenderName != "" {
			senderName = msg.SenderName
			// Optimistically save it
			ui.contactMap[msg.SenderJID] = msg.SenderName
		} else {
			// Clean up JID if it's a raw JID
			lookupJID := msg.SenderJID
			if !strings.Contains(lookupJID, "@") {
				lookupJID += "@s.whatsapp.net"
			}
			if c, err := ui.client.GetContact(context.Background(), lookupJID); err == nil && (c.Name != "" || c.PushName != "") {
				if c.Name != "" {
					senderName = c.Name
				} else {
					senderName = c.PushName
				}
				ui.contactMap[msg.SenderJID] = senderName
			} else {
				if strings.Contains(senderName, "@") {
					senderName = strings.Split(senderName, "@")[0]
				}
				if strings.Contains(senderName, ":") {
					senderName = strings.Split(senderName, ":")[0]
				}
				senderName = "+" + senderName
			}
		}

		displaySenderName := senderName
		if msg.SenderJID == prevSenderJID {
			displaySenderName = ""
		}
		prevSenderJID = msg.SenderJID

		var avatar []byte
		if !msg.IsFromMe {
			lookupJID := msg.SenderJID
			if !strings.Contains(lookupJID, "@") {
				lookupJID += "@s.whatsapp.net"
			}
			if av, ok := ui.avatarMap[lookupJID]; ok {
				avatar = av
			} else {
				// Async fetch avatar
				ui.avatarMap[lookupJID] = nil // Avoid repeated requests
				go func(jid string) {
					url, err := ui.client.GetProfilePicture(context.Background(), jid)
					if err == nil && url != "" {
						resp, err := http.Get(url)
						if err == nil {
							if resp.StatusCode == 200 {
								data, err := io.ReadAll(resp.Body)
								if err == nil && len(data) > 0 {
									ui.avatarMap[jid] = applyCircularMask(data)
									fyne.Do(func() {
										ui.refreshMessagesList()
									})
								}
							}
							resp.Body.Close()
						}
					}
				}(lookupJID)
			}
		}

		renderer := GetMessageRenderer(msg)
		ui.msgVBox.Add(renderer.Render(msg, displaySenderName, avatar))
		
		// Add y-padding
		spacer := canvas.NewRectangle(color.Transparent)
		spacer.SetMinSize(fyne.NewSize(1, 10))
		ui.msgVBox.Add(spacer)
	}
	ui.msgVBox.Refresh()
	if len(ui.filteredMsg) > 0 {
		go func() {
			time.Sleep(100 * time.Millisecond)
			fyne.Do(func() {
				ui.msgScroll.ScrollToBottom()
			})
		}()
	}
}

func (ui *AppUI) sendMessage() {
	text := ui.msgEntry.Text
	if text == "" || ui.activeJID == "" {
		return
	}

	ui.msgEntry.SetText("")

	// Optimistic UI update
	msg := domain.Message{
		ChatJID:   ui.activeJID,
		SenderJID: "Me",
		Text:      text,
		IsFromMe:  true,
		Timestamp: time.Now(),
	}
	ui.allMsgs = append(ui.allMsgs, msg)
	ui.filterMessages(ui.searchMsg.Text)

	// Update chat list preview natively
	ui.updateChatPreview(ui.activeJID, text, time.Now())

	go func() {
		err := ui.client.SendMessage(context.Background(), ui.activeJID, text)
		if err != nil {
			fmt.Println("Error sending message:", err)
		}
	}()
}

func (ui *AppUI) onMessageReceived(msg domain.Message) {
	fyne.Do(func() {
		// Let chat_list handle updating the sidebar
		ui.updateChatPreview(msg.ChatJID, msg.Text, msg.Timestamp)

		if msg.ChatJID != ui.activeJID {
			return
		}
		
		updated := false
		for i, m := range ui.allMsgs {
			if m.ID == msg.ID {
				ui.allMsgs[i] = msg
				updated = true
				break
			}
		}
		if !updated {
			ui.allMsgs = append(ui.allMsgs, msg)
		}
		
		ui.filterMessages(ui.searchMsg.Text)
	})
}
