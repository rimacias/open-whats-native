package domain

import (
	"context"
	"time"
)

type Contact struct {
	JID      string `json:"jid"`
	Name     string `json:"name"`
	PushName string `json:"pushName"`
}

type Message struct {
	ID        string
	ChatJID   string
	SenderJID string
	SenderName string // Added to store push name
	Text      string
	IsSticker bool
	MediaURL  string
	Timestamp time.Time
	IsFromMe  bool
}

// ChatPreview represents an active chat in the sidebar.
type ChatPreview struct {
	JID       string
	Name      string
	LastMsg   string
	Timestamp time.Time
}

// WhatsAppClient defines the behavior for the WhatsApp integration.
type WhatsAppClient interface {
	SendMessage(ctx context.Context, jid string, text string) error
	GetContacts(ctx context.Context) ([]Contact, error)
	RegisterMessageCallback(callback func(msg Message))
	RegisterLoginCallbacks(onQR func(code string), onLogin func())
	RegisterSyncCallback(onSync func(isSyncing bool))
	Logout(ctx context.Context) error
}

// MessageStore defines how we store and retrieve local message history.
type MessageStore interface {
	SaveMessage(ctx context.Context, msg Message) error
	GetMessages(ctx context.Context, chatJID string, limit int) ([]Message, error)
	GetChats(ctx context.Context) ([]ChatPreview, error)
}
