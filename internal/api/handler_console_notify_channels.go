package api

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func maskSecret(secret string) string {
	secret = strings.TrimSpace(secret)
	if secret == "" {
		return ""
	}
	if len(secret) <= 8 {
		return "***"
	}
	return secret[:4] + "***" + secret[len(secret)-3:]
}

func (s *Service) consoleNotifyChannelGet(c *gin.Context) {
	channelType := strings.ToLower(strings.TrimSpace(c.Param("type")))
	if channelType != "discord" && channelType != "slack" {
		writeJSON(c, http.StatusBadRequest, gin.H{"error": "type must be discord or slack"})
		return
	}
	st := s.eng.Store()
	if st == nil {
		writeJSON(c, http.StatusOK, gin.H{"channel_type": channelType, "configured": false})
		return
	}
	row, err := st.NotifyChannels().Get(channelType)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			writeJSON(c, http.StatusOK, gin.H{"channel_type": channelType, "configured": false})
			return
		}
		writeErr(c, http.StatusInternalServerError, err)
		return
	}
	bound := strings.TrimSpace(row.BoundChanID) != "" || strings.TrimSpace(row.BoundGuildID) != ""
	writeJSON(c, http.StatusOK, gin.H{
		"channel_type":       channelType,
		"configured":         strings.TrimSpace(row.BotToken) != "" || strings.TrimSpace(row.WebhookURL) != "",
		"token_masked":       maskSecret(row.BotToken),
		"webhook_masked":     maskSecret(row.WebhookURL),
		"is_bound":           bound,
		"bound_guild_id":     row.BoundGuildID,
		"bound_channel_id":   row.BoundChanID,
		"username":           row.Username,
		"model_id":           row.ModelID,
	})
}

func (s *Service) consoleNotifyChannelSave(c *gin.Context) {
	channelType := strings.ToLower(strings.TrimSpace(c.Param("type")))
	if channelType != "discord" && channelType != "slack" {
		writeJSON(c, http.StatusBadRequest, gin.H{"error": "type must be discord or slack"})
		return
	}
	st := s.eng.Store()
	if st == nil {
		writeErr(c, http.StatusServiceUnavailable, errStoreUnavailable)
		return
	}
	var body struct {
		BotToken   string `json:"bot_token"`
		WebhookURL string `json:"webhook_url"`
		ModelID    string `json:"model_id"`
		GuildID    string `json:"guild_id"`
		ChannelID  string `json:"channel_id"`
		Username   string `json:"username"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		writeErr(c, http.StatusBadRequest, err)
		return
	}
	if err := st.NotifyChannels().Save(channelType, body.BotToken, body.WebhookURL, body.ModelID); err != nil {
		writeJSON(c, http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if strings.TrimSpace(body.ChannelID) != "" || strings.TrimSpace(body.GuildID) != "" {
		_ = st.NotifyChannels().Bind(channelType, body.GuildID, body.ChannelID, body.Username)
	}
	hint := "Credentials saved. Runtime alerts (pause / safe mode / crash) fan out via webhook or bot+channel_id."
	if channelType == "slack" {
		hint = "Slack saved. Prefer incoming webhook_url for alerts; bot_token+channel_id also works."
	}
	writeJSON(c, http.StatusOK, gin.H{"ok": true, "channel_type": channelType, "hint": hint})
}

func (s *Service) consoleNotifyChannelUnbind(c *gin.Context) {
	channelType := strings.ToLower(strings.TrimSpace(c.Param("type")))
	st := s.eng.Store()
	if st == nil {
		writeErr(c, http.StatusServiceUnavailable, errStoreUnavailable)
		return
	}
	if err := st.NotifyChannels().Unbind(channelType); err != nil {
		writeErr(c, http.StatusInternalServerError, err)
		return
	}
	writeJSON(c, http.StatusOK, gin.H{"ok": true, "is_bound": false})
}

func (s *Service) consoleNotifyChannelClear(c *gin.Context) {
	channelType := strings.ToLower(strings.TrimSpace(c.Param("type")))
	st := s.eng.Store()
	if st == nil {
		writeErr(c, http.StatusServiceUnavailable, errStoreUnavailable)
		return
	}
	if err := st.NotifyChannels().Clear(channelType); err != nil {
		writeErr(c, http.StatusInternalServerError, err)
		return
	}
	writeJSON(c, http.StatusOK, gin.H{"ok": true, "configured": false})
}
