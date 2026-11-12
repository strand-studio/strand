package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"strand/internal/store"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func strategyPayload(st *store.Strategy) gin.H {
	var cfg any
	if st.Config != "" {
		_ = json.Unmarshal([]byte(st.Config), &cfg)
	}
	if cfg == nil {
		cfg = gin.H{}
	}
	return gin.H{
		"id":             st.ID,
		"name":           st.Name,
		"description":    st.Description,
		"is_active":      st.IsActive,
		"is_default":     st.IsDefault,
		"is_public":      st.IsPublic,
		"config_visible": st.ConfigVisible,
		"config":         cfg,
		"created_at":     st.CreatedAt.Format(time.RFC3339),
		"updated_at":     st.UpdatedAt.Format(time.RFC3339),
	}
}

func (s *Service) consoleStrategies(c *gin.Context) {
	st := s.eng.Store()
	if st == nil {
		writeJSON(c, http.StatusOK, gin.H{"strategies": []any{}})
		return
	}
	list, err := st.Strategy().List(currentUserID(c))
	if err != nil {
		writeErr(c, http.StatusInternalServerError, err)
		return
	}
	out := make([]gin.H, 0, len(list))
	for _, item := range list {
		out = append(out, strategyPayload(item))
	}
	writeJSON(c, http.StatusOK, gin.H{"strategies": out})
}

func (s *Service) consoleStrategyGet(c *gin.Context) {
	st := s.eng.Store()
	if st == nil {
		writeErr(c, http.StatusServiceUnavailable, errStoreUnavailable)
		return
	}
	item, err := st.Strategy().Get(currentUserID(c), c.Param("id"))
	if err != nil {
		writeJSON(c, http.StatusNotFound, gin.H{"error": "strategy not found"})
		return
	}
	writeJSON(c, http.StatusOK, strategyPayload(item))
}

func (s *Service) consoleStrategyActive(c *gin.Context) {
	st := s.eng.Store()
	if st == nil {
		writeErr(c, http.StatusServiceUnavailable, errStoreUnavailable)
		return
	}
	item, err := st.Strategy().GetActive(currentUserID(c))
	if err != nil {
		writeJSON(c, http.StatusNotFound, gin.H{"error": "no active strategy"})
		return
	}
	writeJSON(c, http.StatusOK, strategyPayload(item))
}

func (s *Service) consoleStrategyDefaultConfig(c *gin.Context) {
	cfg := store.GetDefaultStrategyConfig("en")
	writeJSON(c, http.StatusOK, cfg)
}

func (s *Service) consoleStrategyCreate(c *gin.Context) {
	st := s.eng.Store()
	if st == nil {
		writeErr(c, http.StatusServiceUnavailable, errStoreUnavailable)
		return
	}
	userID := currentUserID(c)
	var body struct {
		Name        string          `json:"name"`
		Description string          `json:"description"`
		Config      json.RawMessage `json:"config"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		writeErr(c, http.StatusBadRequest, err)
		return
	}
	name := strings.TrimSpace(body.Name)
	if name == "" {
		writeJSON(c, http.StatusBadRequest, gin.H{"error": "name required"})
		return
	}
	cfgJSON := "{}"
	if len(body.Config) > 0 {
		cfgJSON = string(body.Config)
	}
	item := &store.Strategy{
		ID:          uuid.NewString(),
		UserID:      userID,
		Name:        name,
		Description: body.Description,
		Config:      cfgJSON,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}
	if err := st.Strategy().Create(item); err != nil {
		writeErr(c, http.StatusInternalServerError, err)
		return
	}
	writeJSON(c, http.StatusOK, strategyPayload(item))
}

func (s *Service) consoleStrategyUpdate(c *gin.Context) {
	st := s.eng.Store()
	if st == nil {
		writeErr(c, http.StatusServiceUnavailable, errStoreUnavailable)
		return
	}
	userID := currentUserID(c)
	id := c.Param("id")
	item, err := st.Strategy().Get(userID, id)
	if err != nil {
		writeJSON(c, http.StatusNotFound, gin.H{"error": "strategy not found"})
		return
	}
	var body struct {
		Name        *string         `json:"name"`
		Description *string         `json:"description"`
		Config      json.RawMessage `json:"config"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		writeErr(c, http.StatusBadRequest, err)
		return
	}
	if body.Name != nil {
		item.Name = strings.TrimSpace(*body.Name)
	}
	if body.Description != nil {
		item.Description = *body.Description
	}
	if len(body.Config) > 0 {
		item.Config = string(body.Config)
	}
	item.UpdatedAt = time.Now().UTC()
	if err := st.Strategy().Update(item); err != nil {
		writeErr(c, http.StatusInternalServerError, err)
		return
	}
	// Hot-reload running traders bound to this strategy (leverage / risk / coin source).
	appliedLive := 0
	if tm := s.eng.Traders(); tm != nil {
		for _, at := range tm.GetAllTraders() {
			if at == nil || at.GetUserID() != userID || at.GetStrategyID() != id {
				continue
			}
			if err := at.ForceReloadStrategyConfig(); err == nil {
				appliedLive++
			}
		}
	}
	payload := strategyPayload(item)
	payload["applied_live"] = appliedLive
	writeJSON(c, http.StatusOK, payload)
}

func (s *Service) consoleStrategyDelete(c *gin.Context) {
	st := s.eng.Store()
	if st == nil {
		writeErr(c, http.StatusServiceUnavailable, errStoreUnavailable)
		return
	}
	if err := st.Strategy().Delete(currentUserID(c), c.Param("id")); err != nil {
		writeErr(c, http.StatusBadRequest, err)
		return
	}
	writeJSON(c, http.StatusOK, gin.H{"ok": true})
}

func (s *Service) consoleStrategyActivate(c *gin.Context) {
	st := s.eng.Store()
	if st == nil {
		writeErr(c, http.StatusServiceUnavailable, errStoreUnavailable)
		return
	}
	userID := currentUserID(c)
	id := c.Param("id")
	if err := st.Strategy().SetActive(userID, id); err != nil {
		writeErr(c, http.StatusBadRequest, err)
		return
	}
	item, err := st.Strategy().Get(userID, id)
	if err != nil {
		writeErr(c, http.StatusInternalServerError, err)
		return
	}
	writeJSON(c, http.StatusOK, strategyPayload(item))
}

func (s *Service) consoleStrategyDuplicate(c *gin.Context) {
	st := s.eng.Store()
	if st == nil {
		writeErr(c, http.StatusServiceUnavailable, errStoreUnavailable)
		return
	}
	userID := currentUserID(c)
	sourceID := c.Param("id")
	var body struct {
		Name string `json:"name"`
	}
	_ = c.ShouldBindJSON(&body)
	name := strings.TrimSpace(body.Name)
	if name == "" {
		name = "Copy"
	}
	newID := uuid.NewString()
	if err := st.Strategy().Duplicate(userID, sourceID, newID, name); err != nil {
		writeErr(c, http.StatusBadRequest, err)
		return
	}
	item, err := st.Strategy().Get(userID, newID)
	if err != nil {
		writeErr(c, http.StatusInternalServerError, err)
		return
	}
	writeJSON(c, http.StatusOK, strategyPayload(item))
}
