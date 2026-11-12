package store

import (
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
)

// NotifyChannel stores Discord / Slack console bindings (one row per channel type).
type NotifyChannel struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	ChannelType  string    `gorm:"uniqueIndex;size:32;not null" json:"channel_type"` // discord | slack
	BotToken     string    `gorm:"type:text" json:"-"`
	WebhookURL   string    `gorm:"type:text" json:"-"`
	BoundGuildID string    `gorm:"size:128;default:''" json:"bound_guild_id"`
	BoundChanID  string    `gorm:"size:128;default:''" json:"bound_channel_id"`
	Username     string    `gorm:"size:128;default:''" json:"username"`
	ModelID      string    `gorm:"size:128;default:''" json:"model_id"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type NotifyChannelStore struct {
	db *gorm.DB
}

func NewNotifyChannelStore(db *gorm.DB) *NotifyChannelStore {
	return &NotifyChannelStore{db: db}
}

func (s *NotifyChannelStore) initTables() error {
	return s.db.AutoMigrate(&NotifyChannel{})
}

func (s *NotifyChannelStore) Get(channelType string) (*NotifyChannel, error) {
	channelType = strings.ToLower(strings.TrimSpace(channelType))
	var row NotifyChannel
	if err := s.db.Where("channel_type = ?", channelType).First(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (s *NotifyChannelStore) Save(channelType, botToken, webhookURL, modelID string) error {
	channelType = strings.ToLower(strings.TrimSpace(channelType))
	if channelType != "discord" && channelType != "slack" {
		return fmt.Errorf("unsupported channel type")
	}
	botToken = strings.TrimSpace(botToken)
	webhookURL = strings.TrimSpace(webhookURL)
	modelID = strings.TrimSpace(modelID)
	if botToken == "" && webhookURL == "" {
		return fmt.Errorf("bot token or webhook URL required")
	}
	var row NotifyChannel
	err := s.db.Where("channel_type = ?", channelType).First(&row).Error
	now := time.Now().UTC()
	if err == gorm.ErrRecordNotFound {
		row = NotifyChannel{
			ChannelType: channelType,
			BotToken:    botToken,
			WebhookURL:  webhookURL,
			ModelID:     modelID,
			UpdatedAt:   now,
		}
		return s.db.Create(&row).Error
	}
	if err != nil {
		return err
	}
	if botToken != "" {
		row.BotToken = botToken
	}
	if webhookURL != "" {
		row.WebhookURL = webhookURL
	}
	if modelID != "" {
		row.ModelID = modelID
	}
	row.UpdatedAt = now
	return s.db.Save(&row).Error
}

func (s *NotifyChannelStore) Bind(channelType, guildID, channelID, username string) error {
	channelType = strings.ToLower(strings.TrimSpace(channelType))
	return s.db.Model(&NotifyChannel{}).
		Where("channel_type = ?", channelType).
		Updates(map[string]interface{}{
			"bound_guild_id":   strings.TrimSpace(guildID),
			"bound_channel_id": strings.TrimSpace(channelID),
			"username":         strings.TrimSpace(username),
			"updated_at":       time.Now().UTC(),
		}).Error
}

func (s *NotifyChannelStore) Unbind(channelType string) error {
	channelType = strings.ToLower(strings.TrimSpace(channelType))
	return s.db.Model(&NotifyChannel{}).
		Where("channel_type = ?", channelType).
		Updates(map[string]interface{}{
			"bound_guild_id":   "",
			"bound_channel_id": "",
			"username":         "",
			"updated_at":       time.Now().UTC(),
		}).Error
}

func (s *NotifyChannelStore) Clear(channelType string) error {
	channelType = strings.ToLower(strings.TrimSpace(channelType))
	return s.db.Where("channel_type = ?", channelType).Delete(&NotifyChannel{}).Error
}
