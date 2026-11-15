package store

import (
	"fmt"

	"strand/internal/planning/live"
)

func (s *Store) initPlanningLiveTables() error {
	if err := s.gdb.AutoMigrate(&live.Run{}, &live.Lease{}); err != nil {
		return fmt.Errorf("failed to migrate planning live tables: %w", err)
	}
	return nil
}
