package store

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// PlanningConversation is one Trade assistant thread for a user.
type PlanningConversation struct {
	ID         string    `gorm:"primaryKey;size:64" json:"id"`
	UserID     string    `gorm:"index;size:64;not null" json:"user_id"`
	Symbol     string    `gorm:"size:64;default:''" json:"symbol"`
	ExchangeID string    `gorm:"size:64;default:''" json:"exchange_id"`
	ModelID    string    `gorm:"size:128;default:''" json:"model_id"`
	Messages   string    `gorm:"type:text" json:"messages"` // JSON array of {role,content}
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type PlanningConversationStore struct {
	db *gorm.DB
}

func NewPlanningConversationStore(db *gorm.DB) *PlanningConversationStore {
	return &PlanningConversationStore{db: db}
}

func (s *PlanningConversationStore) initTables() error {
	return s.db.AutoMigrate(&PlanningConversation{})
}

func (s *PlanningConversationStore) List(userID string, limit int) ([]PlanningConversation, error) {
	if limit <= 0 || limit > 100 {
		limit = 40
	}
	var rows []PlanningConversation
	err := s.db.Where("user_id = ?", userID).
		Order("updated_at DESC").
		Limit(limit).
		Find(&rows).Error
	return rows, err
}

// Upsert creates or replaces a conversation owned by userID.
func (s *PlanningConversationStore) Upsert(row *PlanningConversation) error {
	if row == nil {
		return fmt.Errorf("conversation required")
	}
	row.UserID = strings.TrimSpace(row.UserID)
	if row.UserID == "" {
		return fmt.Errorf("user_id required")
	}
	row.ID = strings.TrimSpace(row.ID)
	if row.ID == "" {
		row.ID = uuid.NewString()
	}
	now := time.Now().UTC()
	row.UpdatedAt = now
	var existing PlanningConversation
	err := s.db.Where("id = ?", row.ID).First(&existing).Error
	if err == gorm.ErrRecordNotFound {
		row.CreatedAt = now
		return s.db.Create(row).Error
	}
	if err != nil {
		return err
	}
	if existing.UserID != row.UserID {
		return fmt.Errorf("conversation not found")
	}
	return s.db.Model(&existing).Updates(map[string]interface{}{
		"symbol":      row.Symbol,
		"exchange_id": row.ExchangeID,
		"model_id":    row.ModelID,
		"messages":    row.Messages,
		"updated_at":  now,
	}).Error
}
