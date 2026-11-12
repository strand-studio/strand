package store

import (
	"strings"
	"time"

	"gorm.io/gorm"
)

// CompetitionSettings is a single-row fair-play config for the arena.
type CompetitionSettings struct {
	ID                 uint      `gorm:"primaryKey" json:"id"`
	ReferenceInitialUSD float64  `gorm:"not null;default:1000" json:"reference_initial_usd"`
	StartAt            *time.Time `json:"start_at"`
	EndAt              *time.Time `json:"end_at"`
	Scoring            string    `gorm:"size:64;default:pnl_pct" json:"scoring"` // pnl_pct | normalized_initial | window_equity
	UpdatedAt          time.Time `json:"updated_at"`
}

type CompetitionSettingsStore struct {
	db *gorm.DB
}

func NewCompetitionSettingsStore(db *gorm.DB) *CompetitionSettingsStore {
	return &CompetitionSettingsStore{db: db}
}

func (s *CompetitionSettingsStore) initTables() error {
	if err := s.db.AutoMigrate(&CompetitionSettings{}); err != nil {
		return err
	}
	var count int64
	if err := s.db.Model(&CompetitionSettings{}).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return s.db.Create(&CompetitionSettings{
			ID:                  1,
			ReferenceInitialUSD: 1000,
			Scoring:             "pnl_pct",
			UpdatedAt:           time.Now().UTC(),
		}).Error
	}
	return nil
}

func (s *CompetitionSettingsStore) Get() (*CompetitionSettings, error) {
	var row CompetitionSettings
	if err := s.db.First(&row, 1).Error; err != nil {
		return nil, err
	}
	if row.ReferenceInitialUSD <= 0 {
		row.ReferenceInitialUSD = 1000
	}
	if strings.TrimSpace(row.Scoring) == "" {
		row.Scoring = "pnl_pct"
	}
	return &row, nil
}

func (s *CompetitionSettingsStore) Update(refInitial float64, scoring string, startAt, endAt *time.Time) (*CompetitionSettings, error) {
	row, err := s.Get()
	if err != nil {
		return nil, err
	}
	if refInitial > 0 {
		row.ReferenceInitialUSD = refInitial
	}
	scoring = strings.TrimSpace(scoring)
	if scoring == "pnl_pct" || scoring == "normalized_initial" {
		row.Scoring = scoring
	}
	row.StartAt = startAt
	row.EndAt = endAt
	row.UpdatedAt = time.Now().UTC()
	if err := s.db.Save(row).Error; err != nil {
		return nil, err
	}
	return row, nil
}
