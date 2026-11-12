package api

import (
	"context"
	"errors"
	"fmt"

	"strand/internal/core"
	"strand/internal/engine"
	"strand/internal/logger"
)

var errTraderAlreadyRunning = errors.New("trader already running")

type Service struct {
	eng     *engine.Engine
	version string
}

// NewService binds a running engine to the API.
func NewService(eng *engine.Engine, version string) API {
	return &Service{eng: eng, version: version}
}

func (s *Service) Health(context.Context) (Health, error) {
	return Health{Status: "ok", Version: s.version}, nil
}

func (s *Service) Status(context.Context) (engine.Status, error) {
	if s.eng == nil {
		return engine.Status{}, fmt.Errorf("engine not ready")
	}
	return s.eng.Snapshot(), nil
}

func (s *Service) ListTraders(context.Context) ([]map[string]any, error) {
	tm := s.eng.Traders()
	if tm == nil {
		return []map[string]any{}, nil
	}
	all := tm.GetAllTraders()
	out := make([]map[string]any, 0, len(all))
	for _, t := range all {
		out = append(out, t.GetStatus())
	}
	return out, nil
}

func (s *Service) GetTrader(_ context.Context, id string) (map[string]any, error) {
	tm := s.eng.Traders()
	if tm == nil {
		return nil, fmt.Errorf("trading disabled")
	}
	t, err := tm.GetTrader(id)
	if err != nil {
		return nil, err
	}
	return t.GetStatus(), nil
}

func (s *Service) StartTrader(_ context.Context, id string) error {
	tm := s.eng.Traders()
	if tm == nil {
		return fmt.Errorf("trading disabled")
	}
	t, err := tm.GetTrader(id)
	if err != nil {
		return err
	}
	status := t.GetStatus()
	if running, _ := status["is_running"].(bool); running {
		return errTraderAlreadyRunning
	}
	t.ClearRuntimeError()
	go func() {
		if err := t.Run(); err != nil {
			logger.Warnf("trader %s runtime error: %v", id, err)
			t.NotifyCrashed(err)
			s.setTraderRunning(id, false)
		}
	}()
	s.setTraderRunning(id, true)
	return nil
}

func (s *Service) StopTrader(_ context.Context, id string) error {
	tm := s.eng.Traders()
	if tm == nil {
		return fmt.Errorf("trading disabled")
	}
	t, err := tm.GetTrader(id)
	if err != nil {
		return err
	}
	t.Stop()
	s.setTraderRunning(id, false)
	return nil
}

func (s *Service) setTraderRunning(id string, running bool) {
	st := s.eng.Store()
	if st == nil {
		return
	}
	tr, err := st.Trader().GetByID(id)
	if err != nil || tr == nil {
		return
	}
	_ = st.Trader().UpdateStatus(tr.UserID, id, running)
}

func (s *Service) ListAgents(context.Context) ([]string, error) {
	return core.ListRegisteredAgents(), nil
}

func (s *Service) ListPlatforms(context.Context) ([]string, error) {
	return core.ListRegisteredPlatforms(), nil
}
