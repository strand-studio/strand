package api

import (
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"strand/internal/messaging"
	"strand/internal/store"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func maskMessagingSecret(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return ""
	}
	if len(v) <= 8 {
		return "***"
	}
	return v[:3] + "***" + v[len(v)-3:]
}

func (s *Service) consoleMessagingSupported(c *gin.Context) {
	writeJSON(c, http.StatusOK, gin.H{
		"platforms": messaging.SupportedPlatforms(),
		"agents":    messaging.Agents(),
	})
}

func (s *Service) consoleMessagingWorkspaceGet(c *gin.Context) {
	st := s.eng.Store()
	if st == nil {
		writeErr(c, http.StatusServiceUnavailable, errStoreUnavailable)
		return
	}
	ws, err := st.Messaging().GetWorkspace(currentUserID(c))
	if err != nil {
		writeErr(c, http.StatusInternalServerError, err)
		return
	}
	writeJSON(c, http.StatusOK, ws)
}

func (s *Service) consoleMessagingWorkspacePut(c *gin.Context) {
	st := s.eng.Store()
	if st == nil {
		writeErr(c, http.StatusServiceUnavailable, errStoreUnavailable)
		return
	}
	var body struct {
		AgentType  string `json:"agent_type"`
		AgentModel string `json:"agent_model"`
		WorkDir    string `json:"work_dir"`
		Language   string `json:"language"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		writeErr(c, http.StatusBadRequest, err)
		return
	}
	userID := currentUserID(c)
	ws := &store.MessagingWorkspace{
		UserID:     userID,
		AgentType:  body.AgentType,
		AgentModel: body.AgentModel,
		WorkDir:    body.WorkDir,
		Language:   body.Language,
	}
	if ws.WorkDir == "" {
		ws.WorkDir = filepath.Join(s.eng.Settings().Data.Dir, "workspace", "messaging")
	}
	if err := st.Messaging().SaveWorkspace(ws); err != nil {
		writeErr(c, http.StatusInternalServerError, err)
		return
	}
	writeJSON(c, http.StatusOK, ws)
}

func channelPublic(ch *store.MessagingChannel) gin.H {
	creds, _ := ch.CredentialMap()
	masked := map[string]string{}
	for k, v := range creds {
		if strings.Contains(strings.ToLower(k), "token") ||
			strings.Contains(strings.ToLower(k), "secret") ||
			strings.Contains(strings.ToLower(k), "password") ||
			strings.Contains(strings.ToLower(k), "key") {
			masked[k] = maskMessagingSecret(v)
		} else if len(v) > 64 {
			masked[k] = maskMessagingSecret(v)
		} else {
			masked[k] = v
		}
	}
	return gin.H{
		"id":             ch.ID,
		"name":           ch.Name,
		"platform_type":  ch.PlatformType,
		"chat_enabled":   ch.ChatEnabled,
		"alerts_enabled": ch.AlertsEnabled,
		"enabled":        ch.Enabled,
		"allow_from":     ch.AllowFrom,
		"bound_peer":     ch.BoundPeer,
		"status":         ch.Status,
		"last_error":     ch.LastError,
		"credentials":    masked,
		"created_at":     ch.CreatedAt,
		"updated_at":     ch.UpdatedAt,
	}
}

func (s *Service) consoleMessagingChannelsList(c *gin.Context) {
	st := s.eng.Store()
	if st == nil {
		writeErr(c, http.StatusServiceUnavailable, errStoreUnavailable)
		return
	}
	userID := currentUserID(c)
	rows, err := st.Messaging().ListChannels(userID)
	if err != nil {
		writeErr(c, http.StatusInternalServerError, err)
		return
	}
	health, _ := st.Resources().HealthMap(userID, store.ResourceKindPlatform)
	out := make([]gin.H, 0, len(rows))
	for _, row := range rows {
		out = append(out, attachHealthFields(channelPublic(row), health[row.ID]))
	}
	writeJSON(c, http.StatusOK, gin.H{"channels": out, "count": len(out)})
}

func (s *Service) consoleMessagingChannelCreate(c *gin.Context) {
	st := s.eng.Store()
	if st == nil {
		writeErr(c, http.StatusServiceUnavailable, errStoreUnavailable)
		return
	}
	var body struct {
		Name          string            `json:"name"`
		PlatformType  string            `json:"platform_type"`
		Credentials   map[string]string `json:"credentials"`
		ChatEnabled   *bool             `json:"chat_enabled"`
		AlertsEnabled *bool             `json:"alerts_enabled"`
		Enabled       *bool             `json:"enabled"`
		AllowFrom     string            `json:"allow_from"`
		BoundPeer     string            `json:"bound_peer"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		writeErr(c, http.StatusBadRequest, err)
		return
	}
	ch := &store.MessagingChannel{
		ID:           uuid.NewString(),
		UserID:       currentUserID(c),
		Name:         body.Name,
		PlatformType: body.PlatformType,
		AllowFrom:    strings.TrimSpace(body.AllowFrom),
		BoundPeer:    strings.TrimSpace(body.BoundPeer),
		ChatEnabled:  true,
		AlertsEnabled: true,
		Enabled:      true,
		Status:       "stopped",
	}
	if body.ChatEnabled != nil {
		ch.ChatEnabled = *body.ChatEnabled
	}
	if body.AlertsEnabled != nil {
		ch.AlertsEnabled = *body.AlertsEnabled
	}
	if body.Enabled != nil {
		ch.Enabled = *body.Enabled
	}
	if err := ch.SetCredentialMap(body.Credentials); err != nil {
		writeErr(c, http.StatusBadRequest, err)
		return
	}
	if err := validateMessagingChannel(ch); err != nil {
		writeErr(c, http.StatusBadRequest, err)
		return
	}
	if err := st.Messaging().CreateChannel(ch); err != nil {
		writeErr(c, http.StatusInternalServerError, err)
		return
	}
	_ = s.eng.ReloadMessaging()
	fresh, _ := st.Messaging().GetChannel(ch.UserID, ch.ID)
	if fresh == nil {
		fresh = ch
	}
	// Probe once so Resources shows bound/OK after create (not an empty dash).
	start := time.Now()
	probeErr := probePlatformChannel(fresh)
	latency := time.Since(start).Milliseconds()
	errMsg := ""
	if probeErr != nil {
		errMsg = probeErr.Error()
	}
	_ = st.Resources().RecordTest(ch.UserID, store.ResourceKindPlatform, fresh.ID, probeErr == nil, latency, errMsg)
	writeJSON(c, http.StatusOK, channelPublic(fresh))
}

func (s *Service) consoleMessagingChannelUpdate(c *gin.Context) {
	st := s.eng.Store()
	if st == nil {
		writeErr(c, http.StatusServiceUnavailable, errStoreUnavailable)
		return
	}
	userID := currentUserID(c)
	id := c.Param("id")
	ch, err := st.Messaging().GetChannel(userID, id)
	if err != nil {
		writeErr(c, http.StatusNotFound, fmt.Errorf("channel not found"))
		return
	}
	var body struct {
		Name          *string           `json:"name"`
		Credentials   map[string]string `json:"credentials"`
		ChatEnabled   *bool             `json:"chat_enabled"`
		AlertsEnabled *bool             `json:"alerts_enabled"`
		Enabled       *bool             `json:"enabled"`
		AllowFrom     *string           `json:"allow_from"`
		BoundPeer     *string           `json:"bound_peer"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		writeErr(c, http.StatusBadRequest, err)
		return
	}
	if body.Name != nil {
		ch.Name = *body.Name
	}
	if body.ChatEnabled != nil {
		ch.ChatEnabled = *body.ChatEnabled
	}
	if body.AlertsEnabled != nil {
		ch.AlertsEnabled = *body.AlertsEnabled
	}
	if body.Enabled != nil {
		ch.Enabled = *body.Enabled
	}
	if body.AllowFrom != nil {
		ch.AllowFrom = strings.TrimSpace(*body.AllowFrom)
	}
	if body.BoundPeer != nil {
		ch.BoundPeer = strings.TrimSpace(*body.BoundPeer)
	}
	if body.Credentials != nil {
		existing, _ := ch.CredentialMap()
		for k, v := range body.Credentials {
			if strings.TrimSpace(v) == "" || strings.Contains(v, "***") {
				continue // keep previous secret when UI sends masked value
			}
			existing[k] = v
		}
		if err := ch.SetCredentialMap(existing); err != nil {
			writeErr(c, http.StatusBadRequest, err)
			return
		}
	}
	if err := validateMessagingChannel(ch); err != nil {
		writeErr(c, http.StatusBadRequest, err)
		return
	}
	if err := st.Messaging().UpdateChannel(ch); err != nil {
		writeErr(c, http.StatusInternalServerError, err)
		return
	}
	_ = s.eng.ReloadMessaging()
	fresh, _ := st.Messaging().GetChannel(userID, id)
	if fresh == nil {
		fresh = ch
	}
	writeJSON(c, http.StatusOK, channelPublic(fresh))
}

func (s *Service) consoleMessagingChannelDelete(c *gin.Context) {
	st := s.eng.Store()
	if st == nil {
		writeErr(c, http.StatusServiceUnavailable, errStoreUnavailable)
		return
	}
	if err := st.Messaging().DeleteChannel(currentUserID(c), c.Param("id")); err != nil {
		writeErr(c, http.StatusInternalServerError, err)
		return
	}
	_ = s.eng.ReloadMessaging()
	writeJSON(c, http.StatusOK, gin.H{"ok": true})
}

func (s *Service) consoleMessagingReload(c *gin.Context) {
	if err := s.eng.ReloadMessaging(); err != nil {
		writeErr(c, http.StatusInternalServerError, err)
		return
	}
	writeJSON(c, http.StatusOK, gin.H{"ok": true})
}

func validateMessagingChannel(ch *store.MessagingChannel) error {
	if ch.Name == "" {
		return fmt.Errorf("name required")
	}
	if ch.PlatformType == "" {
		return fmt.Errorf("platform_type required")
	}
	creds, err := ch.CredentialMap()
	if err != nil {
		return err
	}
	for _, schema := range messaging.SupportedPlatforms() {
		if schema.Type != ch.PlatformType {
			continue
		}
		for _, f := range schema.Fields {
			if f.Required && strings.TrimSpace(creds[f.Key]) == "" {
				return fmt.Errorf("%s required", f.Key)
			}
		}
		return nil
	}
	// Unknown but registered types: require at least one credential.
	if len(creds) == 0 {
		return fmt.Errorf("credentials required")
	}
	return nil
}
