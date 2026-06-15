package whatsapp

import (
	"context"
	"fmt"
	"os"
	"time"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/proto"
	
	"encoding/base64"
	"open-whats/internal/domain"
)

// Client is an adapter for whatsmeow.Client
type Client struct {
	client          *whatsmeow.Client
	msgStore        domain.MessageStore
	messageCallback func(msg domain.Message)
	onQR            func(code string)
	onLogin         func()
	onSync          func(isSyncing bool)
}

// NewClient creates a new whatsapp client adapter
func NewClient(deviceStore *store.Device, msgStore domain.MessageStore) *Client {
	clientLog := waLog.Stdout("Client", "DEBUG", true)
	client := whatsmeow.NewClient(deviceStore, clientLog)
	
	c := &Client{
		client:   client,
		msgStore: msgStore,
	}

	c.client.AddEventHandler(c.eventHandler)

	return c
}

// Connect starts the client connection
func (c *Client) Connect(ctx context.Context) error {
	if c.client.Store.ID == nil {
		qrChan, _ := c.client.GetQRChannel(ctx)
		err := c.client.Connect()
		if err != nil {
			return fmt.Errorf("failed to connect: %w", err)
		}
		go func() {
			for evt := range qrChan {
				if evt.Event == "code" {
					if c.onQR != nil {
						c.onQR(evt.Code)
					}
				} else if evt.Event == "success" {
					if c.onLogin != nil {
						c.onLogin()
					}
				}
			}
		}()
	} else {
		err := c.client.Connect()
		if err != nil {
			return fmt.Errorf("failed to connect: %w", err)
		}
		if c.onLogin != nil {
			c.onLogin()
		}
	}
	return nil
}

// Disconnect gracefully stops the client
func (c *Client) Disconnect() {
	c.client.Disconnect()
}

// Logout implements domain.WhatsAppClient
func (c *Client) Logout(ctx context.Context) error {
	err := c.client.Logout(ctx)
	if err != nil {
		return fmt.Errorf("failed to logout: %w", err)
	}
	// Attempt to remove the local database file to ensure a clean slate for next login
	_ = os.Remove("db/store.db")
	return nil
}

// SendMessage implements domain.WhatsAppClient
func (c *Client) SendMessage(ctx context.Context, jid string, text string) error {
	targetJID, err := types.ParseJID(jid)
	if err != nil {
		return fmt.Errorf("invalid JID: %w", err)
	}

	msg := &waE2E.Message{
		Conversation: proto.String(text),
	}

	resp, err := c.client.SendMessage(ctx, targetJID, msg)
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	// Save sent message locally
	c.msgStore.SaveMessage(ctx, domain.Message{
		ID:        resp.ID,
		ChatJID:   targetJID.ToNonAD().String(),
		SenderJID: c.client.Store.ID.ToNonAD().String(),
		Text:      text,
		IsSticker: false,
		Timestamp: resp.Timestamp,
		IsFromMe:  true,
	})

	return nil
}

// GetContacts implements domain.WhatsAppClient
func (c *Client) GetContacts(ctx context.Context) ([]domain.Contact, error) {
	if c.client.Store.ID == nil {
		return nil, fmt.Errorf("client not logged in")
	}
	if c.client.Store.Contacts == nil {
		return nil, fmt.Errorf("contacts store is nil")
	}
	contactMap, err := c.client.Store.Contacts.GetAllContacts(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get contacts: %w", err)
	}

	var contacts []domain.Contact
	for jid, info := range contactMap {
		contacts = append(contacts, domain.Contact{
			JID:      jid.ToNonAD().String(),
			Name:     info.FullName,
			PushName: info.PushName,
		})
	}

	groups, err := c.client.GetJoinedGroups(ctx)
	if err == nil {
		for _, g := range groups {
			contacts = append(contacts, domain.Contact{
				JID:      g.JID.ToNonAD().String(),
				Name:     g.Name,
				PushName: g.Name,
			})
		}
	}

	return contacts, nil
}

// RegisterMessageCallback implements domain.WhatsAppClient
func (c *Client) RegisterMessageCallback(callback func(msg domain.Message)) {
	c.messageCallback = callback
}

// RegisterLoginCallbacks implements domain.WhatsAppClient
func (c *Client) RegisterLoginCallbacks(onQR func(code string), onLogin func()) {
	c.onQR = onQR
	c.onLogin = onLogin
}

// RegisterSyncCallback implements domain.WhatsAppClient
func (c *Client) RegisterSyncCallback(onSync func(isSyncing bool)) {
	c.onSync = onSync
}

// eventHandler processes incoming WhatsApp events
func (c *Client) eventHandler(evt interface{}) {
	switch v := evt.(type) {
	case *events.Message:
		var text string
		var isSticker bool
		var mediaURL string

		if v.Message.GetConversation() != "" {
			text = v.Message.GetConversation()
		} else if v.Message.GetExtendedTextMessage() != nil {
			text = v.Message.GetExtendedTextMessage().GetText()
		} else if v.Message.GetStickerMessage() != nil {
			isSticker = true
			data, err := c.client.Download(context.Background(), v.Message.GetStickerMessage())
			if err == nil {
				mediaURL = "data:image/webp;base64," + base64.StdEncoding.EncodeToString(data)
			}
		}

		if text != "" || isSticker {
			chatJID := v.Info.Chat.ToNonAD().String()
			
			domainMsg := domain.Message{
				ID:         v.Info.ID,
				ChatJID:    chatJID,
				SenderJID:  v.Info.Sender.ToNonAD().String(),
				SenderName: v.Info.PushName,
				Text:       text,
				IsSticker: isSticker,
				MediaURL:  mediaURL,
				Timestamp: v.Info.Timestamp,
				IsFromMe:  v.Info.IsFromMe,
			}

			// Store it locally so history is maintained
			c.msgStore.SaveMessage(context.Background(), domainMsg)

			if c.messageCallback != nil {
				c.messageCallback(domainMsg)
			}
		}
	
	case *events.HistorySync:
		if c.onSync != nil {
			c.onSync(true)
		}
		// Basic history sync handling - this fires when linking a new device
		for _, conv := range v.Data.GetConversations() {
			chatJID, _ := types.ParseJID(conv.GetID())
			for _, historyMsg := range conv.GetMessages() {
				if historyMsg.GetMessage() == nil || historyMsg.GetMessage().GetMessage() == nil {
					continue
				}
				
				msgData := historyMsg.GetMessage().GetMessage()
				var text string
				var isSticker bool
				var mediaURL string
				
				if msgData.GetConversation() != "" {
					text = msgData.GetConversation()
				} else if msgData.GetExtendedTextMessage() != nil {
					text = msgData.GetExtendedTextMessage().GetText()
				} else if msgData.GetStickerMessage() != nil {
					isSticker = true
					data, err := c.client.Download(context.Background(), msgData.GetStickerMessage())
					if err == nil {
						mediaURL = "data:image/webp;base64," + base64.StdEncoding.EncodeToString(data)
					}
				}

				if text != "" || isSticker {
					info := historyMsg.GetMessage()
					isFromMe := info.GetKey().GetFromMe()
					senderJID := chatJID.ToNonAD().String()
					if isFromMe {
						if c.client.Store.ID != nil {
							senderJID = c.client.Store.ID.ToNonAD().String()
						}
					} else {
						if info.GetKey().GetParticipant() != "" {
							senderJID = info.GetKey().GetParticipant()
						} else if info.GetParticipant() != "" {
							senderJID = info.GetParticipant()
						}
					}

					domainMsg := domain.Message{
						ID:        info.GetKey().GetID(),
						ChatJID:   chatJID.ToNonAD().String(),
						SenderJID: senderJID,
						SenderName: info.GetPushName(),
						Text:      text,
						IsSticker: isSticker,
						MediaURL:  mediaURL,
						Timestamp: time.Unix(int64(info.GetMessageTimestamp()), 0),
						IsFromMe:  isFromMe,
					}
					c.msgStore.SaveMessage(context.Background(), domainMsg)
				}
			}
		}
		if c.onSync != nil {
			c.onSync(false)
		}
	}
}
