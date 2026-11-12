package api

import (
	"context"

	"strand/internal/engine"
)

// API is the service contract behind the HTTP routes.
type API interface {
	Health(ctx context.Context) (Health, error)
	Status(ctx context.Context) (engine.Status, error)
	ListTraders(ctx context.Context) ([]map[string]any, error)
	GetTrader(ctx context.Context, id string) (map[string]any, error)
	StartTrader(ctx context.Context, id string) error
	StopTrader(ctx context.Context, id string) error
	ListAgents(ctx context.Context) ([]string, error)
	ListPlatforms(ctx context.Context) ([]string, error)
}

// Health is GET /api/v1/health.
type Health struct {
	Status  string `json:"status"`
	Version string `json:"version,omitempty"`
}
