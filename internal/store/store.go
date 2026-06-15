package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/store/sqlstore"
	waLog "go.mau.fi/whatsmeow/util/log"
	
	"open-whats/internal/domain"
)

type LocalStore struct {
	db          *sql.DB
	deviceStore *store.Device
}

// InitDatabase initializes our custom SQLite database and whatsmeow's store.
func InitDatabase(ctx context.Context, path string) (*LocalStore, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, fmt.Errorf("failed to create db directory: %w", err)
	}

	dsn := fmt.Sprintf("file:%s?_foreign_keys=on", path)
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Initialize our custom tables
	if err := initTables(ctx, db); err != nil {
		return nil, fmt.Errorf("failed to init tables: %w", err)
	}

	dbLog := waLog.Stdout("Database", "DEBUG", true)
	container := sqlstore.NewWithDB(db, "sqlite3", dbLog)
	
	if err := container.Upgrade(ctx); err != nil {
		return nil, fmt.Errorf("failed to upgrade sqlstore: %w", err)
	}

	deviceStore, err := container.GetFirstDevice(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get device from store: %w", err)
	}

	if deviceStore == nil {
		deviceStore = container.NewDevice()
	}

	return &LocalStore{
		db:          db,
		deviceStore: deviceStore,
	}, nil
}

func initTables(ctx context.Context, db *sql.DB) error {
	query := `
	CREATE TABLE IF NOT EXISTS messages (
		id TEXT PRIMARY KEY,
		chat_jid TEXT NOT NULL,
		sender_jid TEXT NOT NULL,
		text TEXT,
		is_sticker BOOLEAN,
		media_url TEXT,
		timestamp DATETIME,
		is_from_me BOOLEAN,
		sender_name TEXT,
		reactions TEXT
	);
	CREATE INDEX IF NOT EXISTS idx_messages_chat_jid ON messages(chat_jid);
	CREATE INDEX IF NOT EXISTS idx_messages_timestamp ON messages(timestamp);
	`
	_, err := db.ExecContext(ctx, query)
	
	// Migration: Add sender_name and reactions if missing
	_, _ = db.ExecContext(ctx, "ALTER TABLE messages ADD COLUMN sender_name TEXT")
	_, _ = db.ExecContext(ctx, "ALTER TABLE messages ADD COLUMN reactions TEXT")
	
	return err
}

func (ls *LocalStore) GetDevice() *store.Device {
	return ls.deviceStore
}

// SaveMessage implements domain.MessageStore
func (ls *LocalStore) SaveMessage(ctx context.Context, msg domain.Message) error {
	reactionsJSON, _ := json.Marshal(msg.Reactions)
	query := `
		INSERT OR REPLACE INTO messages (id, chat_jid, sender_jid, sender_name, text, is_sticker, media_url, timestamp, is_from_me, reactions)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := ls.db.ExecContext(ctx, query,
		msg.ID, msg.ChatJID, msg.SenderJID, msg.SenderName, msg.Text, msg.IsSticker, msg.MediaURL, msg.Timestamp, msg.IsFromMe, string(reactionsJSON),
	)
	return err
}

// GetMessages implements domain.MessageStore
func (ls *LocalStore) GetMessages(ctx context.Context, chatJID string, limit int) ([]domain.Message, error) {
	query := `
		SELECT id, chat_jid, sender_jid, sender_name, text, is_sticker, media_url, timestamp, is_from_me, reactions
		FROM messages
		WHERE chat_jid = ?
		ORDER BY timestamp ASC
		LIMIT ?
	`
	rows, err := ls.db.QueryContext(ctx, query, chatJID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var msgs []domain.Message
	for rows.Next() {
		var m domain.Message
		var nullSenderName sql.NullString
		var nullReactions sql.NullString
		err := rows.Scan(&m.ID, &m.ChatJID, &m.SenderJID, &nullSenderName, &m.Text, &m.IsSticker, &m.MediaURL, &m.Timestamp, &m.IsFromMe, &nullReactions)
		if err != nil {
			return nil, err
		}
		if nullSenderName.Valid {
			m.SenderName = nullSenderName.String
		}
		if nullReactions.Valid && nullReactions.String != "" {
			json.Unmarshal([]byte(nullReactions.String), &m.Reactions)
		}
		msgs = append(msgs, m)
	}
	return msgs, nil
}

// GetMessage implements domain.MessageStore
func (ls *LocalStore) GetMessage(ctx context.Context, id string) (*domain.Message, error) {
	query := `
		SELECT id, chat_jid, sender_jid, sender_name, text, is_sticker, media_url, timestamp, is_from_me, reactions
		FROM messages
		WHERE id = ?
	`
	row := ls.db.QueryRowContext(ctx, query, id)

	var m domain.Message
	var nullSenderName sql.NullString
	var nullReactions sql.NullString
	err := row.Scan(&m.ID, &m.ChatJID, &m.SenderJID, &nullSenderName, &m.Text, &m.IsSticker, &m.MediaURL, &m.Timestamp, &m.IsFromMe, &nullReactions)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	if nullSenderName.Valid {
		m.SenderName = nullSenderName.String
	}
	if nullReactions.Valid && nullReactions.String != "" {
		json.Unmarshal([]byte(nullReactions.String), &m.Reactions)
	}
	return &m, nil
}

// UpdateMessageReactions implements domain.MessageStore
func (ls *LocalStore) UpdateMessageReactions(ctx context.Context, id string, reactions []domain.Reaction) error {
	reactionsJSON, _ := json.Marshal(reactions)
	query := `UPDATE messages SET reactions = ? WHERE id = ?`
	_, err := ls.db.ExecContext(ctx, query, string(reactionsJSON), id)
	return err
}

// GetChats implements domain.MessageStore
func (ls *LocalStore) GetChats(ctx context.Context) ([]domain.ChatPreview, error) {
	query := `
		SELECT chat_jid, text, is_sticker, timestamp
		FROM (
			SELECT chat_jid, text, is_sticker, timestamp,
				   ROW_NUMBER() OVER (PARTITION BY chat_jid ORDER BY timestamp DESC) as rn
			FROM messages
		)
		WHERE rn = 1
		ORDER BY timestamp DESC;
	`
	rows, err := ls.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var chats []domain.ChatPreview
	for rows.Next() {
		var chat domain.ChatPreview
		var isSticker bool
		err := rows.Scan(&chat.JID, &chat.LastMsg, &isSticker, &chat.Timestamp)
		if err != nil {
			return nil, err
		}
		if isSticker {
			chat.LastMsg = "🖼️ Sticker"
		}
		chats = append(chats, chat)
	}

	return chats, nil
}

// ClearAllData implements domain.MessageStore
func (ls *LocalStore) ClearAllData(ctx context.Context) error {
	_, err := ls.db.ExecContext(ctx, "DELETE FROM messages")
	if err != nil {
		return err
	}
	_, err = ls.db.ExecContext(ctx, "DELETE FROM contacts")
	return err
}
