package store

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ChatSession is a persisted Market Chat thread for one user.
type ChatSession struct {
	ID        string    `gorm:"primaryKey;size:64" json:"id"`
	UserID    string    `gorm:"index;size:64;not null" json:"user_id"`
	Title     string    `gorm:"size:200;default:''" json:"title"`
	ModelID   string    `gorm:"size:128;default:''" json:"model_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ChatMessage is one turn inside a ChatSession.
type ChatMessage struct {
	ID        string    `gorm:"primaryKey;size:64" json:"id"`
	SessionID string    `gorm:"index;size:64;not null" json:"session_id"`
	UserID    string    `gorm:"index;size:64;not null" json:"user_id"`
	Role      string    `gorm:"size:32;not null" json:"role"` // user | assistant | tool
	Content   string    `gorm:"type:text" json:"content"`
	ToolName  string    `gorm:"size:128;default:''" json:"tool_name,omitempty"`
	ToolCallID string   `gorm:"size:128;default:''" json:"tool_call_id,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type ChatSessionStore struct {
	db *gorm.DB
}

func NewChatSessionStore(db *gorm.DB) *ChatSessionStore {
	return &ChatSessionStore{db: db}
}

func (s *ChatSessionStore) initTables() error {
	return s.db.AutoMigrate(&ChatSession{}, &ChatMessage{})
}

func (s *ChatSessionStore) List(userID string, limit int) ([]ChatSession, error) {
	return s.Search(userID, "", limit)
}

// Search lists sessions optionally filtered by title / message content.
func (s *ChatSessionStore) Search(userID, query string, limit int) ([]ChatSession, error) {
	if limit <= 0 || limit > 100 {
		limit = 40
	}
	query = strings.TrimSpace(query)
	var rows []ChatSession
	db := s.db.Where("user_id = ?", userID)
	if query != "" {
		like := "%" + query + "%"
		db = db.Where(
			"title LIKE ? OR id IN (SELECT DISTINCT session_id FROM chat_messages WHERE user_id = ? AND content LIKE ?)",
			like, userID, like,
		)
	}
	err := db.Order("updated_at DESC").Limit(limit).Find(&rows).Error
	return rows, err
}

func (s *ChatSessionStore) Get(userID, id string) (*ChatSession, error) {
	var row ChatSession
	if err := s.db.Where("id = ? AND user_id = ?", id, userID).First(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (s *ChatSessionStore) Create(userID, title, modelID string) (*ChatSession, error) {
	now := time.Now().UTC()
	row := &ChatSession{
		ID:        uuid.NewString(),
		UserID:    userID,
		Title:     strings.TrimSpace(title),
		ModelID:   strings.TrimSpace(modelID),
		CreatedAt: now,
		UpdatedAt: now,
	}
	if row.Title == "" {
		row.Title = "New chat"
	}
	if err := s.db.Create(row).Error; err != nil {
		return nil, err
	}
	return row, nil
}

func (s *ChatSessionStore) Rename(userID, id, title string) error {
	title = strings.TrimSpace(title)
	if title == "" {
		return fmt.Errorf("title required")
	}
	return s.db.Model(&ChatSession{}).
		Where("id = ? AND user_id = ?", id, userID).
		Updates(map[string]interface{}{
			"title":      title,
			"updated_at": time.Now().UTC(),
		}).Error
}

func (s *ChatSessionStore) Delete(userID, id string) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("session_id = ? AND user_id = ?", id, userID).Delete(&ChatMessage{}).Error; err != nil {
			return err
		}
		return tx.Where("id = ? AND user_id = ?", id, userID).Delete(&ChatSession{}).Error
	})
}

func (s *ChatSessionStore) ListMessages(userID, sessionID string, limit int) ([]ChatMessage, error) {
	if _, err := s.Get(userID, sessionID); err != nil {
		return nil, err
	}
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	var rows []ChatMessage
	err := s.db.Where("session_id = ? AND user_id = ?", sessionID, userID).
		Order("created_at ASC").
		Limit(limit).
		Find(&rows).Error
	return rows, err
}

func (s *ChatSessionStore) AppendMessage(userID, sessionID, role, content string) (*ChatMessage, error) {
	return s.AppendToolMessage(userID, sessionID, role, content, "", "")
}

// AppendToolMessage stores a chat turn, optionally tagging a tool call result.
func (s *ChatSessionStore) AppendToolMessage(userID, sessionID, role, content, toolName, toolCallID string) (*ChatMessage, error) {
	if _, err := s.Get(userID, sessionID); err != nil {
		return nil, err
	}
	role = strings.TrimSpace(role)
	if role == "" {
		role = "user"
	}
	msg := &ChatMessage{
		ID:         uuid.NewString(),
		SessionID:  sessionID,
		UserID:     userID,
		Role:       role,
		Content:    content,
		ToolName:   strings.TrimSpace(toolName),
		ToolCallID: strings.TrimSpace(toolCallID),
		CreatedAt:  time.Now().UTC(),
	}
	if err := s.db.Create(msg).Error; err != nil {
		return nil, err
	}
	updates := map[string]interface{}{"updated_at": time.Now().UTC()}
	var sess ChatSession
	if err := s.db.Where("id = ? AND user_id = ?", sessionID, userID).First(&sess).Error; err == nil {
		if sess.Title == "" || sess.Title == "New chat" {
			if role == "user" {
				title := truncateRunes(strings.TrimSpace(content), 40)
				if title != "" {
					updates["title"] = title
				}
			}
		}
	}
	_ = s.db.Model(&ChatSession{}).
		Where("id = ? AND user_id = ?", sessionID, userID).
		Updates(updates).Error
	return msg, nil
}

func truncateRunes(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}
