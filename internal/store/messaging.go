package store

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"strand/internal/crypto"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// MessagingWorkspace is the shared Agent settings for chat-enabled channels.
type MessagingWorkspace struct {
	UserID     string    `gorm:"primaryKey;size:64" json:"user_id"`
	AgentType  string    `gorm:"size:64;not null;default:''" json:"agent_type"`
	AgentModel string    `gorm:"size:128;default:''" json:"agent_model"`
	WorkDir    string    `gorm:"size:512;default:''" json:"work_dir"`
	Language   string    `gorm:"size:16;default:'zh'" json:"language"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func (MessagingWorkspace) TableName() string { return "messaging_workspaces" }

// MessagingChannel is one social platform binding (chat and/or alerts).
type MessagingChannel struct {
	ID             string                 `gorm:"primaryKey;size:64" json:"id"`
	UserID         string                 `gorm:"index;size:64;not null" json:"user_id"`
	Name           string                 `gorm:"size:128;not null" json:"name"`
	PlatformType   string                 `gorm:"index;size:64;not null" json:"platform_type"`
	Credentials    crypto.EncryptedString `gorm:"type:text" json:"-"`
	ChatEnabled    bool                   `gorm:"not null;default:true" json:"chat_enabled"`
	AlertsEnabled  bool                   `gorm:"not null;default:true" json:"alerts_enabled"`
	Enabled        bool                   `gorm:"not null;default:true" json:"enabled"`
	AllowFrom      string                 `gorm:"size:512;default:''" json:"allow_from"`
	BoundPeer      string                 `gorm:"size:256;default:''" json:"bound_peer"`
	Status         string                 `gorm:"size:32;default:'stopped'" json:"status"`
	LastError      string                 `gorm:"type:text" json:"last_error,omitempty"`
	CreatedAt      time.Time              `json:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at"`
}

func (MessagingChannel) TableName() string { return "messaging_channels" }

// CredentialMap decrypts and parses platform credentials.
func (c *MessagingChannel) CredentialMap() (map[string]string, error) {
	raw := strings.TrimSpace(string(c.Credentials))
	if raw == "" {
		return map[string]string{}, nil
	}
	out := map[string]string{}
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil, fmt.Errorf("invalid credentials json: %w", err)
	}
	return out, nil
}

func (c *MessagingChannel) SetCredentialMap(m map[string]string) error {
	if m == nil {
		m = map[string]string{}
	}
	b, err := json.Marshal(m)
	if err != nil {
		return err
	}
	c.Credentials = crypto.EncryptedString(b)
	return nil
}

// PlatformOpts builds CreatePlatform options (type excluded).
func (c *MessagingChannel) PlatformOpts() (map[string]any, error) {
	creds, err := c.CredentialMap()
	if err != nil {
		return nil, err
	}
	opts := map[string]any{}
	for k, v := range creds {
		if strings.TrimSpace(v) != "" {
			opts[k] = v
		}
	}
	if c.AllowFrom != "" {
		opts["allow_from"] = c.AllowFrom
	}
	return opts, nil
}

type MessagingStore struct {
	db *gorm.DB
}

func NewMessagingStore(db *gorm.DB) *MessagingStore {
	return &MessagingStore{db: db}
}

func (s *MessagingStore) initTables() error {
	return s.db.AutoMigrate(&MessagingWorkspace{}, &MessagingChannel{})
}

func (s *MessagingStore) GetWorkspace(userID string) (*MessagingWorkspace, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		userID = "admin"
	}
	var row MessagingWorkspace
	err := s.db.Where("user_id = ?", userID).First(&row).Error
	if err == gorm.ErrRecordNotFound {
		return &MessagingWorkspace{
			UserID:   userID,
			Language: "zh",
		}, nil
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (s *MessagingStore) SaveWorkspace(ws *MessagingWorkspace) error {
	if ws == nil {
		return fmt.Errorf("workspace required")
	}
	ws.UserID = strings.TrimSpace(ws.UserID)
	if ws.UserID == "" {
		ws.UserID = "admin"
	}
	ws.AgentType = strings.TrimSpace(ws.AgentType)
	ws.AgentModel = strings.TrimSpace(ws.AgentModel)
	ws.WorkDir = strings.TrimSpace(ws.WorkDir)
	ws.Language = strings.TrimSpace(ws.Language)
	if ws.Language == "" {
		ws.Language = "zh"
	}
	ws.UpdatedAt = time.Now().UTC()
	var existing MessagingWorkspace
	err := s.db.Where("user_id = ?", ws.UserID).First(&existing).Error
	if err == gorm.ErrRecordNotFound {
		return s.db.Create(ws).Error
	}
	if err != nil {
		return err
	}
	return s.db.Save(ws).Error
}

func (s *MessagingStore) ListChannels(userID string) ([]*MessagingChannel, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		userID = "admin"
	}
	var rows []*MessagingChannel
	err := s.db.Where("user_id = ?", userID).Order("created_at asc").Find(&rows).Error
	return rows, err
}

func (s *MessagingStore) ListAllEnabledAlertChannels() ([]*MessagingChannel, error) {
	var rows []*MessagingChannel
	err := s.db.Where("enabled = ? AND alerts_enabled = ?", true, true).Find(&rows).Error
	return rows, err
}

// ListEnabledAlertChannelsByIDs returns enabled alert channels filtered by id list.
// Empty ids means "no selection" (caller should fall back to all).
func (s *MessagingStore) ListEnabledAlertChannelsByIDs(ids []string) ([]*MessagingChannel, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	clean := make([]string, 0, len(ids))
	seen := map[string]struct{}{}
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		clean = append(clean, id)
	}
	if len(clean) == 0 {
		return nil, nil
	}
	var rows []*MessagingChannel
	err := s.db.Where("enabled = ? AND alerts_enabled = ? AND id IN ?", true, true, clean).Find(&rows).Error
	return rows, err
}

func (s *MessagingStore) ListAllEnabledChatChannels() ([]*MessagingChannel, error) {
	var rows []*MessagingChannel
	err := s.db.Where("enabled = ? AND chat_enabled = ?", true, true).Find(&rows).Error
	return rows, err
}

func (s *MessagingStore) GetChannel(userID, id string) (*MessagingChannel, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		userID = "admin"
	}
	var row MessagingChannel
	err := s.db.Where("user_id = ? AND id = ?", userID, id).First(&row).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (s *MessagingStore) CreateChannel(ch *MessagingChannel) error {
	if ch == nil {
		return fmt.Errorf("channel required")
	}
	ch.UserID = strings.TrimSpace(ch.UserID)
	if ch.UserID == "" {
		ch.UserID = "admin"
	}
	ch.Name = strings.TrimSpace(ch.Name)
	ch.PlatformType = strings.ToLower(strings.TrimSpace(ch.PlatformType))
	if ch.Name == "" {
		return fmt.Errorf("name required")
	}
	if ch.PlatformType == "" {
		return fmt.Errorf("platform_type required")
	}
	if ch.ID == "" {
		ch.ID = uuid.NewString()
	}
	now := time.Now().UTC()
	ch.CreatedAt = now
	ch.UpdatedAt = now
	if ch.Status == "" {
		ch.Status = "stopped"
	}
	return s.db.Create(ch).Error
}

func (s *MessagingStore) UpdateChannel(ch *MessagingChannel) error {
	if ch == nil || ch.ID == "" {
		return fmt.Errorf("channel id required")
	}
	ch.Name = strings.TrimSpace(ch.Name)
	ch.PlatformType = strings.ToLower(strings.TrimSpace(ch.PlatformType))
	ch.UpdatedAt = time.Now().UTC()
	return s.db.Save(ch).Error
}

func (s *MessagingStore) DeleteChannel(userID, id string) error {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		userID = "admin"
	}
	return s.db.Where("user_id = ? AND id = ?", userID, id).Delete(&MessagingChannel{}).Error
}

func (s *MessagingStore) UpdateChannelStatus(id, status, lastError string) error {
	return s.db.Model(&MessagingChannel{}).Where("id = ?", id).Updates(map[string]any{
		"status":     status,
		"last_error": lastError,
		"updated_at": time.Now().UTC(),
	}).Error
}
