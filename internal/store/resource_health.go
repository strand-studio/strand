package store

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	ResourceKindModel     = "model"
	ResourceKindAgent     = "agent"
	ResourceKindPlatform  = "platform"
	ResourceKindExchange  = "exchange"
)

// ResourceHealth tracks the last probe result for a configured resource.
type ResourceHealth struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	UserID       string    `gorm:"uniqueIndex:uid_kind_res;size:64;not null" json:"user_id"`
	Kind         string    `gorm:"uniqueIndex:uid_kind_res;size:32;not null" json:"kind"`
	ResourceID   string    `gorm:"uniqueIndex:uid_kind_res;size:128;not null" json:"resource_id"`
	LastTestOK   bool      `gorm:"not null;default:false" json:"last_test_ok"`
	LastTestAt   *time.Time `json:"last_test_at,omitempty"`
	LastError    string    `gorm:"type:text" json:"last_error,omitempty"`
	LatencyMS    int64     `gorm:"default:0" json:"latency_ms"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (ResourceHealth) TableName() string { return "resource_health" }

// ResourceDefaults is per-user routing for which resources apps should use.
type ResourceDefaults struct {
	UserID             string    `gorm:"primaryKey;size:64" json:"user_id"`
	DefaultModelID     string    `gorm:"size:128;default:''" json:"default_model_id"`
	DefaultAgentID     string    `gorm:"size:128;default:''" json:"default_agent_id"`
	DefaultExchangeID  string    `gorm:"size:128;default:''" json:"default_exchange_id"`
	AlertPlatformIDs   string    `gorm:"type:text" json:"-"` // JSON string array
	UpdatedAt          time.Time `json:"updated_at"`
}

func (ResourceDefaults) TableName() string { return "resource_defaults" }

func (d *ResourceDefaults) AlertIDs() []string {
	if d == nil || strings.TrimSpace(d.AlertPlatformIDs) == "" {
		return []string{}
	}
	var out []string
	if err := json.Unmarshal([]byte(d.AlertPlatformIDs), &out); err != nil || out == nil {
		return []string{}
	}
	return out
}

func (d *ResourceDefaults) SetAlertIDs(ids []string) {
	if d == nil {
		return
	}
	if ids == nil {
		ids = []string{}
	}
	b, _ := json.Marshal(ids)
	d.AlertPlatformIDs = string(b)
}

type ResourceStore struct {
	db *gorm.DB
}

func NewResourceStore(db *gorm.DB) *ResourceStore {
	return &ResourceStore{db: db}
}

func (s *ResourceStore) initTables() error {
	return s.db.AutoMigrate(&ResourceHealth{}, &ResourceDefaults{})
}

func (s *ResourceStore) GetHealth(userID, kind, resourceID string) (*ResourceHealth, error) {
	userID = normalizeUser(userID)
	var row ResourceHealth
	err := s.db.Where("user_id = ? AND kind = ? AND resource_id = ?", userID, kind, resourceID).First(&row).Error
	if err == gorm.ErrRecordNotFound {
		return &ResourceHealth{UserID: userID, Kind: kind, ResourceID: resourceID}, nil
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (s *ResourceStore) ListHealth(userID, kind string) ([]*ResourceHealth, error) {
	userID = normalizeUser(userID)
	q := s.db.Where("user_id = ?", userID)
	if kind != "" {
		q = q.Where("kind = ?", kind)
	}
	var rows []*ResourceHealth
	err := q.Find(&rows).Error
	return rows, err
}

func (s *ResourceStore) RecordTest(userID, kind, resourceID string, ok bool, latencyMS int64, errMsg string) error {
	userID = normalizeUser(userID)
	now := time.Now().UTC()
	row := ResourceHealth{
		UserID:     userID,
		Kind:       kind,
		ResourceID: resourceID,
		LastTestOK: ok,
		LastTestAt: &now,
		LastError:  errMsg,
		LatencyMS:  latencyMS,
		UpdatedAt:  now,
	}
	return s.db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "user_id"}, {Name: "kind"}, {Name: "resource_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"last_test_ok", "last_test_at", "last_error", "latency_ms", "updated_at",
		}),
	}).Create(&row).Error
}

func (s *ResourceStore) GetDefaults(userID string) (*ResourceDefaults, error) {
	userID = normalizeUser(userID)
	var row ResourceDefaults
	err := s.db.Where("user_id = ?", userID).First(&row).Error
	if err == gorm.ErrRecordNotFound {
		return &ResourceDefaults{UserID: userID}, nil
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (s *ResourceStore) SaveDefaults(d *ResourceDefaults) error {
	if d == nil {
		return fmt.Errorf("defaults required")
	}
	d.UserID = normalizeUser(d.UserID)
	d.UpdatedAt = time.Now().UTC()
	var existing ResourceDefaults
	err := s.db.Where("user_id = ?", d.UserID).First(&existing).Error
	if err == gorm.ErrRecordNotFound {
		return s.db.Create(d).Error
	}
	if err != nil {
		return err
	}
	return s.db.Save(d).Error
}

// ListDefaultsWithAlertSelection returns defaults rows that define alert_platform_ids.
func (s *ResourceStore) ListDefaultsWithAlertSelection() ([]*ResourceDefaults, error) {
	var rows []*ResourceDefaults
	err := s.db.Where("alert_platform_ids IS NOT NULL AND alert_platform_ids != '' AND alert_platform_ids != '[]'").
		Find(&rows).Error
	return rows, err
}

func normalizeUser(userID string) string {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return "admin"
	}
	return userID
}

// HealthMap returns kind+id -> health for quick attach.
func (s *ResourceStore) HealthMap(userID, kind string) (map[string]*ResourceHealth, error) {
	rows, err := s.ListHealth(userID, kind)
	if err != nil {
		return nil, err
	}
	out := make(map[string]*ResourceHealth, len(rows))
	for _, r := range rows {
		out[r.ResourceID] = r
	}
	return out, nil
}

func AttachHealth(h *ResourceHealth) map[string]any {
	if h == nil {
		return map[string]any{
			"last_test_ok": false,
			"tested":       false,
		}
	}
	out := map[string]any{
		"last_test_ok": h.LastTestOK,
		"tested":       h.LastTestAt != nil,
		"latency_ms":   h.LatencyMS,
		"last_error":   h.LastError,
	}
	if h.LastTestAt != nil {
		out["last_test_at"] = h.LastTestAt.UTC().Format(time.RFC3339)
	}
	return out
}
