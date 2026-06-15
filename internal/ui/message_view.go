package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"fyne.io/fyne/v2"
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
			if strings.Contains(senderName, "@") {
				senderName = strings.Split(senderName, "@")[0]
			}
		}

		renderer := GetMessageRenderer(msg)
		ui.msgVBox.Add(renderer.Render(msg, senderName))
	}
	ui.msgVBox.Refresh()
	if len(ui.filteredMsg) > 0 {
		ui.msgScroll.ScrollToBottom()
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
