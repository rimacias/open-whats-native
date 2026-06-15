package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"open-whats/internal/domain"
)

func (ui *AppUI) fetchChats() {
	if !ui.client.IsLoggedIn() {
		return
	}
	
	fyne.Do(func() { ui.syncContainer.Show() })
	defer fyne.Do(func() { ui.syncContainer.Hide() })

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	chats, err := ui.msgStore.GetChats(ctx)
	if err != nil {
		fmt.Println("Error fetching chats:", err)
		return
	}

	contacts, _ := ui.client.GetContacts(ctx)
	ui.allContacts = contacts
	
	for _, c := range contacts {
		name := c.Name
		if name == "" {
			name = c.PushName // Fallback to WhatsApp registered name only if phonebook name is missing
		}
		ui.contactMap[c.JID] = name
	}

	for i, ch := range chats {
		if name, ok := ui.contactMap[ch.JID]; ok && name != "" {
			chats[i].Name = name
		} else {
			chats[i].Name = ch.JID // Fallback to JID
		}
	}

	fyne.Do(func() {
		ui.allChats = chats
		ui.filteredCh = chats
		ui.chatList.Refresh()
	})
}

func (ui *AppUI) filterChats(query string) {
	if query == "" {
		ui.filteredCh = ui.allChats
		ui.chatList.Refresh()
		return
	}
	query = strings.ToLower(query)
	var filtered []domain.ChatPreview
	for _, c := range ui.allChats {
		if strings.Contains(strings.ToLower(c.Name), query) || strings.Contains(strings.ToLower(c.LastMsg), query) {
			filtered = append(filtered, c)
		}
	}
	ui.filteredCh = filtered
	ui.chatList.Refresh()
}

func (ui *AppUI) updateChatPreview(jid, text string, ts time.Time) {
	found := false
	for i, c := range ui.allChats {
		if c.JID == jid {
			ui.allChats[i].LastMsg = text
			ui.allChats[i].Timestamp = ts
			found = true
			break
		}
	}
	if !found {
		// Resolve name for new chat
		name := jid
		if n, ok := ui.contactMap[jid]; ok && n != "" {
			name = n
		}
		
		// New chat
		ui.allChats = append([]domain.ChatPreview{{JID: jid, Name: name, LastMsg: text, Timestamp: ts}}, ui.allChats...)
	} else {
		// Resort
		sortChatsByTime(ui.allChats)
	}
	ui.filterChats(ui.searchChat.Text)
}

func sortChatsByTime(chats []domain.ChatPreview) {
	for i := 1; i < len(chats); i++ {
		j := i
		for j > 0 && chats[j].Timestamp.After(chats[j-1].Timestamp) {
			chats[j], chats[j-1] = chats[j-1], chats[j]
			j--
		}
	}
}
